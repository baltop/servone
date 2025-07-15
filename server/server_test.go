package server

type MockKafkaPublisher struct {
	PublishedTopic string
	PublishedData  map[string]interface{}
}

// Close implements kafka.KafkaPublisherInterface.
func (m *MockKafkaPublisher) Close() {
	panic("unimplemented")
}

func (m *MockKafkaPublisher) Publish(topic string, data map[string]interface{}) error {
	m.PublishedTopic = topic
	m.PublishedData = data
	return nil
}
