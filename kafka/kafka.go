package kafka

import (
	"context"
	"encoding/json"
	"log"
	"regexp"
	"strings"

	"github.com/twmb/franz-go/pkg/kgo"
)

// 전역 정규표현식 컴파일 (성능 최적화)
var topicSanitizeRegex = regexp.MustCompile("[^a-zA-Z0-9._-]")

// KafkaPublisher is a struct for publishing messages to Kafka.
type KafkaPublisher struct {
	client *kgo.Client
}

// NewKafkaPublisher creates a new Kafka publisher client.
func NewKafkaPublisher(brokers []string) (*KafkaPublisher, error) {
	opts := []kgo.Opt{
		kgo.SeedBrokers(brokers...),
	}

	client, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, err
	}

	return &KafkaPublisher{client: client}, nil
}

// Publish sends a message to a Kafka topic.
func (p *KafkaPublisher) Publish(topic string, data map[string]interface{}) {
	sanitizedTopic := SanitizeTopic(topic)

	payload, err := json.Marshal(data)
	if err != nil {
		log.Printf("Failed to marshal data for Kafka: %v", err)
		return
	}

	record := &kgo.Record{Topic: sanitizedTopic, Value: payload}
	p.client.Produce(context.Background(), record, func(r *kgo.Record, err error) {
		if err != nil {
			log.Printf("Failed to produce message to Kafka: %v", err)
		}
	})
}

// Close closes the Kafka client.
func (p *KafkaPublisher) Close() {
	p.client.Close()
}

// SanitizeTopic prepares a topic name to be compliant with Kafka's naming rules.
func SanitizeTopic(topic string) string {
	topic = strings.TrimPrefix(topic, "/")
	topic = strings.ReplaceAll(topic, "/", ".")

	// Kafka topics can only contain letters, numbers, periods, underscores, and dashes.
	topic = topicSanitizeRegex.ReplaceAllString(topic, "")

	// Add a prefix to avoid potential conflicts with internal topics.
	if !strings.HasPrefix(topic, "bz.") {
		topic = "bz." + topic
	}

	return topic
}
