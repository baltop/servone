package snmpclient

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleSNMPRequest(t *testing.T) {
	// Save original client
	originalClient := GlobalSNMPClient
	defer func() {
		GlobalSNMPClient = originalClient
	}()

	// Test with nil client
	t.Run("NilClient", func(t *testing.T) {
		GlobalSNMPClient = nil
		req := httptest.NewRequest("POST", "/api/snmp/get", nil)
		w := httptest.NewRecorder()

		HandleSNMPRequest(w, req, "get")

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
		}
	})

	// Test invalid JSON
	t.Run("InvalidJSON", func(t *testing.T) {
		GlobalSNMPClient = &SNMPClient{} // Mock client
		req := httptest.NewRequest("POST", "/api/snmp/get", bytes.NewBufferString("invalid json"))
		w := httptest.NewRecorder()

		HandleSNMPRequest(w, req, "get")

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	// Test unknown operation
	t.Run("UnknownOperation", func(t *testing.T) {
		GlobalSNMPClient = &SNMPClient{} // Mock client
		body := map[string]interface{}{"target": "localhost"}
		jsonBody, _ := json.Marshal(body)
		req := httptest.NewRequest("POST", "/api/snmp/unknown", bytes.NewBuffer(jsonBody))
		w := httptest.NewRecorder()

		HandleSNMPRequest(w, req, "unknown")

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})
}

func TestHandleGet(t *testing.T) {
	tests := []struct {
		name       string
		data       map[string]interface{}
		wantStatus int
	}{
		{
			name:       "MissingTarget",
			data:       map[string]interface{}{"oids": []interface{}{".1.3.6.1.2.1.1.1.0"}},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "MissingOIDs",
			data:       map[string]interface{}{"target": "localhost"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "EmptyOIDs",
			data:       map[string]interface{}{"target": "localhost", "oids": []interface{}{}},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "InvalidOIDFormat",
			data: map[string]interface{}{
				"target": "localhost",
				"oids":   []interface{}{123}, // Should be string
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			handleGet(w, tt.data)

			if w.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}
