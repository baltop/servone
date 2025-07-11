// Package db provides database operations for the servone application.
// It is currently transitioning from global variables to dependency injection.
// 
// Migration path:
// 1. Use NewDatabase() to create a database instance
// 2. Pass the instance to services that need it
// 3. Use instance methods instead of global functions
// 4. Eventually remove global DbPool and global functions
package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"servone/kafka"
	"time"

	_ "github.com/lib/pq"
	"servone/metrics"
)

// Database represents the database connection and operations
type Database struct {
	db *sql.DB
}

// NewDatabase creates a new database connection
func NewDatabase(connStr string) (*Database, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	fmt.Println("Database connection pool established.")
	
	// Start monitoring connection pool metrics
	database := &Database{db: db}
	go database.monitorConnectionPool()
	
	return database, nil
}

// monitorConnectionPool periodically updates connection pool metrics
func (d *Database) monitorConnectionPool() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		stats := d.db.Stats()
		metrics.UpdateDBConnectionMetrics(
			stats.InUse,
			stats.Idle,
			stats.OpenConnections,
		)
	}
}

// Global database instance for backward compatibility
// TODO: Remove this after updating all references
var DbPool *sql.DB

func InitDB(connStr string) error {
	db, err := NewDatabase(connStr)
	if err != nil {
		return err
	}
	DbPool = db.db
	return nil
}

// Close closes the database connection
func (d *Database) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

// CloseDB closes the global database connection pool (for backward compatibility)
func CloseDB() error {
	if DbPool != nil {
		return DbPool.Close()
	}
	return nil
}

// SetupTables creates all necessary database tables
func (d *Database) SetupTables() error {
	if err := d.createClientDataTable(); err != nil {
		return err
	}
	if err := d.createMQTTMessagesTable(); err != nil {
		return err
	}
	if err := d.createCoapMessagesTable(); err != nil {
		return err
	}
	if err := d.createSNMPDataTable(); err != nil {
		return err
	}
	return nil
}

func (d *Database) createClientDataTable() error {
	createSQL := `
	CREATE TABLE IF NOT EXISTS client_data (
		id SERIAL PRIMARY KEY,
		url TEXT NOT NULL,
		data JSONB,
		parameters JSONB,
		created_at BIGINT
	);`
	
	_, err := d.db.Exec(createSQL)
	if err != nil {
		return fmt.Errorf("failed to create 'client_data' table: %w", err)
	}
	fmt.Println("Table 'client_data' created successfully or already exists.")
	return nil
}

func (d *Database) createMQTTMessagesTable() error {
	createSQL := `
	CREATE TABLE IF NOT EXISTS mqtt_messages (
		id SERIAL PRIMARY KEY,
		topic TEXT NOT NULL,
		payload TEXT,
		created_at BIGINT
	);`
	
	_, err := d.db.Exec(createSQL)
	if err != nil {
		return fmt.Errorf("failed to create 'mqtt_messages' table: %w", err)
	}
	fmt.Println("Table 'mqtt_messages' created successfully or already exists.")
	return nil
}

func (d *Database) createCoapMessagesTable() error {
	createSQL := `
	CREATE TABLE IF NOT EXISTS coap_messages (
		id SERIAL PRIMARY KEY,
		path TEXT NOT NULL,
		payload TEXT,
		method TEXT,
		created_at BIGINT
	);`
	
	_, err := d.db.Exec(createSQL)
	if err != nil {
		return fmt.Errorf("failed to create 'coap_messages' table: %w", err)
	}
	fmt.Println("Table 'coap_messages' created successfully or already exists.")
	return nil
}

func (d *Database) createSNMPDataTable() error {
	createSQL := `
	CREATE TABLE IF NOT EXISTS snmp_data (
		id SERIAL PRIMARY KEY,
		host TEXT NOT NULL,
		data JSONB,
		created_at BIGINT
	);`
	
	_, err := d.db.Exec(createSQL)
	if err != nil {
		return fmt.Errorf("failed to create 'snmp_data' table: %w", err)
	}
	fmt.Println("Table 'snmp_data' created successfully or already exists.")
	return nil
}

// SetupDatabase is for backward compatibility
func SetupDatabase() {
	createClientDataTableSQL := `
	CREATE TABLE IF NOT EXISTS client_data (
		id SERIAL PRIMARY KEY,
		url TEXT NOT NULL,
		data JSONB,
		parameters JSONB,
		created_at BIGINT
	);`

	_, err := DbPool.Exec(createClientDataTableSQL)
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

	_, err = DbPool.Exec(createMQTTMessagesTableSQL)
	if err != nil {
		log.Fatalf("Failed to create 'mqtt_messages' table: %v", err)
	}
	fmt.Println("Table 'mqtt_messages' created successfully or already exists.")

	// Create CoAP messages table
	createCoapMessagesTableSQL := `
	CREATE TABLE IF NOT EXISTS coap_messages (
		id SERIAL PRIMARY KEY,
		path TEXT NOT NULL,
		payload TEXT,
		method TEXT,
		created_at BIGINT
	);`

	_, err = DbPool.Exec(createCoapMessagesTableSQL)
	if err != nil {
		log.Fatalf("Failed to create 'coap_messages' table: %v", err)
	}
	fmt.Println("Table 'coap_messages' created successfully or already exists.")

	// Create SNMP data table
	createSNMPDataTableSQL := `
	CREATE TABLE IF NOT EXISTS snmp_data (
		id SERIAL PRIMARY KEY,
		host TEXT NOT NULL,
		data JSONB,
		created_at BIGINT
	);`

	_, err = DbPool.Exec(createSNMPDataTableSQL)
	if err != nil {
		log.Fatalf("Failed to create 'snmp_data' table: %v", err)
	}
	fmt.Println("Table 'snmp_data' created successfully or already exists.")
}

