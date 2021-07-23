package purgeman

import (
	"net/http"
	"strings"

	irodsfs_client "github.com/cyverse/go-irodsclient/fs"
	irodsfs_clienttype "github.com/cyverse/go-irodsclient/irods/types"
	log "github.com/sirupsen/logrus"
)

// PurgemanService is a service object
type PurgemanService struct {
	Config                 *Config
	IRODSClient            *irodsfs_client.FileSystem
	MessageQueueConnection *IRODSMessageQueueConnection
}

// NewPurgeman creates a new purgeman service
func NewPurgeman(config *Config) (*PurgemanService, error) {
	return &PurgemanService{
		Config: config,
	}, nil
}

func (svc *PurgemanService) Start() error {
	logger := log.WithFields(log.Fields{
		"package":  "purgeman",
		"function": "PurgemanService.Start",
	})

	logger.Info("Starting the purgeman service")

	logger.Info("Connecting to iRODS")
	iRODSAccount, err := irodsfs_clienttype.CreateIRODSAccount(svc.Config.IRODSHost, svc.Config.IRODSPort, svc.Config.IRODSUsername, svc.Config.IRODSZone, irodsfs_clienttype.AuthSchemeNative, svc.Config.IRODSPassword, "")
	if err != nil {
		logger.WithError(err).Error("Failed to create an iRODSAccount")
		return err
	}

	// connect to iRODS
	fsclient, err := irodsfs_client.NewFileSystemWithDefault(iRODSAccount, "purgeman")
	if err != nil {
		log.WithError(err).Errorf("Error connecting to iRODS")
		return err
	}

	svc.IRODSClient = fsclient

	// connect to AMQP
	mqConfig := IRODSMessageQueueConfig{
		Username: svc.Config.AMQPUsername,
		Password: svc.Config.AMQPPassword,
		Host:     svc.Config.AMQPHost,
		Port:     svc.Config.AMQPPort,
		VHost:    svc.Config.AMQPVHost,
		Queue:    svc.Config.AMQPQueue,
	}

	logger.Info("Connecting to iRODS Message Queue")
	mqConn, err := ConnectIRODSMessageQueue(&mqConfig)
	if err != nil {
		logger.WithError(err).Error("Failed to connect to an iRODS Message Queue")
		defer fsclient.Release()
		return err
	}

	svc.MessageQueueConnection = mqConn

	// should not return
	err = mqConn.MonitorFSChanges(svc.fsEventHandler)
	if err != nil {
		logger.Error(err)
		defer mqConn.Disconnect()
		defer fsclient.Release()
		return err
	}

	return nil
}

// Destroy destroys the purgeman service
func (svc *PurgemanService) Destroy() {
	logger := log.WithFields(log.Fields{
		"package":  "purgeman",
		"function": "PurgemanService.Destroy",
	})

	logger.Info("Destroying the purgeman service")

	if svc.IRODSClient != nil {
		svc.IRODSClient.Release()
		svc.IRODSClient = nil
	}

	if svc.MessageQueueConnection != nil {
		svc.MessageQueueConnection.Disconnect()
		svc.MessageQueueConnection = nil
	}
}

// fetchIRODSPath returns path from uuid
func (svc *PurgemanService) fetchIRODSPath(uuid string) string {
	entries, err := svc.IRODSClient.SearchByMeta("ipc_UUID", uuid)
	if err == nil {
		// only one entry must be found
		if len(entries) == 1 {
			// return full path of the data object or the collection
			return entries[0].Path
		}
	}

	// if we couldn't find, return empty string
	return ""
}

// fsEventHandler handles a fs event
func (svc *PurgemanService) fsEventHandler(eventtype string, path string, uuid string) {
	logger := log.WithFields(log.Fields{
		"package":  "purgeman",
		"function": "PurgemanService.fsEventHandler",
	})

	iRODSPath := path
	if len(path) == 0 && len(uuid) > 0 {
		// conv uuid to path
		iRODSPath = svc.fetchIRODSPath(uuid)
	}

	logger.Infof("Reveiced a %s event on file %s", eventtype, iRODSPath)
	svc.purgeCache(iRODSPath)
}

// purgeCache purges cache
func (svc *PurgemanService) purgeCache(path string) {
	logger := log.WithFields(log.Fields{
		"package":  "purgeman",
		"function": "PurgemanService.purgeCache",
	})

	// purge cache on the path
	logger.Info("Purging a cache for %s", path)

	urlPrefix := svc.Config.VarnishURLPrefix
	if !strings.HasSuffix(urlPrefix, "/") {
		urlPrefix = svc.Config.VarnishURLPrefix + "/"
	}

	requestURL := urlPrefix + path

	req, err := http.NewRequest("PURGE", requestURL, nil)
	if err != nil {
		logger.WithError(err).Errorf("Failed to create a PURGE request for url %s", requestURL)
		return
	}

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.WithError(err).Errorf("Failed to make a PURGE request to url %s", requestURL)
		return
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		logger.WithError(err).Errorf("Unexpected response for a PURGE request to url %s - %s", requestURL, response.Status)
		return
	}
}
