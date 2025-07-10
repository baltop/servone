package main

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

const testDBConnectionString = "postgresql://smart:smart1234@localhost:5432/strange?sslmode=disable"

func setupTestDB(t *testing.T) *sql.DB {
	if err := InitDB(testDBConnectionString); err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}
	setupDatabase()

	db, err := sql.Open("postgres", testDBConnectionString)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	t.Cleanup(func() {
		_, err := db.Exec(`TRUNCATE client_data, mqtt_messages RESTART IDENTITY`)
		if err != nil {
			t.Logf("Failed to truncate tables: %v", err)
		}
		db.Close()
	})

	return db
}

func Test_saveToDB(t *testing.T) {
	db := setupTestDB(t)

	config := &Config{
		Database: DatabaseConfig{
			ConnectionString: testDBConnectionString,
		},
	}

	t.Run("successful save", func(t *testing.T) {
		url := "/test/save"
		data := map[string]interface{}{"key": "value"}
		params := map[string]string{"param1": "value1"}

		var _ *Config = config
		saveToDB(db, url, data, params, nil) // publisher is nil for this test

		// Verify the data was saved
		var ( // Explicitly declare types for clarity
			id          int
			savedURL    string
			savedData   []byte
			savedParams []byte
			createdAt   int64
		)

		row := db.QueryRow("SELECT id, url, data, parameters, created_at FROM client_data WHERE url = $1", url)
		err := row.Scan(&id, &savedURL, &savedData, &savedParams, &createdAt)
		if err != nil {
			t.Fatalf("Failed to query saved data: %v", err)
		}

		if savedURL != url {
			t.Errorf("savedURL = %s, want %s", savedURL, url)
		}

		var unmarshaledData map[string]interface{}
		if err := json.Unmarshal(savedData, &unmarshaledData); err != nil {
			t.Fatalf("Failed to unmarshal saved data: %v", err)
		}

		// Note: The original saveToDB merges params into data
		expectedData := map[string]interface{}{"key": "value", "param1": "value1"}
		if !jsonEqual(unmarshaledData, expectedData) {
			t.Errorf("savedData = %v, want %v", unmarshaledData, expectedData)
		}

		var unmarshaledParams map[string]string
		if err := json.Unmarshal(savedParams, &unmarshaledParams); err != nil {
			t.Fatalf("Failed to unmarshal saved params: %v", err)
		}

		if !reflectDeepEqual(unmarshaledParams, params) {
			t.Errorf("savedParams = %v, want %v", unmarshaledParams, params)
		}

		if time.Since(time.Unix(0, createdAt)) > 5*time.Second {
			t.Errorf("createdAt is too old: %v", time.Unix(0, createdAt))
		}
	})
}

// jsonEqual compares two maps by marshaling them to JSON and comparing the strings.
// This is a simple way to compare for equality without worrying about key order in maps.
func jsonEqual(a, b map[string]interface{}) bool {
	ja, err := json.Marshal(a)
	if err != nil {
		return false
	}
	jb, err := json.Marshal(b)
	if err != nil {
		return false
	}
	return string(ja) == string(jb)
}

// reflectDeepEqual is needed for comparing maps of string to string
func reflectDeepEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if w, ok := b[k]; !ok || v != w {
			return false
		}
	}
	return true
}
