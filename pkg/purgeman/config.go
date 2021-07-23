package purgeman

import (
	"fmt"

	yaml "gopkg.in/yaml.v2"
)

const (
	AMQPPortDefault         int    = 5672
	IRODSPortDefault        int    = 1247
	VarnishURLPrefixDefault string = "http://127.0.0.1:6081/"
)

// Config holds the parameters list which can be configured
type Config struct {
	AMQPHost     string `yaml:"amqp_host"`
	AMQPPort     int    `yaml:"amqp_port"`
	AMQPVHost    string `yaml:"amqp_vhost"`
	AMQPQueue    string `yaml:"amqp_queue"`
	AMQPUsername string `yaml:"amqp_username,omitempty"`
	AMQPPassword string `yaml:"amqp_password,omitempty"`

	IRODSHost     string `yaml:"irods_host"`
	IRODSPort     int    `yaml:"irods_port"`
	IRODSUsername string `yaml:"irods_username,omitempty"`
	IRODSPassword string `yaml:"irods_password,omitempty"`
	IRODSZone     string `yaml:"irods_zone"`

	VarnishURLPrefix string `yaml:"varnish_url"`

	LogPath string `yaml:"log_path,omitempty"`

	Foreground   bool `yaml:"foreground,omitempty"`
	ChildProcess bool `yaml:"childprocess,omitempty"`
}

// NewDefaultConfig creates DefaultConfig
func NewDefaultConfig() *Config {
	return &Config{
		AMQPPort: AMQPPortDefault,

		IRODSPort: IRODSPortDefault,

		VarnishURLPrefix: VarnishURLPrefixDefault,

		LogPath: "",

		Foreground:   false,
		ChildProcess: false,
	}
}

// NewConfigFromYAML creates Config from YAML
func NewConfigFromYAML(yamlBytes []byte) (*Config, error) {
	config := Config{
		AMQPPort: AMQPPortDefault,
	}

	err := yaml.Unmarshal(yamlBytes, &config)
	if err != nil {
		return nil, fmt.Errorf("YAML Unmarshal Error - %v", err)
	}

	return &config, nil
}

// Validate validates configuration
func (config *Config) Validate() error {
	if len(config.AMQPHost) == 0 {
		return fmt.Errorf("AMQP hostname must be given")
	}

	if config.AMQPPort <= 0 {
		return fmt.Errorf("AMQP port must be given")
	}

	if len(config.AMQPVHost) == 0 {
		return fmt.Errorf("AMQP vhost must be given")
	}

	if len(config.AMQPQueue) == 0 {
		return fmt.Errorf("AMQP queue must be given")
	}

	if len(config.AMQPUsername) == 0 {
		return fmt.Errorf("AMQP username must be given")
	}

	if len(config.AMQPPassword) == 0 {
		return fmt.Errorf("AMQP password must be given")
	}

	if len(config.IRODSHost) == 0 {
		return fmt.Errorf("IRODS hostname must be given")
	}

	if config.IRODSPort <= 0 {
		return fmt.Errorf("IRODS port must be given")
	}

	if len(config.IRODSUsername) == 0 {
		return fmt.Errorf("IRODS username must be given")
	}

	if len(config.IRODSPassword) == 0 {
		return fmt.Errorf("IRODS password must be given")
	}

	if len(config.IRODSZone) == 0 {
		return fmt.Errorf("IRODS zone must be given")
	}

	if len(config.VarnishURLPrefix) == 0 {
		return fmt.Errorf("Varnish URL Prefix is not given")
	}

	return nil
}
