package main

import (
	"log"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Reloadable defines the interface for objects that can be reloaded with a new configuration.
type Reloadable interface {
	Reload(config *Config)
}

// ConfigWatcher detects changes in the configuration file and reloads servers.
type ConfigWatcher struct {
	configPath string
	servers    []Reloadable
	watcher    *fsnotify.Watcher
}

// NewConfigWatcher creates a new ConfigWatcher.
func NewConfigWatcher(configPath string, servers ...Reloadable) (*ConfigWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	absPath, err := filepath.Abs(configPath)
	if err != nil {
		return nil, err
	}

	cw := &ConfigWatcher{
		configPath: absPath,
		servers:    servers,
		watcher:    watcher,
	}

	return cw, nil
}

// Start begins watching the configuration file.
func (cw *ConfigWatcher) Start() error {
	configDir := filepath.Dir(cw.configPath)
	err := cw.watcher.Add(configDir)
	if err != nil {
		return err
	}

	go cw.watchLoop()
	log.Printf("Started watching config file: %s", cw.configPath)

	return nil
}

// watchLoop listens for file system events.
func (cw *ConfigWatcher) watchLoop() {
	debounce := time.NewTimer(0)
	if !debounce.Stop() {
		<-debounce.C
	}

	for {
		select {
		case event, ok := <-cw.watcher.Events:
			if !ok {
				return
			}

			eventPath, _ := filepath.Abs(event.Name)
			if eventPath == cw.configPath && (event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create) {
				log.Printf("Config file event detected: %s %s", event.Op, event.Name)
				debounce.Reset(500 * time.Millisecond)
			}

		case <-debounce.C:
			cw.reloadConfig()

		case err, ok := <-cw.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Config watcher error: %v", err)
		}
	}
}

// reloadConfig reloads the configuration and applies it to the servers.
func (cw *ConfigWatcher) reloadConfig() {
	log.Println("Config file changed, reloading...")

	newConfig, err := LoadConfig(cw.configPath)
	if err != nil {
		log.Printf("Failed to reload config: %v", err)
		return
	}

	for _, s := range cw.servers {
		if s != nil {
			s.Reload(newConfig)
		}
	}
}

// Stop stops the watcher.
func (cw *ConfigWatcher) Stop() {
	if cw.watcher != nil {
		cw.watcher.Close()
	}
}
