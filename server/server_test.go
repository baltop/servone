package server

import (
	"net/http"
	"net/http/httptest"
	"servone/config"
	"strings"
	"testing"
)

type MockKafkaPublisher struct {
	PublishedTopic string
	PublishedData  map[string]interface{}
}

// Close implements kafka.KafkaPublisherInterface.
func (m *MockKafkaPublisher) Close() {
	panic("unimplemented")
}

func (m *MockKafkaPublisher) Publish(topic string, data map[string]interface{}) error {
	m.PublishedTopic = topic
	m.PublishedData = data
	return nil
}

func TestDynamicServer(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{Port: "8080", Host: "localhost"},
		Endpoints: []config.EndpointConfig{
			{
				Path:   "/hello",
				Method: "GET",
				Response: config.ResponseConfig{
					Status: 200,
					Body:   "Hello, World!",
					Headers: map[string]string{
						"Content-Type": "text/plain",
					},
				},
			},
			{
				Path:   "/users/{id}",
				Method: "GET",
				Response: config.ResponseConfig{
					Status: 200,
					Body:   `{"id": {{.id}}}`,
					Headers: map[string]string{
						"Content-Type": "application/json",
					},
				},
			},
			{
				Path:   "/create",
				Method: "POST",
				Response: config.ResponseConfig{
					Status: 201,
					Body:   "Created",
				},
			},
		},
	}

	// Mock Kafka publisher
	mockPublisher := &MockKafkaPublisher{}

	ds := NewDynamicServer(cfg, mockPublisher)

	t.Run("Basic GET request", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/hello", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		ds.router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusOK)
		}

		expectedBody := "Hello, World!"
		if rr.Body.String() != expectedBody {
			t.Errorf("handler returned unexpected body: got %v want %v",
				rr.Body.String(), expectedBody)
		}

		expectedHeader := "text/plain"
		if rr.Header().Get("Content-Type") != expectedHeader {
			t.Errorf("handler returned wrong header: got %v want %v",
				rr.Header().Get("Content-Type"), expectedHeader)
		}
	})

	t.Run("GET request with path variables", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/users/123", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		ds.router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusOK)
		}

		expectedBody := `{"id": 123}`
		if rr.Body.String() != expectedBody {
			t.Errorf("handler returned unexpected body: got %v want %v",
				rr.Body.String(), expectedBody)
		}
	})

	t.Run("POST request", func(t *testing.T) {
		jsonBody := `{"name": "test"}`
		req, err := http.NewRequest("POST", "/create", strings.NewReader(jsonBody))
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		ds.router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusCreated {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusCreated)
		}

		expectedBody := "Created"
		if rr.Body.String() != expectedBody {
			t.Errorf("handler returned unexpected body: got %v want %v",
				rr.Body.String(), expectedBody)
		}
	})

	t.Run("Route not found", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/notfound", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		ds.router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusNotFound)
		}
	})
}
