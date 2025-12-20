package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestRecordHTTPRequest(t *testing.T) {
	// 메트릭 기록 테스트
	method := "GET"
	endpoint := "/api/test"
	status := "200"
	duration := 0.1

	// 메트릭 기록 (패닉이 발생하지 않는지 확인)
	assert.NotPanics(t, func() {
		RecordHTTPRequest(method, endpoint, status, duration)
	})
}

func TestRecordCoapRequest(t *testing.T) {
	method := "POST"
	endpoint := "/coap/test"
	status := "205"

	assert.NotPanics(t, func() {
		RecordCoapRequest(method, endpoint, status)
	})
}

func TestRecordMQTTMessage(t *testing.T) {
	topic := "test/topic"

	assert.NotPanics(t, func() {
		RecordMQTTMessage(topic)
	})
}

func TestRecordSNMPOperation(t *testing.T) {
	operation := "GET"
	host := "192.168.1.1"
	status := "success"

	assert.NotPanics(t, func() {
		RecordSNMPOperation(operation, host, status)
	})
}

func TestRecordDBOperation(t *testing.T) {
	operation := "INSERT"
	table := "mqtt_messages"
	status := "success"
	duration := 0.05

	assert.NotPanics(t, func() {
		RecordDBOperation(operation, table, status, duration)
	})
}

func TestRecordKafkaPublish(t *testing.T) {
	topic := "test-topic"
	status := "success"
	duration := 0.02

	assert.NotPanics(t, func() {
		RecordKafkaPublish(topic, status, duration)
	})
}

func TestUpdateDBConnectionMetrics(t *testing.T) {
	active := 5
	idle := 3
	total := 8

	assert.NotPanics(t, func() {
		UpdateDBConnectionMetrics(active, idle, total)
	})
}

func TestMetricsRegistration(t *testing.T) {
	// 메트릭이 Prometheus 레지스트리에 등록되었는지 확인
	metricFamilies, err := prometheus.DefaultGatherer.Gather()
	assert.NoError(t, err)
	assert.NotEmpty(t, metricFamilies)

	// 주요 메트릭들이 존재하는지 확인
	metricNames := make(map[string]bool)
	for _, mf := range metricFamilies {
		metricNames[*mf.Name] = true
	}

	expectedMetrics := []string{
		"servone_http_requests_total",
		"servone_http_request_duration_seconds",
		"servone_coap_requests_total",
		"servone_mqtt_messages_received_total",
		"servone_snmp_operations_total",
		"servone_db_operations_total",
		"servone_kafka_publish_total",
	}

	for _, metricName := range expectedMetrics {
		assert.True(t, metricNames[metricName], "Metric %s should be registered", metricName)
	}
}
