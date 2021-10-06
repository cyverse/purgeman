package purgeman

import (
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	irodsfs_clientfs "github.com/cyverse/go-irodsclient/fs"
	irodsfs_clienttype "github.com/cyverse/go-irodsclient/irods/types"
	"github.com/cyverse/purgeman/pkg/commons"
	log "github.com/sirupsen/logrus"
)

// PurgemanService is a service object
type PurgemanService struct {
	Config                 *commons.Config
	IRODSClient            *irodsfs_clientfs.FileSystem
	MessageQueueConnection *IRODSMessageQueueConnection
	Terminate              bool
	Mutex                  sync.Mutex
}

// NewPurgeman creates a new purgeman service
func NewPurgeman(config *commons.Config) (*PurgemanService, error) {
	return &PurgemanService{
		Config: config,
	}, nil
}

func (svc *PurgemanService) connectIRODS() error {
	logger := log.WithFields(log.Fields{
		"package":  "purgeman",
		"struct":   "PurgemanService",
		"function": "connectIRODS",
	})

	svc.Mutex.Lock()
	defer svc.Mutex.Unlock()

	if svc.IRODSClient == nil {
		logger.Info("Connecting to iRODS")
		iRODSAccount, err := irodsfs_clienttype.CreateIRODSAccount(svc.Config.IRODSHost, svc.Config.IRODSPort, svc.Config.IRODSUsername, svc.Config.IRODSZone, irodsfs_clienttype.AuthSchemeNative, svc.Config.IRODSPassword, "")
		if err != nil {
			logger.WithError(err).Error("Failed to create an iRODSAccount")
			return err
		}

		// connect to iRODS
		fsclient, err := irodsfs_clientfs.NewFileSystemWithDefault(iRODSAccount, "purgeman")
		if err != nil {
			log.WithError(err).Errorf("Error connecting to iRODS")
			return err
		}

		svc.IRODSClient = fsclient
	}
	return nil
}

func (svc *PurgemanService) connectMessageQueue() error {
	logger := log.WithFields(log.Fields{
		"package":  "purgeman",
		"struct":   "PurgemanService",
		"function": "connectMessageQueue",
	})

	svc.Mutex.Lock()
	defer svc.Mutex.Unlock()

	if svc.MessageQueueConnection == nil {
		logger.Info("Connecting to iRODS Message Queue")

		mqConfig := IRODSMessageQueueConfig{
			Username: svc.Config.AMQPUsername,
			Password: svc.Config.AMQPPassword,
			Host:     svc.Config.AMQPHost,
			Port:     svc.Config.AMQPPort,
			VHost:    svc.Config.AMQPVHost,
			Exchange: svc.Config.AMQPExchange,
		}

		// connect to AMQP
		mqConn, err := ConnectIRODSMessageQueue(&mqConfig)
		if err != nil {
			logger.WithError(err).Error("Failed to connect to an iRODS Message Queue")
			return err
		}

		svc.MessageQueueConnection = mqConn
	}
	return nil
}

func (svc *PurgemanService) Start() error {
	logger := log.WithFields(log.Fields{
		"package":  "purgeman",
		"struct":   "PurgemanService",
		"function": "Start",
	})

	logger.Info("Starting the purgeman service")
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			svc.Mutex.Lock()
			if svc.Terminate {
				svc.Mutex.Unlock()
				return
			}
			svc.Mutex.Unlock()

			err := svc.connectIRODS()
			if err == nil {
				// now connected to iRODS.
				// if disconnected for any reason, iRODS session manage will handle it
				return
			}

			logger.WithError(err).Error("Failed to connect to iRODS, retry after 1 min")
			time.Sleep(1 * time.Minute)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			svc.Mutex.Lock()
			if svc.Terminate {
				svc.Mutex.Unlock()
				return
			}
			svc.Mutex.Unlock()

			err := svc.connectMessageQueue()
			if err == nil {
				// connected
				// will not return until it fails to receive messages
				err = svc.MessageQueueConnection.MonitorFSChanges(svc.fsEventHandler)
				if err != nil {
					logger.Error(err)
				}

				// reconnect?
				svc.Mutex.Lock()
				svc.MessageQueueConnection.Disconnect()
				svc.MessageQueueConnection = nil

				// is the failure due to termination?
				if svc.Terminate {
					svc.Mutex.Unlock()
					return
				}

				svc.Mutex.Unlock()
				// fall below for retry
			}

			logger.WithError(err).Error("Failed to connect to MessageQueue, retry after 1 min")
			time.Sleep(1 * time.Minute)
		}
	}()

	wg.Wait()
	return nil
}

