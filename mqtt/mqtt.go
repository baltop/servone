package mqtt

import (
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type MQTTClient struct {
	client         mqtt.Client
	kafkaPublisher KafkaPublisher
	dbConfig       DBConfig
}

type KafkaPublisher interface {
	Publish(topic string, data map[string]interface{})
}

type DBConfig interface {
	SaveMQTTMessage(topic string, payload string, receivedTime int64)
}

func NewMQTTClient(broker string, clientID string, kafkaPublisher KafkaPublisher, dbConfig DBConfig) (*MQTTClient, error) {
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
		dbConfig:       dbConfig,
	}, nil
}

func (c *MQTTClient) Subscribe(topic string) {
	token := c.client.Subscribe(topic, 1, func(client mqtt.Client, msg mqtt.Message) {
		log.Printf("Received message on topic: %s", msg.Topic())
		receivedTime := time.Now().UnixNano()
		// Save to DB
		c.dbConfig.SaveMQTTMessage(msg.Topic(), string(msg.Payload()), receivedTime)

		// Publish to Kafka
		kafkaTopic := "mq." + msg.Topic()
		kafkaPayload := map[string]interface{}{
			"topic":    msg.Topic(),
			"payload":  string(msg.Payload()),
			"received": receivedTime,
		}
		c.kafkaPublisher.Publish(kafkaTopic, kafkaPayload)
	})
	token.Wait()
	log.Printf("Subscribed to topic: %s", topic)
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
