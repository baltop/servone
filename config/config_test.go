package config

import (
	"os"
	"reflect"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Test case 1: Successful loading
	t.Run("successful loading", func(t *testing.T) {
		content := `
server:
  port: "8080"
  host: "localhost"
database:
  connection_string: "user=test password=test dbname=test sslmode=disable"
kafka:
  brokers:
    - "localhost:9092"
endpoints:
  - path: "/test"
    method: "GET"
    response:
      status: 200
      body: "Hello, World!"
      headers:
        Content-Type: "text/plain"
`
		tmpfile, err := os.CreateTemp("", "config.*.yaml")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpfile.Name())

		if _, err := tmpfile.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
		if err := tmpfile.Close(); err != nil {
			t.Fatal(err)
		}

		config, err := LoadConfig(tmpfile.Name())
		if err != nil {
			t.Fatalf("LoadConfig() error = %v, wantErr nil", err)
		}

		expectedConfig := &Config{
			Rest:     RestConfig{Port: "8080", Host: "localhost"},
			Database: DatabaseConfig{ConnectionString: "user=test password=test dbname=test sslmode=disable"},
			Kafka:    KafkaConfig{Brokers: []string{"localhost:9092"}},
		}

		if !reflect.DeepEqual(config, expectedConfig) {
			t.Errorf("LoadConfig() = %v, want %v", config, expectedConfig)
		}
	})

	// Test case 2: File not found
	t.Run("file not found", func(t *testing.T) {
		_, err := LoadConfig("nonexistent.yaml")
		if err == nil {
			t.Error("LoadConfig() error = nil, wantErr not nil")
		}
	})

	// Test case 3: Invalid YAML format
	t.Run("invalid yaml format", func(t *testing.T) {
		content := "this is not a valid yaml"
		tmpfile, err := os.CreateTemp("", "config.*.yaml")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpfile.Name())

		if _, err := tmpfile.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
		if err := tmpfile.Close(); err != nil {
			t.Fatal(err)
		}

		_, err = LoadConfig(tmpfile.Name())
		if err == nil {
			t.Error("LoadConfig() error = nil, wantErr not nil")
		}
	})
}
