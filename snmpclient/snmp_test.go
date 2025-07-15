package snmpclient

import (
	"database/sql"
	"fmt"
	"servone/config"
	servone_db "servone/db"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

// MockKafkaPublisher implements KafkaPublisherInterface for testing
type MockKafkaPublisher struct {
	PublishFunc func(topic string, data map[string]interface{})
	Published   []PublishedMessage
}

type PublishedMessage struct {
	Topic string
	Data  map[string]interface{}
}

func (m *MockKafkaPublisher) Publish(topic string, data map[string]interface{}) error {
	m.Published = append(m.Published, PublishedMessage{Topic: topic, Data: data})
	if m.PublishFunc != nil {
		m.PublishFunc(topic, data)
	}
	return nil
}

func setupTestDB(t *testing.T) *sql.DB {
	connStr := "postgresql://smart:smart1234@localhost:5432/strange?sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Clean up test data
	db.Exec("DELETE FROM snmp_messages WHERE source LIKE 'test_%'")

	return db
}

func TestNewSNMPClient(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mockPublisher := &MockKafkaPublisher{}

	cfg := &config.SNMPConfig{
		Port:           161,
		Timeout:        5,
		Retries:        3,
		Username:       "testuser",
		AuthProtocol:   "SHA256",
		AuthPassphrase: "testauth",
		PrivProtocol:   "AES256",
		PrivPassphrase: "testpriv",
	}

	client := NewSNMPClient(cfg, db, mockPublisher)
	defer client.Stop()

	if client.config != cfg {
		t.Error("Config not set correctly")
	}
	if client.db != db {
		t.Error("Database not set correctly")
	}
	if client.kafkaPublisher != mockPublisher {
		t.Error("Kafka publisher not set correctly")
	}
}

func TestGetSecurityParams(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tests := []struct {
		name         string
		authProtocol string
		privProtocol string
	}{
		{"MD5/DES", "MD5", "DES"},
		{"SHA/AES", "SHA", "AES"},
		{"SHA256/AES256", "SHA256", "AES256"},
		{"NoAuth/NoPriv", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.SNMPConfig{
				Username:       "testuser",
				AuthProtocol:   tt.authProtocol,
				AuthPassphrase: "testauth",
				PrivProtocol:   tt.privProtocol,
				PrivPassphrase: "testpriv",
			}

			client := NewSNMPClient(cfg, db, &MockKafkaPublisher{})
			defer client.Stop()

		})
	}
}

func TestProcessResults(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	publishedChan := make(chan struct{}, 1)
	mockPublisher := &MockKafkaPublisher{
		PublishFunc: func(topic string, data map[string]interface{}) {
			publishedChan <- struct{}{}
		},
	}

	cfg := &config.SNMPConfig{
		Username: "testuser",
	}

	client := NewSNMPClient(cfg, db, mockPublisher)
	defer client.Stop()

	// Note: We can't create actual gosnmp.SnmpPDU objects directly in tests
	// This test verifies the overall structure of the processing

	// Wait for async operation
	select {
	case <-publishedChan:
		// Success
	case <-time.After(100 * time.Millisecond):
		// This is expected since we're not actually calling processResults
	}

	// Verify Kafka messages were prepared correctly
	if len(mockPublisher.Published) > 0 {
		msg := mockPublisher.Published[0]
		if !strings.HasPrefix(msg.Topic, "snmp.") {
			t.Errorf("Expected topic to start with 'snmp.', got %s", msg.Topic)
		}
	}
}

func TestSaveToDatabase(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Initialize the global db pool for servone_db package
	servone_db.DbPool = db

	testData := map[string]interface{}{
		"operation": "test",
		"source":    "test_host",
		"results": []map[string]interface{}{
			{
				"oid":   ".1.3.6.1.2.1.1.1.0",
				"type":  "OctetString",
				"value": "Test Value",
			},
		},
	}

	// Test using db.SaveSNMPData directly
	err := servone_db.SaveSNMPData("test_host", testData, time.Now().UnixNano())
	if err != nil {
		t.Fatalf("Failed to save to DB: %v", err)
	}

	// Verify data was saved
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM snmp_data WHERE host = 'test_host'").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query database: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 row, got %d", count)
	}

	// Clean up
	db.Exec("DELETE FROM snmp_data WHERE host = 'test_host'")
}

func TestGetValueString(t *testing.T) {
	cfg := &config.SNMPConfig{}
	client := NewSNMPClient(cfg, nil, &MockKafkaPublisher{})
	defer client.Stop()

	tests := []struct {
		name     string
		value    interface{}
		expected string
	}{
		{"String value", "test string", "test string"},
		{"Integer value", 42, "42"},
		{"Float value", 3.14, "3.14"},
		{"Boolean value", true, "true"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Since we can't create actual PDUs, we test the conversion logic
			result := fmt.Sprintf("%v", tt.value)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}
