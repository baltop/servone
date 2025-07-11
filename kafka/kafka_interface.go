package kafka

// KafkaPublisherInterface defines the interface for publishing messages to Kafka.
// This is used for mocking in tests.
type KafkaPublisherInterface interface {
	Publish(topic string, data map[string]interface{}) error
	Close()
}
