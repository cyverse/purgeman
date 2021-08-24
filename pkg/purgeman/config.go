package purgeman

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
	yaml "gopkg.in/yaml.v2"
)

const (
	AMQPPortDefault         int    = 5672
	IRODSPortDefault        int    = 1247
	VarnishURLPrefixDefault string = "http://127.0.0.1:6081/"
)

// Config holds the parameters list which can be configured
type Config struct {
	AMQPHost     string `envconfig:"PURGEMAN_AMQP_HOST" yaml:"amqp_host"`
	AMQPPort     int    `envconfig:"PURGEMAN_AMQP_PORT" yaml:"amqp_port"`
	AMQPVHost    string `envconfig:"PURGEMAN_AMQP_VHOST" yaml:"amqp_vhost"`
	AMQPExchange string `envconfig:"PURGEMAN_AMQP_EXCHANGE" yaml:"amqp_exchange"`
	AMQPQueue    string `envconfig:"PURGEMAN_AMQP_QUEUE" yaml:"amqp_queue"`
	AMQPUsername string `envconfig:"PURGEMAN_AMQP_USERNAME" yaml:"amqp_username,omitempty"`
	AMQPPassword string `envconfig:"PURGEMAN_AMQP_PASSWORD" yaml:"amqp_password,omitempty"`

	IRODSHost     string `envconfig:"PURGEMAN_IRODS_HOST" yaml:"irods_host"`
	IRODSPort     int    `envconfig:"PURGEMAN_IRODS_PORT" yaml:"irods_port"`
	IRODSUsername string `envconfig:"PURGEMAN_IRODS_USERNAME" yaml:"irods_username,omitempty"`
	IRODSPassword string `envconfig:"PURGEMAN_IRODS_PASSWORD" yaml:"irods_password,omitempty"`
	IRODSZone     string `envconfig:"PURGEMAN_IRODS_ZONE" yaml:"irods_zone"`

	VarnishURLPrefixes []string `envconfig:"PURGEMAN_VARNISH_URLS" yaml:"varnish_urls"`

	LogPath string `envconfig:"PURGEMAN_LOG_PATH" yaml:"log_path,omitempty"`

	Foreground   bool `yaml:"foreground,omitempty"`
	ChildProcess bool `yaml:"childprocess,omitempty"`
}

// NewDefaultConfig creates DefaultConfig
func NewDefaultConfig() *Config {
	return &Config{
		AMQPPort: AMQPPortDefault,

		IRODSPort: IRODSPortDefault,

		VarnishURLPrefixes: []string{
			VarnishURLPrefixDefault,
		},

		LogPath: "",

		Foreground:   false,
		ChildProcess: false,
	}
}

// NewConfigFromENV creates Config from Environmental Variables
func NewConfigFromENV() (*Config, error) {
	config := Config{
		AMQPPort: AMQPPortDefault,

		IRODSPort: IRODSPortDefault,

		VarnishURLPrefixes: []string{
			VarnishURLPrefixDefault,
		},
	}

	err := envconfig.Process("", &config)
	if err != nil {
		return nil, fmt.Errorf("Env Read Error - %v", err)
	}

	return &config, nil
}

// NewConfigFromYAML creates Config from YAML
func NewConfigFromYAML(yamlBytes []byte) (*Config, error) {
	config := Config{
		AMQPPort: AMQPPortDefault,

		IRODSPort: IRODSPortDefault,

		VarnishURLPrefixes: []string{
			VarnishURLPrefixDefault,
		},
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

	if len(config.AMQPExchange) == 0 && len(config.AMQPQueue) == 0 {
		return fmt.Errorf("either AMQP exchange or AMQP Queue must be given")
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

	if len(config.VarnishURLPrefixes) == 0 {
		return fmt.Errorf("Varnish URL Prefix is not given")
	}

	return nil
}
