package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"

	"golang.org/x/term"
	"gopkg.in/natefinch/lumberjack.v2"
	"gopkg.in/yaml.v2"

	"github.com/cyverse/purgeman/pkg/commons"
	log "github.com/sirupsen/logrus"
)

const (
	ChildProcessArgument = "child_process"
)

func inputMissingParams(config *commons.Config, stdinClosed bool) error {
	logger := log.WithFields(log.Fields{
		"package":  "main",
		"function": "inputMissingParams",
	})

	if len(config.AMQPUsername) == 0 {
		if stdinClosed {
			err := fmt.Errorf("AMQP user is not set")
			logger.Error(err)
			return err
		}

		fmt.Print("AMQP Username: ")
		fmt.Scanln(&config.AMQPUsername)
	}

	if len(config.AMQPPassword) == 0 {
		if stdinClosed {
			err := fmt.Errorf("AMQP password is not set")
			logger.Error(err)
			return err
		}

		fmt.Print("AMQP Password: ")
		bytePassword, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Print("\n")
		if err != nil {
			logger.WithError(err).Error("Error occurred while reading AMQP password")
			return err
		}

		config.AMQPPassword = string(bytePassword)
	}

	if len(config.IRODSUsername) == 0 {
		if stdinClosed {
			err := fmt.Errorf("IRODS user is not set")
			logger.Error(err)
			return err
		}

		fmt.Print("IRODS Username: ")
		fmt.Scanln(&config.AMQPUsername)
	}

	if len(config.IRODSPassword) == 0 {
		if stdinClosed {
			err := fmt.Errorf("IRODS password is not set")
			logger.Error(err)
			return err
		}

		fmt.Print("IRODS Password: ")
		bytePassword, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Print("\n")
		if err != nil {
			logger.WithError(err).Error("Error occurred while reading IRODS password")
			return err
		}

		config.IRODSPassword = string(bytePassword)
	}

	return nil
}

func processArguments() (*commons.Config, io.WriteCloser, error, bool) {
	logger := log.WithFields(log.Fields{
		"package":  "main",
		"function": "processArguments",
	})

	var version bool
	var help bool
	var configFilePath string

	config := commons.NewDefaultConfig()

	// Parse parameters
	flag.BoolVar(&version, "version", false, "Print client version information")
	flag.BoolVar(&version, "v", false, "Print client version information (shorthand form)")
	flag.BoolVar(&help, "h", false, "Print help")
	flag.StringVar(&configFilePath, "config", "", "Set Config YAML File")
	flag.BoolVar(&config.Foreground, "f", false, "Run in foreground")
	flag.BoolVar(&config.ChildProcess, ChildProcessArgument, false, "")
	flag.StringVar(&config.LogPath, "log", commons.LogFilePathDefault, "Set log file path")

	flag.Parse()

	if version {
		info, err := commons.GetVersionJSON()
		if err != nil {
			logger.WithError(err).Error("failed to get client version info")
			return nil, nil, err, true
		}

		fmt.Println(info)
		return nil, nil, nil, true
	}

	if help {
		flag.Usage()
		return nil, nil, nil, true
	}

	var logWriter io.WriteCloser
	if config.LogPath == "-" || len(config.LogPath) == 0 {
		log.SetOutput(os.Stderr)
	} else {
		logWriter = getLogWriter(config.LogPath)

		// use multi output - to output to file and stdout
		mw := io.MultiWriter(os.Stderr, logWriter)
		log.SetOutput(mw)
	}

	logger.Infof("Logging to %s", config.LogPath)

	stdinClosed := false
	if len(configFilePath) == 0 {
		// read from Environmental variables
		envConfig, err := commons.NewConfigFromENV()
		if err != nil {
			logger.WithError(err).Error("failed to read Environmental Variables")
			return nil, logWriter, err, true
		}

		envConfig.Foreground = config.Foreground
		// overwrite
		config = envConfig
	} else if configFilePath == "-" {
		// read from stdin
		stdinReader := bufio.NewReader(os.Stdin)
		yamlBytes, err := ioutil.ReadAll(stdinReader)
		if err != nil {
			logger.WithError(err).Error("failed to read STDIN")
			return nil, logWriter, err, true
		}

		err = yaml.Unmarshal(yamlBytes, &config)
		if err != nil {
			return nil, logWriter, fmt.Errorf("failed to unmarshal YAML - %v", err), true
		}

		stdinClosed = true
	} else {
		// read config
		configFileAbsPath, err := filepath.Abs(configFilePath)
		if err != nil {
			logger.WithError(err).Errorf("failed to access the local yaml file %s", configFilePath)
			return nil, logWriter, err, true
		}

		fileinfo, err := os.Stat(configFileAbsPath)
		if err != nil {
			logger.WithError(err).Errorf("failed to access the local yaml file %s", configFileAbsPath)
			return nil, logWriter, err, true
		}

		if fileinfo.IsDir() {
			logger.WithError(err).Errorf("local yaml file %s is not a file", configFileAbsPath)
			return nil, logWriter, fmt.Errorf("local yaml file %s is not a file", configFileAbsPath), true
		}

		yamlBytes, err := ioutil.ReadFile(configFileAbsPath)
		if err != nil {
			logger.WithError(err).Errorf("failed to read the local yaml file %s", configFileAbsPath)
			return nil, logWriter, err, true
		}

		err = yaml.Unmarshal(yamlBytes, &config)
		if err != nil {
			return nil, logWriter, fmt.Errorf("failed to unmarshal YAML - %v", err), true
		}
	}

	err := inputMissingParams(config, stdinClosed)
	if err != nil {
		logger.WithError(err).Error("Could not input missing parameters")
		return nil, logWriter, err, true
	}

	return config, logWriter, nil, false
}

func getLogWriter(logPath string) io.WriteCloser {
	return &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    100, // 100MB
		MaxBackups: 3,
		MaxAge:     30, // 30 days
		Compress:   false,
	}
}
