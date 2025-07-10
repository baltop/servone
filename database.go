package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

var dbPool *sql.DB

func InitDB(connStr string) error {
	var err error
	dbPool, err = sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	dbPool.SetMaxOpenConns(25)
	dbPool.SetMaxIdleConns(25)
	dbPool.SetConnMaxLifetime(5 * time.Minute)

	if err = dbPool.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	fmt.Println("Database connection pool established.")
	return nil
}

func setupDatabase() {
	createClientDataTableSQL := `
	CREATE TABLE IF NOT EXISTS client_data (
		id SERIAL PRIMARY KEY,
		url TEXT NOT NULL,
		data JSONB,
		parameters JSONB,
		created_at BIGINT
	);`

	_, err := dbPool.Exec(createClientDataTableSQL)
	if err != nil {
		log.Fatalf("Failed to create 'client_data' table: %v", err)
	}
	fmt.Println("Table 'client_data' created successfully or already exists.")

	createMQTTMessagesTableSQL := `
	CREATE TABLE IF NOT EXISTS mqtt_messages (
		id SERIAL PRIMARY KEY,
		topic TEXT NOT NULL,
		payload TEXT,
		created_at BIGINT
	);`

	_, err = dbPool.Exec(createMQTTMessagesTableSQL)
	if err != nil {
		log.Fatalf("Failed to create 'mqtt_messages' table: %v", err)
	}
	fmt.Println("Table 'mqtt_messages' created successfully or already exists.")
}

func saveToDB(config *Config, url string, data map[string]interface{}, params map[string]string, publisher *KafkaPublisher) {
	// Merge data and params, prioritizing existing keys in data
	mergedData := make(map[string]interface{})
	for k, v := range data {
		mergedData[k] = v
	}
	for k, v := range params {
		if _, ok := mergedData[k]; !ok {
			mergedData[k] = v
		}
	}

	mergeDataJSON, err := json.Marshal(mergedData)
	if err != nil {
		log.Printf("Failed to marshal data to JSON: %v", err)
		return
	}

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		log.Printf("Failed to marshal params to JSON: %v", err)
		return
	}

	insertSQL := `
	INSERT INTO client_data (url, data, parameters, created_at)
	VALUES ($1, $2, $3, $4);`

	kafkaData := map[string]interface{}{
		"data":     mergedData,
		"params":   params,
		"url":      url,
		"received": time.Now().UnixNano(),
	}

	_, err = dbPool.Exec(insertSQL, url, mergeDataJSON, paramsJSON, time.Now().UnixNano())
	if err != nil {
		log.Printf("Failed to insert data into database: %v", err)
	} else {
		if publisher != nil {
			publisher.Publish(url, kafkaData)
		}
	}
}

func (c *Config) SaveMQTTMessage(topic string, payload string, receivedTime int64) {
	insertSQL := `
	INSERT INTO mqtt_messages (topic, payload, created_at)
	VALUES ($1, $2, $3);`

	_, err := dbPool.Exec(insertSQL, topic, payload, receivedTime)
	if err != nil {
		log.Printf("Failed to insert data into database: %v", err)
	}
}