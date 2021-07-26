package purgeman

import (
	"encoding/json"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

// IRODSMessageQueueConfig is a configuration object for iRODS message queue
type IRODSMessageQueueConfig struct {
	Username string
	Password string
	Host     string
	Port     int
	VHost    string
	Exchange string // can be empty
	Queue    string // can be empty
}

// IRODSMessageQueueConnection is a connection object for iRODS message queue
type IRODSMessageQueueConnection struct {
	Config         *IRODSMessageQueueConfig
	AMQPConnection *amqp.Connection
	AMQPChannel    *amqp.Channel
	StartMonitor   bool
}

// FSEventHandler is a handler for file system events
type FSEventHandler func(eventtype string, path string, uuid string)

func makeAMQPURL(config *IRODSMessageQueueConfig) string {
	return fmt.Sprintf("amqp://%s:%s@%s:%d/%s", config.Username, config.Password, config.Host, config.Port, config.VHost)
}

// NewIRODSMessageQueue creates a new message queue conneciton
func ConnectIRODSMessageQueue(config *IRODSMessageQueueConfig) (*IRODSMessageQueueConnection, error) {
	logger := log.WithFields(log.Fields{
		"package":  "purgeman",
		"function": "ConnectIRODSMessageQueue",
	})

	amqpURL := makeAMQPURL(config)
	logger.Infof("Connecting to %s:%d", config.Host, config.Port)
	messageQueueConn, err := amqp.Dial(amqpURL)
	if err != nil {
		logger.WithError(err).Errorf("Could not connect to %s:%d", config.Host, config.Port)
		return nil, err
	}

	messageQueueChan, err := messageQueueConn.Channel()
	if err != nil {
		logger.WithError(err).Errorf("Could not open a channel")
		return nil, err
	}

	return &IRODSMessageQueueConnection{
		Config:         config,
		AMQPConnection: messageQueueConn,
		AMQPChannel:    messageQueueChan,
		StartMonitor:   true,
	}, nil
}

// MonitorFSChanges monitors iRODS changes
func (conn *IRODSMessageQueueConnection) MonitorFSChanges(handler FSEventHandler) error {
	logger := log.WithFields(log.Fields{
		"package":  "purgeman",
		"function": "IRODSMessageQueueConnection.MonitorFSChanges",
	})

	if len(conn.Config.Queue) == 0 && len(conn.Config.Exchange) > 0 {
		// create a queue
		// auto generate name
		queue, err := conn.AMQPChannel.QueueDeclare("", false, true, false, false, amqp.Table{})
		if err != nil {
			logger.WithError(err).Errorf("Could not declare a queue")
			return err
		}

		err = conn.AMQPChannel.QueueBind(queue.Name, "#", conn.Config.Exchange, false, amqp.Table{})
		if err != nil {
			logger.WithError(err).Errorf("Could not bind the queue")
			return err
		}

		conn.Config.Queue = queue.Name
	} else if len(conn.Config.Queue) == 0 && len(conn.Config.Exchange) == 0 {
		return fmt.Errorf("no queue or exchange given")
	}

	for conn.StartMonitor {
		msgs, err := conn.AMQPChannel.Consume(
			conn.Config.Queue, // queue
			"",                // consumer
			true,              // autoAck
			false,             // exclusive
			false,             // noLocal
			false,             // noWait
			nil,               // args
		)

		if err != nil {
			logger.WithError(err).Error("Failed to register an RabbitMQ consumer")
			return err
		}

		for msg := range msgs {
			// filter file system events
			if conn.acceptFSEvents(msg) {
				go conn.handleFSEvents(msg, handler)
			}
		}
	}
	return nil
}

// Destroy destroys the purgeman service
func (conn *IRODSMessageQueueConnection) Disconnect() {
	conn.StartMonitor = false

	if conn.AMQPChannel != nil {
		conn.AMQPChannel.Close()
		conn.AMQPChannel = nil
	}

	if conn.AMQPConnection != nil {
		if !conn.AMQPConnection.IsClosed() {
			conn.AMQPConnection.Close()
		}

		conn.AMQPConnection = nil
	}
}

func (conn *IRODSMessageQueueConnection) acceptFSEvents(msg amqp.Delivery) bool {
	switch msg.RoutingKey {
	case "data-object.add", "data-object.mod", "data-object.mv", "data-object.rm":
		return true
	case "collection.add", "collection.mv", "collection.rm":
		return true
	default:
		return false
	}
}

func (conn *IRODSMessageQueueConnection) handleFSEvents(msg amqp.Delivery, handler FSEventHandler) {
	logger := log.WithFields(log.Fields{
		"package":  "purgeman",
		"function": "IRODSMessageQueueConnection.handleFSEvents",
	})

	if strings.Contains(string(msg.Body), "\r") {
		logger.Error("Body with return in it: %s\n", string(msg.Body))
		return
	}

	body := map[string]interface{}{}
	err := json.Unmarshal(msg.Body, &body)
	if err != nil {
		logger.WithError(err).Errorf("Failed to parse message body - %s : %v", msg.RoutingKey, string(msg.Body))
		return
	}

	switch msg.RoutingKey {
	case "data-object.add", "data-object.rm":
		handler(msg.RoutingKey, body["path"].(string), body["entity"].(string))
	case "data-object.mv":
		handler(msg.RoutingKey, body["old-path"].(string), body["entity"].(string))
		handler(msg.RoutingKey, body["new-path"].(string), body["entity"].(string))
	case "collection.add", "collection.rm":
		handler(msg.RoutingKey, body["path"].(string), body["entity"].(string))
	case "collection.mv":
		handler(msg.RoutingKey, body["old-path"].(string), body["entity"].(string))
		handler(msg.RoutingKey, body["new-path"].(string), body["entity"].(string))
	case "data-object.mod":
		// does not have path
		handler(msg.RoutingKey, "", body["entity"].(string))
	default:
		return
	}
}
