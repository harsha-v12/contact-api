package config

import (
	"crypto/tls"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/plain"
)

var KafkaWriter *kafka.Writer

// ConnectKafka initializes the Kafka writer and attempts to auto-create the topic if missing
func ConnectKafka() {
	broker := getEnv("KAFKA_BROKER", "localhost:9092")
	topic := getEnv("KAFKA_TOPIC", "contact-import")

	username := getEnv("KAFKA_USERNAME", "")
	password := getEnv("KAFKA_PASSWORD", "")
	secure := getEnv("KAFKA_SECURE", "false")

	// Pre-create topic if it doesn't exist (helpful for local setup)
	createTopicIfNotExists(broker, topic, username, password, secure)

	transport := &kafka.Transport{
		DialTimeout: 10 * time.Second,
	}
	if username != "" {
		mechanism := plain.Mechanism{
			Username: username,
			Password: password,
		}
		var tlsConfig *tls.Config
		if secure == "true" {
			tlsConfig = &tls.Config{}
		}
		transport.SASL = mechanism
		transport.TLS = tlsConfig
	}

	KafkaWriter = &kafka.Writer{
		Addr:      kafka.TCP(broker),
		Topic:     topic,
		Balancer:  &kafka.LeastBytes{},
		Transport: transport,
	}

	fmt.Printf("Kafka writer initialized for broker %s and topic '%s' (SASL Auth: %t)\n", broker, topic, username != "")
}

// CloseKafka closes the Kafka writer
func CloseKafka() {
	if KafkaWriter != nil {
		if err := KafkaWriter.Close(); err != nil {
			fmt.Printf("Error closing Kafka writer: %v\n", err)
		} else {
			fmt.Println("Kafka writer closed successfully.")
		}
	}
}

// GetKafkaDialer returns a dialer configured with SASL/TLS for QA/Prod environments
func GetKafkaDialer() *kafka.Dialer {
	username := getEnv("KAFKA_USERNAME", "")
	password := getEnv("KAFKA_PASSWORD", "")
	secure := getEnv("KAFKA_SECURE", "false")

	dialer := &kafka.Dialer{
		Timeout:   10 * time.Second,
		DualStack: true,
	}

	if username != "" {
		dialer.SASLMechanism = plain.Mechanism{
			Username: username,
			Password: password,
		}
		if secure == "true" {
			dialer.TLS = &tls.Config{}
		}
	}
	return dialer
}

func createTopicIfNotExists(broker, topic, username, password, secure string) {
	var conn *kafka.Conn
	var err error

	// Dial using custom dialer if Cloud SASL auth is used
	if username != "" {
		dialer := &kafka.Dialer{
			SASLMechanism: plain.Mechanism{
				Username: username,
				Password: password,
			},
			Timeout: 10 * time.Second,
		}
		if secure == "true" {
			dialer.TLS = &tls.Config{}
		}
		conn, err = dialer.Dial("tcp", broker)
	} else {
		conn, err = kafka.Dial("tcp", broker)
	}

	if err != nil {
		fmt.Printf("Warning: failed to connect to Kafka broker for topic check: %v\n", err)
		return
	}	
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		fmt.Printf("Warning: failed to get Kafka controller node: %v\n", err)
		return
	}

	var controllerConn *kafka.Conn
	controllerAddr := net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port))

	if username != "" {
		dialer := &kafka.Dialer{
			SASLMechanism: plain.Mechanism{
				Username: username,
				Password: password,
			},
			Timeout: 10 * time.Second,
		}
		if secure == "true" {
			dialer.TLS = &tls.Config{}
		}
		controllerConn, err = dialer.Dial("tcp", controllerAddr)
	} else {
		controllerConn, err = kafka.Dial("tcp", controllerAddr)
	}

	if err != nil {
		fmt.Printf("Warning: failed to connect to controller node: %v\n", err)
		return
	}
	defer controllerConn.Close()

	topicConfig := kafka.TopicConfig{
		Topic:             topic,
		NumPartitions:     1,
		ReplicationFactor: 1,
	}

	err = controllerConn.CreateTopics(topicConfig)
	if err != nil {
		// If using SaaS Kafka like Upstash, topic may already be created in Web UI
		fmt.Printf("Topic check: Kafka topic '%s' verification complete (or already exists): %v\n", topic, err)
	} else {
		fmt.Printf("Kafka topic '%s' created successfully. ✅\n", topic)
	}
	
}
