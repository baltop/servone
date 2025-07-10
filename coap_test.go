package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/udp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockKafkaPublisher는 KafkaPublisherInterface의 모의(mock) 구현입니다.
type MockKafkaPublisher struct {
	PublishFunc func(topic string, data map[string]interface{})
}

func (m *MockKafkaPublisher) Publish(topic string, data map[string]interface{}) {
	if m.PublishFunc != nil {
		m.PublishFunc(topic, data)
	}
}

func (m *MockKafkaPublisher) Close() {
	// Do nothing
}

func TestCoapServer(t *testing.T) {
	// 테스트용 설정
	config := &Config{
		Coap: CoapConfig{
			Host: "localhost",
			Port: "5689", // 테스트용 포트
		},
		Endpoints: []EndpointConfig{
			{
				Path:   "/test",
				Method: "POST",
				Response: ResponseConfig{
					Status: 205, // CoAP Content
					Body:   "OK",
				},
			},
		},
		Database: DatabaseConfig{
			ConnectionString: testDBConnectionString, // from database_test.go
		},
	}

	// 모의 Kafka publisher 설정
	publishedChan := make(chan struct{}, 1)
	var publishedTopic string
	var publishedData map[string]interface{}
	mockPublisher := &MockKafkaPublisher{
		PublishFunc: func(topic string, data map[string]interface{}) {
			publishedTopic = topic
			publishedData = data
			publishedChan <- struct{}{}
		},
	}

	// 테스트용 데이터베이스 설정
	db := setupTestDB(t)
	defer db.Close()

	// CoAP 서버 시작
	coapServer := NewCoapServer(config, mockPublisher, db)
	defer coapServer.Stop()

	time.Sleep(100 * time.Millisecond) // 서버가 시작될 시간을 짧게 줍니다.

	t.Run("POST request to CoAP server", func(t *testing.T) {
		// CoAP 클라이언트 설정
		co, err := udp.Dial(config.Coap.Host + ":" + config.Coap.Port)
		require.NoError(t, err)
		defer co.Close()

		// 테스트용 데이터
		testData := map[string]interface{}{"key": "value"}
		payload, err := json.Marshal(testData)
		require.NoError(t, err)

		// CoAP POST 요청 보내기
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		resp, err := co.Post(ctx, "/test", message.AppJSON, bytes.NewReader(payload))
		require.NoError(t, err)

		// 응답 확인
		assert.Equal(t, codes.Code(205), resp.Code())
		body, err := io.ReadAll(resp.Body())
		require.NoError(t, err)
		assert.Equal(t, "OK", string(body))

		// Kafka 발행 확인 (채널을 통해 대기)
		select {
		case <-publishedChan:
			// 성공적으로 발행됨
			assert.Equal(t, "bz.test", publishedTopic)
			// 'data' 필드 안의 원본 데이터를 확인합니다.
			if data, ok := publishedData["data"].(map[string]interface{}); ok {
				assert.Equal(t, testData, data)
			} else {
				t.Errorf("publishedData['data'] is not of expected type")
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for Kafka publish")
		}

		// 데이터베이스 저장 확인
		var (
			id          int
			savedURL    string
			savedData   []byte
			savedParams []byte
			createdAt   int64
		)
		row := db.QueryRow("SELECT id, url, data, parameters, created_at FROM client_data WHERE url = $1", "/test")
		err = row.Scan(&id, &savedURL, &savedData, &savedParams, &createdAt)
		require.NoError(t, err)

		assert.Equal(t, "/test", savedURL)
		var unmarshaledData map[string]interface{}
		require.NoError(t, json.Unmarshal(savedData, &unmarshaledData))
		assert.Equal(t, testData, unmarshaledData)
	})
}
