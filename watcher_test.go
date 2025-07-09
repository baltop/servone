

package main

import (
	"os"
	"sync"
	"testing"
	"time"
)

// MockDynamicServer is a mock implementation of DynamicServer for testing.
type MockDynamicServer struct {
	reloadCalled bool
	mu           sync.Mutex
}

// Reload marks the mock server as reloaded.
func (m *MockDynamicServer) Reload(newConfig *Config) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reloadCalled = true
}

func (m *MockDynamicServer) IsReloaded() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.reloadCalled
}

func TestConfigWatcher(t *testing.T) {
	// Create a temporary config file
	content := "server:\n  port: \"8081\""
	tmpfile, err := os.CreateTemp("", "config.*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	

	// Create a new config watcher
	watcher, err := NewConfigWatcher(tmpfile.Name(), (*DynamicServer)(nil)) // we pass nil for the server and then set the mock
	if err != nil {
		t.Fatalf("NewConfigWatcher() error = %v", err)
	}
	// This is a bit of a hack, as the server field is not exported.
	// A better approach would be to use an interface for the server.
	// For this test, we will use a real server that does nothing.

	// Let's create a real server, but we will check our mock server
	config, _ := LoadConfig(tmpfile.Name())
	ds := NewDynamicServer(config, nil)

	// We need a way to intercept the reload. Let's modify the watcher to accept an interface.
	// Since we can't modify the original code, we will have to test it differently.

	// Let's try another approach. We can check if the config of the server has changed.

	// Create a new config watcher
	watcher, err = NewConfigWatcher(tmpfile.Name(), ds)
	if err != nil {
		t.Fatalf("NewConfigWatcher() error = %v", err)
	}

	err = watcher.Start()
	if err != nil {
		t.Fatalf("watcher.Start() error = %v", err)
	}
	defer watcher.Stop()

	// Wait for the watcher to start
	time.Sleep(100 * time.Millisecond)

	// Update the config file
	newContent := "server:\n  port: \"8082\""
	f, err := os.OpenFile(tmpfile.Name(), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write([]byte(newContent)); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	// Wait for the watcher to trigger the reload
	time.Sleep(1 * time.Second)

	// Check if the server's config has been updated
	if ds.config.Server.Port != "8082" {
		t.Errorf("server config not reloaded, port = %s, want 8082", ds.config.Server.Port)
	}
}
