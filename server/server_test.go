package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"servone/config"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockKafkaPublisher struct {
	PublishedTopic string
	PublishedData  map[string]interface{}
}

func (m *MockKafkaPublisher) Close() {
	// Do nothing for mock
}

func (m *MockKafkaPublisher) Publish(topic string, data map[string]interface{}) error {
	m.PublishedTopic = topic
	m.PublishedData = data
	return nil
}

func TestDynamicServer(t *testing.T) {
	// 테스트용 설정
	cfg := &config.Config{
		Rest: config.RestConfig{
			Host: "localhost",
			Port: "8080",
			Endpoints: []config.EndpointConfig{
				{
					Path:   "/api/test",
					Method: "GET",
					Response: config.ResponseConfig{
						Status: 200,
						Body:   `{"message": "test"}`,
						Headers: map[string]string{
							"Content-Type": "application/json",
						},
					},
				},
				{
					Path:   "/api/health",
					Method: "GET",
					Response: config.ResponseConfig{
						Status: 200,
						Body:   `{"status": "OK", "timestamp": "{{.timestamp}}"}`,
						Headers: map[string]string{
							"Content-Type": "application/json",
						},
					},
				},
			},
		},
	}

	mockPublisher := &MockKafkaPublisher{}
	server := NewDynamicServer(cfg, mockPublisher)

	t.Run("GET /api/test", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/test", nil)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		server.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
		
		var response map[string]interface{}
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "test", response["message"])
	})

	t.Run("GET /api/health with template", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/health", nil)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		server.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
		
		var response map[string]interface{}
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "OK", response["status"])
		assert.NotEmpty(t, response["timestamp"])
		
		// 타임스탬프가 유효한 RFC3339 형식인지 확인
		_, err = time.Parse(time.RFC3339, response["timestamp"].(string))
		assert.NoError(t, err)
	})

	t.Run("POST with body", func(t *testing.T) {
		// 새로운 설정으로 새 서버 생성
		newCfg := &config.Config{
			Rest: config.RestConfig{
				Host: "localhost",
				Port: "8080",
				Endpoints: []config.EndpointConfig{
					{
						Path:   "/api/data",
						Method: "POST",
						Response: config.ResponseConfig{
							Status: 201,
							Body:   `{"result": "created"}`,
							Headers: map[string]string{
								"Content-Type": "application/json",
							},
						},
					},
				},
			},
		}
		
		// 새 서버 인스턴스 생성
		newServer := NewDynamicServer(newCfg, mockPublisher)

		testData := map[string]interface{}{"key": "value"}
		jsonData, _ := json.Marshal(testData)
		
		req, err := http.NewRequest("POST", "/api/data", bytes.NewBuffer(jsonData))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		newServer.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)
		
		var response map[string]interface{}
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "created", response["result"])
	})

	t.Run("404 for undefined route", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/undefined", nil)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		server.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})
}
