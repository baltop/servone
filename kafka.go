package main

import (
	"context"
	"encoding/json"
	"log"
	"regexp"
	"strings"

	"github.com/twmb/franz-go/pkg/kgo"
)

type KafkaPublisher struct {
	client *kgo.Client
}

func NewKafkaPublisher(brokers []string) (*KafkaPublisher, error) {
	opts := []kgo.Opt{
		kgo.SeedBrokers(brokers...),
		kgo.AllowAutoTopicCreation(),
	}
	client, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, err
	}
	return &KafkaPublisher{client: client}, nil
}

// sanitizeTopic transforms the topic name as follows:
// - The first '/' is replaced with 'bz.'
// - All subsequent '/' are replaced with '.'
// - All special characters except '_', '-', and '.' are removed
func sanitizeTopic(topic string) string {
	if len(topic) == 0 {
		return topic
	}
	// Replace the first '/' with 'bz.'
	if topic[0] == '/' {
		topic = "bz." + topic[1:]
	}
	// Replace all remaining '/' with '.'
	topic = strings.ReplaceAll(topic, "/", ".")
	// Remove all special characters except '_', '-', and '.'
	re := regexp.MustCompile(`[^a-zA-Z0-9_.-]`)
	topic = re.ReplaceAllString(topic, "")
	return topic
}

func (p *KafkaPublisher) Publish(topic string, data map[string]interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Failed to marshal data for Kafka: %v", err)
		return
	}

	ctx := context.Background()

	record := &kgo.Record{
		Topic: sanitizeTopic(topic),
		Value: jsonData,
	}

	p.client.Produce(ctx, record, func(r *kgo.Record, err error) {
		if err != nil {
			log.Printf("Failed to produce message to Kafka: %v", err)
		}
	})
}

func (p *KafkaPublisher) Close() {
	p.client.Close()
}
