package main

import (
	"log"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

type ConfigWatcher struct {
	configPath string
	server     *DynamicServer
	watcher    *fsnotify.Watcher
}

func NewConfigWatcher(configPath string, server *DynamicServer) (*ConfigWatcher, error) {
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
		server:     server,
		watcher:    watcher,
	}

	return cw, nil
}

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

func (cw *ConfigWatcher) watchLoop() {
	debounce := time.NewTimer(0)
	debounce.Stop()

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

func (cw *ConfigWatcher) reloadConfig() {
	log.Println("Config file changed, reloading...")
	
	newConfig, err := LoadConfig(cw.configPath)
	if err != nil {
		log.Printf("Failed to reload config: %v", err)
		return
	}

	cw.server.Reload(newConfig)
}

func (cw *ConfigWatcher) Stop() {
	if cw.watcher != nil {
		cw.watcher.Close()
	}
}