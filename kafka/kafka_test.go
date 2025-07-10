package kafka

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

const (
	testKafkaBroker    = "localhost:9092"
	invalidKafkaBroker = "localhost:9999"
	testTopic          = "test-topic-publish"
)

func TestNewKafkaPublisher(t *testing.T) {
	t.Run("successful connection", func(t *testing.T) {
		publisher, err := NewKafkaPublisher([]string{testKafkaBroker})
		if err != nil {
			t.Fatalf("NewKafkaPublisher() error = %v, wantErr nil", err)
		}
		if publisher == nil {
			t.Fatal("NewKafkaPublisher() publisher = nil, want non-nil")
		}
		publisher.Close()
	})

	t.Run("failed connection", func(t *testing.T) {
		// This test may take a few seconds to time out
		publisher, err := NewKafkaPublisher([]string{invalidKafkaBroker})
		if err != nil {
			t.Fatalf("NewKafkaPublisher() error = %v, wantErr nil", err)
		}
		if publisher == nil {
			t.Fatal("NewKafkaPublisher() publisher = nil, want non-nil")
		}
		defer publisher.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := publisher.client.Ping(ctx); err == nil {
			t.Error("publisher.client.Ping() error = nil, wantErr not nil")
		}
	})
}

func TestKafkaPublisher_Publish(t *testing.T) {
	// Setup Publisher
	publisher, err := NewKafkaPublisher([]string{testKafkaBroker})
	if err != nil {
		t.Fatalf("Failed to create publisher for test: %v", err)
	}
	defer publisher.Close()

	// Setup Consumer
	opts := []kgo.Opt{
		kgo.SeedBrokers(testKafkaBroker),
		kgo.ConsumerGroup("test-group"),
		kgo.ConsumeTopics(testTopic),
		kgo.DisableAutoCommit(),
	}
	consumer, err := kgo.NewClient(opts...)
	if err != nil {
		t.Fatalf("Failed to create consumer for test: %v", err)
	}
	defer consumer.Close()

	t.Run("successful publish", func(t *testing.T) {
		// Data to be published
		testData := map[string]interface{}{
			"message": "hello world",
			"id":      123,
		}

		// Publish the message
		publisher.Publish(testTopic, testData)

		// Consume the message
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		fetches := consumer.PollFetches(ctx)
		if fetches.IsClientClosed() || ctx.Err() != nil {
			t.Fatalf("Consumer closed or context timed out while waiting for message: %v", ctx.Err())
		}

		errs := fetches.Errors()
		if len(errs) > 0 {
			t.Fatalf("Failed to fetch messages: %v", errs)
		}

		var receivedData map[string]interface{}
		var found bool
		fetches.EachRecord(func(r *kgo.Record) {
			if err := json.Unmarshal(r.Value, &receivedData); err != nil {
				t.Logf("Failed to unmarshal received message: %v", err)
				return
			}

			// Simple comparison
			if receivedData["message"] == "hello world" {
				found = true
			}
		})

		if !found {
			t.Fatal("Published message not found in topic")
		}
	})
}

func Test_sanitizeTopic(t *testing.T) {
	type args struct {
		topic string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "topic starting with /",
			args: args{topic: "/my/topic"},
			want: "bz.my.topic",
		},
		{
			name: "topic with multiple /",
			args: args{topic: "my/topic/with/slashes"},
			want: "bz.my.topic.with.slashes",
		},
		{
			name: "topic with special characters",
			args: args{topic: "my_topic-with.special$chars!"},
			want: "bz.my_topic-with.specialchars",
		},
		{
			name: "empty topic",
			args: args{topic: ""},
			want: "bz.",
		},
		{
			name: "topic with no slashes or special characters",
			args: args{topic: "normaltopic"},
			want: "bz.normaltopic",
		},
		{
			name: "topic already prefixed",
			args: args{topic: "bz.prefixed.topic"},
			want: "bz.prefixed.topic",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SanitizeTopic(tt.args.topic); got != tt.want {
				t.Errorf("SanitizeTopic() = %v, want %v", got, tt.want)
			}
		})
	}
}