func SaveToDB(url string, data map[string]interface{}, params map[string]string, publisher kafka.KafkaPublisherInterface) {
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

	_, err = DbPool.Exec(insertSQL, url, mergeDataJSON, paramsJSON, time.Now().UnixNano())
	if err != nil {
		log.Printf("Failed to insert data into database: %v", err)
	} else {
		if publisher != nil {
			// Sanitize the topic before publishing
			topic := kafka.SanitizeTopic(url)
			if err := publisher.Publish(topic, kafkaData); err != nil {
				log.Printf("Failed to publish to Kafka: %v", err)
			}
		}
	}
}

// SaveToDBWithError is a version of SaveToDB that returns errors
func SaveToDBWithError(url string, data map[string]interface{}, params map[string]string, publisher kafka.KafkaPublisherInterface) error {
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
		return fmt.Errorf("failed to marshal data to JSON: %w", err)
	}

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("failed to marshal params to JSON: %w", err)
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

	// Use context with timeout for database operations
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = DbPool.ExecContext(ctx, insertSQL, url, mergeDataJSON, paramsJSON, time.Now().UnixNano())
	if err != nil {
		return fmt.Errorf("failed to insert data into database: %w", err)
	}
	
	if publisher != nil {
		// Sanitize the topic before publishing
		topic := kafka.SanitizeTopic(url)
		if err := publisher.Publish(topic, kafkaData); err != nil {
			log.Printf("Failed to publish to Kafka: %v", err)
			// Don't return error as we've already saved to DB
		}
	}
	
	return nil
}

// SaveMQTTMessage saves MQTT message to database (method version)
func (d *Database) SaveMQTTMessage(topic string, payload string, receivedTime int64) error {
	start := time.Now()
	insertSQL := `
	INSERT INTO mqtt_messages (topic, payload, created_at)
	VALUES ($1, $2, $3);`

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := d.db.ExecContext(ctx, insertSQL, topic, payload, receivedTime)
	if err != nil {
		metrics.RecordDBOperation("insert", "mqtt_messages", "failed", time.Since(start).Seconds())
		return fmt.Errorf("failed to insert MQTT message: %w", err)
	}
	metrics.RecordDBOperation("insert", "mqtt_messages", "success", time.Since(start).Seconds())
	return nil
}

// SaveCoapMessage saves CoAP message to database (method version)
func (d *Database) SaveCoapMessage(path string, payload string, method string, receivedTime int64) error {
	insertSQL := `
	INSERT INTO coap_messages (path, payload, method, created_at)
	VALUES ($1, $2, $3, $4);`

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := d.db.ExecContext(ctx, insertSQL, path, payload, method, receivedTime)
	if err != nil {
		return fmt.Errorf("failed to insert CoAP message: %w", err)
	}
	return nil
}

// SaveSNMPData saves SNMP data to database (method version)
func (d *Database) SaveSNMPData(host string, data map[string]interface{}, receivedTime int64) error {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal SNMP data: %w", err)
	}

	insertSQL := `
	INSERT INTO snmp_data (host, data, created_at)
	VALUES ($1, $2, $3);`

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = d.db.ExecContext(ctx, insertSQL, host, dataJSON, receivedTime)
	if err != nil {
		return fmt.Errorf("failed to insert SNMP data: %w", err)
	}
	return nil
}

// SaveToDBWithError saves data to database with error handling (method version)
func (d *Database) SaveToDBWithError(url string, data map[string]interface{}, params map[string]string, publisher kafka.KafkaPublisherInterface) error {
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
		return fmt.Errorf("failed to marshal data to JSON: %w", err)
	}

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("failed to marshal params to JSON: %w", err)
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

	// Use context with timeout for database operations
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = d.db.ExecContext(ctx, insertSQL, url, mergeDataJSON, paramsJSON, time.Now().UnixNano())
	if err != nil {
		return fmt.Errorf("failed to insert data into database: %w", err)
	}
	
	if publisher != nil {
		// Sanitize the topic before publishing
		topic := kafka.SanitizeTopic(url)
		if err := publisher.Publish(topic, kafkaData); err != nil {
			log.Printf("Failed to publish to Kafka: %v", err)
			// Don't return error as we've already saved to DB
		}
	}
	
	return nil
}

// Global function versions for backward compatibility

func SaveMQTTMessage(topic string, payload string, receivedTime int64) error {
	insertSQL := `
	INSERT INTO mqtt_messages (topic, payload, created_at)
	VALUES ($1, $2, $3);`

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := DbPool.ExecContext(ctx, insertSQL, topic, payload, receivedTime)
	if err != nil {
		return fmt.Errorf("failed to insert MQTT message: %w", err)
	}
	return nil
}

// SaveCoapMessage saves CoAP message to database
func SaveCoapMessage(path string, payload string, method string, receivedTime int64) error {
	insertSQL := `
	INSERT INTO coap_messages (path, payload, method, created_at)
	VALUES ($1, $2, $3, $4);`

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := DbPool.ExecContext(ctx, insertSQL, path, payload, method, receivedTime)
	if err != nil {
		return fmt.Errorf("failed to insert CoAP message: %w", err)
	}
	return nil
}

// SaveSNMPData saves SNMP data to database
func SaveSNMPData(host string, data map[string]interface{}, receivedTime int64) error {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal SNMP data: %w", err)
	}

	insertSQL := `
	INSERT INTO snmp_data (host, data, created_at)
	VALUES ($1, $2, $3);`

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = DbPool.ExecContext(ctx, insertSQL, host, dataJSON, receivedTime)
	if err != nil {
		return fmt.Errorf("failed to insert SNMP data: %w", err)
	}
	return nil
}
