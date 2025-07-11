package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP metrics
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "servone_http_requests_total",
			Help: "Total number of HTTP requests processed",
		},
		[]string{"method", "endpoint", "status"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "servone_http_request_duration_seconds",
			Help:    "HTTP request latencies in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	// CoAP metrics
	CoapRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "servone_coap_requests_total",
			Help: "Total number of CoAP requests processed",
		},
		[]string{"method", "endpoint", "status"},
	)

	// MQTT metrics
	MQTTMessagesReceived = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "servone_mqtt_messages_received_total",
			Help: "Total number of MQTT messages received",
		},
		[]string{"topic"},
	)

	// SNMP metrics
	SNMPOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "servone_snmp_operations_total",
			Help: "Total number of SNMP operations",
		},
		[]string{"operation", "host", "status"},
	)

	// Database metrics
	DBOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "servone_db_operations_total",
			Help: "Total number of database operations",
		},
		[]string{"operation", "table", "status"},
	)

	DBOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "servone_db_operation_duration_seconds",
			Help:    "Database operation latencies in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation", "table"},
	)

	// Kafka metrics
	KafkaPublishTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "servone_kafka_publish_total",
			Help: "Total number of Kafka publish attempts",
		},
		[]string{"topic", "status"},
	)

	KafkaPublishDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "servone_kafka_publish_duration_seconds",
			Help:    "Kafka publish latencies in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"topic"},
	)

	// Connection pool metrics
	DBConnectionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "servone_db_connections_active",
			Help: "Number of active database connections",
		},
	)

	DBConnectionsIdle = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "servone_db_connections_idle",
			Help: "Number of idle database connections",
		},
	)

	DBConnectionsTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "servone_db_connections_total",
			Help: "Total number of database connections",
		},
	)
)

// RecordHTTPRequest records HTTP request metrics
func RecordHTTPRequest(method, endpoint, status string, duration float64) {
	HTTPRequestsTotal.WithLabelValues(method, endpoint, status).Inc()
	HTTPRequestDuration.WithLabelValues(method, endpoint).Observe(duration)
}

// RecordCoapRequest records CoAP request metrics
func RecordCoapRequest(method, endpoint, status string) {
	CoapRequestsTotal.WithLabelValues(method, endpoint, status).Inc()
}

// RecordMQTTMessage records MQTT message received
func RecordMQTTMessage(topic string) {
	MQTTMessagesReceived.WithLabelValues(topic).Inc()
}

// RecordSNMPOperation records SNMP operation metrics
func RecordSNMPOperation(operation, host, status string) {
	SNMPOperationsTotal.WithLabelValues(operation, host, status).Inc()
}

// RecordDBOperation records database operation metrics
func RecordDBOperation(operation, table, status string, duration float64) {
	DBOperationsTotal.WithLabelValues(operation, table, status).Inc()
	DBOperationDuration.WithLabelValues(operation, table).Observe(duration)
}

// RecordKafkaPublish records Kafka publish metrics
func RecordKafkaPublish(topic, status string, duration float64) {
	KafkaPublishTotal.WithLabelValues(topic, status).Inc()
	KafkaPublishDuration.WithLabelValues(topic).Observe(duration)
}

// UpdateDBConnectionMetrics updates database connection pool metrics
func UpdateDBConnectionMetrics(active, idle, total int) {
	DBConnectionsActive.Set(float64(active))
	DBConnectionsIdle.Set(float64(idle))
	DBConnectionsTotal.Set(float64(total))
}