// Destroy destroys the purgeman service
func (svc *PurgemanService) Destroy() {
	logger := log.WithFields(log.Fields{
		"package":  "purgeman",
		"struct":   "PurgemanService",
		"function": "Destroy",
	})

	svc.Mutex.Lock()
	defer svc.Mutex.Unlock()

	if svc.Terminate {
		// already terminated
		return
	}

	svc.Terminate = true

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
	logger := log.WithFields(log.Fields{
		"package":  "purgeman",
		"struct":   "PurgemanService",
		"function": "fetchIRODSPath",
	})

	svc.Mutex.Lock()
	defer svc.Mutex.Unlock()

	if svc.Terminate {
		return ""
	}

	if svc.IRODSClient == nil {
		logger.Errorf("Failed to connect to iRODS")
		return ""
	}

	logger.Infof("fetching iRODS Path from UUID %s", uuid)
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
		"struct":   "PurgemanService",
		"function": "fsEventHandler",
	})

	iRODSPath := path
	if len(path) == 0 && len(uuid) > 0 {
		// conv uuid to path
		iRODSPath = svc.fetchIRODSPath(uuid)
	}

	if len(iRODSPath) > 0 {
		logger.Infof("Reveiced a %s event on file %s", eventtype, iRODSPath)
		svc.purgeCache(iRODSPath)
	} else {
		logger.Infof("Reveiced a %s event on file UUID %s, but could not resolve", eventtype, uuid)
	}
}

// purgeCache purges cache
func (svc *PurgemanService) purgeCache(path string) {
	logger := log.WithFields(log.Fields{
		"package":  "purgeman",
		"struct":   "PurgemanService",
		"function": "purgeCache",
	})

	// purge cache on the path
	logger.Infof("Purging a cache for %s", path)

	wg := sync.WaitGroup{}
	for idx, varnishURL := range svc.Config.VarnishURLPrefixes {
		wg.Add(1)

		f := func(urlPrefix string) {
			defer wg.Done()

			urlPrefix = strings.TrimRight(urlPrefix, "/")
			requestURL := urlPrefix + path

			hostOverride := ""
			if idx < len(svc.Config.VarnishHostsOverride) {
				hostOverride = svc.Config.VarnishHostsOverride[idx]
			}

			host := ""
			if len(hostOverride) > 0 {
				host = hostOverride
			} else {
				u, err := url.Parse(requestURL)
				if err != nil {
					logger.WithError(err).Errorf("Failed to aprse a request '%s'", requestURL)
					return
				}

				host = u.Host
			}

			logger.Infof("Sending a PURGE request to '%s' for host '%s'", requestURL, host)

			req, err := http.NewRequest("PURGE", requestURL, nil)
			if err != nil {
				logger.WithError(err).Errorf("Failed to create a PURGE request to url '%s' for host '%s'", requestURL, host)
				return
			}

			if len(hostOverride) > 0 {
				req.Host = hostOverride
			}

			req.SetBasicAuth(svc.Config.IRODSUsername, svc.Config.IRODSPassword)

			response, err := http.DefaultClient.Do(req)
			if err != nil {
				logger.WithError(err).Errorf("Failed to make a PURGE request to url '%s' for host '%s'", requestURL, host)
				return
			}

			if response.StatusCode < 200 || response.StatusCode >= 300 {
				logger.Errorf("Unexpected response for a PURGE request to url '%s' for host '%s' - %s", requestURL, host, response.Status)
				return
			}
		}

		go f(varnishURL)
	}

	wg.Wait()
}
