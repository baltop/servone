package mqttclient

import (
	"log"
	"servone/config"
	"servone/db"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type MQTTClient struct {
	client         mqtt.Client
	kafkaPublisher KafkaPublisher
}

type KafkaPublisher interface {
	Publish(topic string, data map[string]interface{}) error
}

func NewMQTTClient(broker string, clientID string, kafkaPublisher KafkaPublisher, dbConfig *config.Config) (*MQTTClient, error) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientID(clientID)
	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}

	return &MQTTClient{
		client:         client,
		kafkaPublisher: kafkaPublisher,
	}, nil
}

func (c *MQTTClient) Subscribe(topic string) error {
	token := c.client.Subscribe(topic, 1, func(client mqtt.Client, msg mqtt.Message) {
		log.Printf("Received message on topic: %s and message : %s", msg.Topic(), string(msg.Payload()))
		receivedTime := time.Now().UnixNano()
		
		// Save to DB
		if err := db.SaveMQTTMessage(msg.Topic(), string(msg.Payload()), receivedTime); err != nil {
			log.Printf("Failed to save MQTT message to database: %v", err)
			// Continue to publish to Kafka even if DB save fails
		}

		// Publish to Kafka
		kafkaTopic := "mq." + msg.Topic()
		kafkaPayload := map[string]interface{}{
			"topic":    msg.Topic(),
			"payload":  string(msg.Payload()),
			"received": receivedTime,
		}
		if err := c.kafkaPublisher.Publish(kafkaTopic, kafkaPayload); err != nil {
			log.Printf("Failed to publish MQTT message to Kafka: %v", err)
		}
	})
	
	if token.Wait() && token.Error() != nil {
		return token.Error()
	}
	
	log.Printf("Subscribed to topic: %s", topic)
	return nil
}

var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	log.Printf("Received message: %s from topic: %s", msg.Payload(), msg.Topic())
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	log.Println("Connected to MQTT broker")
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	log.Printf("Connection lost: %v", err)
}

// Disconnect gracefully disconnects from the MQTT broker
func (c *MQTTClient) Disconnect() {
	if c.client != nil && c.client.IsConnected() {
		c.client.Disconnect(250)
		log.Println("Disconnected from MQTT broker")
	}
}
