package config

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Test case 1: Successful loading
	t.Run("successful loading", func(t *testing.T) {
		content := `
rest:
  port: "8080"
  host: "localhost"
  endpoints: []
coap:
  port: "5683"
  host: "localhost"
  endpoints: []
database:
  connection_string: "user=test password=test dbname=test sslmode=disable"
kafka:
  brokers:
    - "localhost:9092"
mqtt:
  broker: ""
  client_id: ""
snmp:
  port: 0
  timeout: 0
  retries: 0
  username: ""
  auth_protocol: ""
  auth_passphrase: ""
  priv_protocol: ""
  priv_passphrase: ""
  term: 0
  targets: []
snmptrap:
  enabled: false
  host: ""
  port: 0
  username: ""
  auth_protocol: ""
  auth_passphrase: ""
  priv_protocol: ""
  priv_passphrase: ""
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

		// 주요 필드들만 검증
		if config.Rest.Port != "8080" {
			t.Errorf("Expected Rest.Port = 8080, got %s", config.Rest.Port)
		}
		if config.Rest.Host != "localhost" {
			t.Errorf("Expected Rest.Host = localhost, got %s", config.Rest.Host)
		}
		if config.Database.ConnectionString != "user=test password=test dbname=test sslmode=disable" {
			t.Errorf("Expected database connection string mismatch")
		}
		if len(config.Kafka.Brokers) != 1 || config.Kafka.Brokers[0] != "localhost:9092" {
			t.Errorf("Expected Kafka.Brokers = [localhost:9092], got %v", config.Kafka.Brokers)
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
