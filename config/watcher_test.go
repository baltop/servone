package config

import (
	"os"

	"testing"
	"time"
)

// MockReloadable is a mock implementation of the Reloadable interface for testing.
type MockReloadable struct {
	ReloadFunc func(config *Config)
}

func (m *MockReloadable) Reload(config *Config) {
	if m.ReloadFunc != nil {
		m.ReloadFunc(config)
	}
}

// TestConfigWatcher tests the configuration watcher functionality.
func TestConfigWatcher(t *testing.T) {
	// Create a temporary config file
	tempFile, err := os.CreateTemp("", "config.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Initial configuration
	initialConfig := `
endpoints:
  - path: /test
    method: GET
    response:
      status: 200
      body: "Hello, World!"
`
	if _, err := tempFile.Write([]byte(initialConfig)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tempFile.Close()

	// Mock DynamicServer
	var serverReloaded bool
	mockServer := &MockReloadable{
		ReloadFunc: func(config *Config) {
			serverReloaded = true
		},
	}

	// Mock CoapServer
	var coapReloaded bool
	mockCoapServer := &MockReloadable{
		ReloadFunc: func(config *Config) {
			coapReloaded = true
		},
	}

	// Start the watcher
	watcher, err := NewConfigWatcher(tempFile.Name(), mockServer, mockCoapServer)
	if err != nil {
		t.Fatalf("Failed to create config watcher: %v", err)
	}
	err = watcher.Start()
	if err != nil {
		t.Fatalf("Failed to start watcher: %v", err)
	}
	defer watcher.Stop()

	// Give the watcher a moment to start
	time.Sleep(100 * time.Millisecond)

	// Update the config file
	updatedConfig := `
endpoints:
  - path: /test
    method: POST
    response:
      status: 201
      body: "Created!"
`
	if err := os.WriteFile(tempFile.Name(), []byte(updatedConfig), 0644); err != nil {
		t.Fatalf("Failed to write updated config: %v", err)
	}

	// Wait for the watcher to detect the change and reload
	time.Sleep(2 * time.Second)

	if !serverReloaded {
		t.Error("Expected server to be reloaded, but it wasn't")
	}
	if !coapReloaded {
		t.Error("Expected coap server to be reloaded, but it wasn't")
	}
}
