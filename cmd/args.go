package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/term"
	"gopkg.in/yaml.v2"

	"github.com/cyverse/purgeman/pkg/purgeman"
	log "github.com/sirupsen/logrus"
)

const (
	ChildProcessArgument = "child_process"
)

func inputMissingParams(config *purgeman.Config, stdinClosed bool) error {
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

func processArguments() (*purgeman.Config, error, bool) {
	logger := log.WithFields(log.Fields{
		"package":  "main",
		"function": "processArguments",
	})

	var help bool

	config := purgeman.NewDefaultConfig()

	// Parse parameters
	flag.BoolVar(&help, "h", false, "Print help")
	flag.BoolVar(&config.Foreground, "f", false, "Run in foreground")
	flag.BoolVar(&config.ChildProcess, ChildProcessArgument, false, "")
	flag.StringVar(&config.LogPath, "log", "", "Set log file path")

	flag.Parse()

	if help {
		flag.Usage()
		return nil, nil, true
	}

	if len(config.LogPath) > 0 {
		logFile, err := os.OpenFile(config.LogPath, os.O_WRONLY|os.O_CREATE, 0755)
		if err != nil {
			logger.WithError(err).Error("Could not create log file - %s", config.LogPath)
		} else {
			log.SetOutput(logFile)
		}
	}

	if flag.NArg() == 0 {
		flag.Usage()
		return nil, nil, true
	}

	if flag.NArg() != 1 {
		flag.Usage()
		err := fmt.Errorf("Illegal arguments given, required 1, but received %d (%s)", flag.NArg(), strings.Join(flag.Args(), " "))
		logger.Error(err)
		return nil, err, true
	}

	configFilePath := flag.Arg(0)

	stdinClosed := false
	if configFilePath == "-" {
		// read from stdin
		stdinReader := bufio.NewReader(os.Stdin)
		yamlBytes, err := ioutil.ReadAll(stdinReader)
		if err != nil {
			logger.WithError(err).Error("Could not read STDIN")
			return nil, err, true
		}

		err = yaml.Unmarshal(yamlBytes, &config)
		if err != nil {
			return nil, fmt.Errorf("YAML Unmarshal Error - %v", err), true
		}

		stdinClosed = true
	} else {
		// read config
		configFileAbsPath, err := filepath.Abs(configFilePath)
		if err != nil {
			logger.WithError(err).Errorf("Could not access the local yaml file %s", configFilePath)
			return nil, err, true
		}

		fileinfo, err := os.Stat(configFileAbsPath)
		if err != nil {
			logger.WithError(err).Errorf("local yaml file (%s) error", configFileAbsPath)
			return nil, err, true
		}

		if fileinfo.IsDir() {
			logger.WithError(err).Errorf("local yaml file (%s) is not a file", configFileAbsPath)
			return nil, fmt.Errorf("local yaml file (%s) is not a file", configFileAbsPath), true
		}

		yamlBytes, err := ioutil.ReadFile(configFileAbsPath)
		if err != nil {
			logger.WithError(err).Errorf("Could not read the local yaml file %s", configFileAbsPath)
			return nil, err, true
		}

		err = yaml.Unmarshal(yamlBytes, &config)
		if err != nil {
			return nil, fmt.Errorf("YAML Unmarshal Error - %v", err), true
		}
	}

	err := inputMissingParams(config, stdinClosed)
	if err != nil {
		logger.WithError(err).Error("Could not input missing parameters")
		return nil, err, true
	}

	return config, nil, false
}
