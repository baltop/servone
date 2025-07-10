package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

func setupDatabase(config *Config) {
	setupMQTTDatabase(config)

	connStr := config.Database.ConnectionString
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	createTableSQL := `
	CREATE TABLE IF NOT EXISTS client_data (
		id SERIAL PRIMARY KEY,
		url TEXT NOT NULL,
		data JSONB,
		parameters JSONB,
		created_at BIGINT
	);`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}

	fmt.Println("Table 'client_data' created successfully or already exists.")
}

func saveToDB(config *Config, url string, data map[string]interface{}, params map[string]string, publisher *KafkaPublisher) {
	connStr := config.Database.ConnectionString
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Printf("Failed to connect to database for saving data: %v", err)
		return
	}
	defer db.Close()

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

	// 카프카에 보내는 데이터는 { "data": mergedData, "params": params, "url": url, "received": time.Now().UnixNano() } 형태로 구성
	kafkaData := map[string]interface{}{
		"data":     mergedData,
		"params":   params,
		"url":      url,
		"received": time.Now().UnixNano(),
	}

	_, err = db.Exec(insertSQL, url, mergeDataJSON, paramsJSON, time.Now().UnixNano())
	if err != nil {
		log.Printf("Failed to insert data into database: %v", err)
	} else {
		if publisher != nil {
			publisher.Publish(url, kafkaData)
		}
	}
}

func setupMQTTDatabase(config *Config) {
	connStr := config.Database.ConnectionString
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	createTableSQL := `
	CREATE TABLE IF NOT EXISTS mqtt_messages (
		id SERIAL PRIMARY KEY,
		topic TEXT NOT NULL,
		payload TEXT,
		created_at BIGINT
	);`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}

	fmt.Println("Table 'mqtt_messages' created successfully or already exists.")
}

func (c *Config) SaveMQTTMessage(topic string, payload string, receivedTime int64) {
	connStr := c.Database.ConnectionString
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Printf("Failed to connect to database for saving data: %v", err)
		return
	}
	defer db.Close()

	insertSQL := `
	INSERT INTO mqtt_messages (topic, payload, created_at)
	VALUES ($1, $2, $3);`

	_, err = db.Exec(insertSQL, topic, payload, receivedTime)
	if err != nil {
		log.Printf("Failed to insert data into database: %v", err)
	}
}
