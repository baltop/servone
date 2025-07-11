package mqttclient

import (
	"servone/config"
	"testing"

	"github.com/stretchr/testify/assert"
)

// MockKafkaPublisher for testing
type MockKafkaPublisher struct {
	PublishedTopic string
	PublishedData  map[string]interface{}
}

func (m *MockKafkaPublisher) Publish(topic string, data map[string]interface{}) error {
	m.PublishedTopic = topic
	m.PublishedData = data
	return nil
}

// MockDBConfig for testing
type MockDBConfig struct {
	SavedTopic     string
	SavedPayload   string
	SavedTimestamp int64
}

func (m *MockDBConfig) SaveMQTTMessage(topic string, payload string, timestamp int64) error {
	m.SavedTopic = topic
	m.SavedPayload = payload
	m.SavedTimestamp = timestamp
	return nil
}

func TestNewMQTTClient(t *testing.T) {
	// This test requires a running MQTT broker.
	// For a true unit test, you would mock the mqtt.Client.
	// For simplicity, we'll assume a local broker is available at tcp://localhost:1883

	borker := "tcp://localhost:1883"
	clientID := "test_client"

	mockKafka := &MockKafkaPublisher{}
	cfg := &config.Config{}

	client, err := NewMQTTClient(borker, clientID, mockKafka, cfg)
	assert.NoError(t, err)
	assert.NotNil(t, client)

	// You can add more assertions here, e.g., check if the client is connected
}

func TestMQTTClient_Subscribe(t *testing.T) {
	// This test also requires a running MQTT broker.
	// You would typically publish a message to the subscribed topic
	// and then assert that the message handler was called and data was processed.

	borker := "tcp://localhost:1883"
	clientID := "test_subscriber"

	mockKafka := &MockKafkaPublisher{}
	cfg := &config.Config{}

	client, err := NewMQTTClient(borker, clientID, mockKafka, cfg)
	assert.NoError(t, err)
	assert.NotNil(t, client)

	topic := "test/topic"
	client.Subscribe(topic)

	// To properly test subscription and message handling, you would need to:
	// 1. Publish a message to "test/topic" from another client.
	// 2. Wait for a short period.
	// 3. Assert that mockKafka.PublishedTopic and mockDB.SavedTopic were updated correctly.
}
