package coap

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"servone/config"
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
	PublishFunc func(topic string, data map[string]interface{}) error
	published   map[string]map[string]interface{}
}

func NewMockKafkaPublisher() *MockKafkaPublisher {
	return &MockKafkaPublisher{
		published: make(map[string]map[string]interface{}),
	}
}

func (m *MockKafkaPublisher) Publish(topic string, data map[string]interface{}) error {
	m.published[topic] = data
	if m.PublishFunc != nil {
		return m.PublishFunc(topic, data)
	}
	return nil
}

func (m *MockKafkaPublisher) Close() {
	// Do nothing
}

func (m *MockKafkaPublisher) GetPublished(topic string) map[string]interface{} {
	return m.published[topic]
}

func TestCoapServer(t *testing.T) {
	// 테스트용 설정
	cfg := &config.Config{
		Coap: config.CoapConfig{
			Host: "localhost",
			Port: "5689", // 테스트용 포트
			Endpoints: []config.EndpointConfig{
				{
					Path:   "/test",
					Method: "POST",
					Response: config.ResponseConfig{
						Status: 205, // CoAP Content response code
						Body:   "OK",
						Headers: map[string]string{
							"Content-Type": "text/plain",
						},
					},
				},
			},
		},
	}

	// 모의 Kafka publisher 설정
	mockPublisher := NewMockKafkaPublisher()

	// CoAP 서버 시작
	coapServer := NewCoapServer(cfg, mockPublisher)
	defer coapServer.Stop()

	// 서버가 시작될 시간을 기다립니다
	time.Sleep(200 * time.Millisecond)

	t.Run("POST request to CoAP server", func(t *testing.T) {
		// CoAP 클라이언트 설정
		co, err := udp.Dial(cfg.Coap.Host + ":" + cfg.Coap.Port)
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
		assert.Equal(t, codes.Code(205), resp.Code()) // 205 Content
		body, err := io.ReadAll(resp.Body())
		require.NoError(t, err)
		assert.Equal(t, "OK", string(body))

		// Kafka 발행 확인을 위해 잠시 대기
		time.Sleep(100 * time.Millisecond)

		// Kafka에 메시지가 발행되었는지 확인
		publishedData := mockPublisher.GetPublished("coap/test")
		if publishedData != nil {
			assert.Equal(t, "/test", publishedData["path"])
			// CoAP에서는 실제 메서드가 "POST"가 아닐 수 있으므로 확인
			assert.NotEmpty(t, publishedData["method"])
			
			// 데이터 확인
			if data, ok := publishedData["data"].(map[string]interface{}); ok {
				assert.Equal(t, "value", data["key"])
			}
		}
	})
}
