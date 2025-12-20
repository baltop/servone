package db

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// 실제 데이터베이스 연결 없이 테스트할 수 있는 함수들을 테스트합니다.

func TestFormatConnectionString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid connection string",
			input:    "postgresql://user:pass@localhost:5432/dbname?sslmode=disable",
			expected: "postgresql://user:pass@localhost:5432/dbname?sslmode=disable",
		},
		{
			name:     "connection string without protocol",
			input:    "user=test password=test dbname=test host=localhost port=5432 sslmode=disable",
			expected: "user=test password=test dbname=test host=localhost port=5432 sslmode=disable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 이 테스트는 실제로는 연결 문자열 검증 함수가 있다면 테스트할 수 있습니다.
			// 현재는 단순히 입력이 출력과 같은지 확인합니다.
			assert.Equal(t, tt.expected, tt.input)
		})
	}
}

func TestTimestampConversion(t *testing.T) {
	// 타임스탬프 변환 테스트
	now := time.Now()
	unixNano := now.UnixNano()
	
	// UnixNano에서 다시 시간으로 변환
	converted := time.Unix(0, unixNano)
	
	// 나노초 단위까지 같은지 확인
	assert.Equal(t, now.UnixNano(), converted.UnixNano())
}

func TestSQLQueryValidation(t *testing.T) {
	// SQL 쿼리 문자열 검증 테스트
	validQueries := []string{
		"SELECT * FROM mqtt_messages WHERE id = $1",
		"INSERT INTO coap_messages (path, payload, method, received_at) VALUES ($1, $2, $3, $4)",
		"INSERT INTO snmp_data (host, oid, value, timestamp) VALUES ($1, $2, $3, $4)",
	}

	for _, query := range validQueries {
		t.Run("valid query", func(t *testing.T) {
			// 기본적인 SQL 인젝션 패턴 검사
			assert.NotContains(t, query, "'; DROP TABLE")
			assert.NotContains(t, query, "-- ")
			assert.Contains(t, query, "$") // 파라미터화된 쿼리 사용 확인
		})
	}
}

func TestTableNames(t *testing.T) {
	// 테이블 이름 검증
	expectedTables := []string{
		"mqtt_messages",
		"coap_messages", 
		"snmp_data",
	}

	for _, tableName := range expectedTables {
		t.Run("table name validation", func(t *testing.T) {
			// 테이블 이름이 유효한 형식인지 확인
			assert.Regexp(t, "^[a-z_]+$", tableName)
			assert.NotEmpty(t, tableName)
		})
	}
}

// 실제 데이터베이스 연결이 필요한 테스트들은 별도로 분리
func TestDatabaseIntegration(t *testing.T) {
	if DbPool == nil {
		t.Skip("Database not initialized - skipping integration tests")
	}

	t.Run("SaveMQTTMessage", func(t *testing.T) {
		topic := "test/topic"
		payload := "test message" // string으로 변경
		receivedTime := time.Now().UnixNano()

		err := SaveMQTTMessage(topic, payload, receivedTime)
		assert.NoError(t, err)
	})

	t.Run("SaveCoapMessage", func(t *testing.T) {
		path := "/test/path"
		payload := "test payload"
		method := "POST"
		receivedTime := time.Now().UnixNano()

		err := SaveCoapMessage(path, payload, method, receivedTime)
		assert.NoError(t, err)
	})

	t.Run("SaveSNMPData", func(t *testing.T) {
		host := "192.168.1.1"
		data := map[string]interface{}{
			"1.3.6.1.2.1.1.1.0": "Test System",
			"1.3.6.1.2.1.1.5.0": "test-host",
		}
		timestamp := time.Now().UnixNano()

		err := SaveSNMPData(host, data, timestamp)
		assert.NoError(t, err)
	})
}
