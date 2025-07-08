
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

func saveToDB(config *Config, url string, data map[string]interface{}, params map[string]string) {
	connStr := config.Database.ConnectionString
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Printf("Failed to connect to database for saving data: %v", err)
		return
	}
	defer db.Close()

	dataJSON, err := json.Marshal(data)
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

	_, err = db.Exec(insertSQL, url, dataJSON, paramsJSON, time.Now().UnixNano())
	if err != nil {
		log.Printf("Failed to insert data into database: %v", err)
	}
}
