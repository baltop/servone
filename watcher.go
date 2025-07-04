package main

import (
	"log"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// 설정 파일 변경을 감지하는 ConfigWatcher 구조체 정의
type ConfigWatcher struct {
	configPath string            // 감시할 설정 파일의 경로
	server     *DynamicServer    // 설정 변경 시 재시작할 서버 인스턴스
	watcher    *fsnotify.Watcher // 파일 시스템 이벤트를 감시하는 watcher
}

// ConfigWatcher 생성자 함수
// configPath: 감시할 설정 파일 경로
// server: 설정 변경 시 재시작할 서버 인스턴스
func NewConfigWatcher(configPath string, server *DynamicServer) (*ConfigWatcher, error) {
	watcher, err := fsnotify.NewWatcher() // 파일 시스템 watcher 생성
	if err != nil {
		return nil, err
	}

	absPath, err := filepath.Abs(configPath) // 설정 파일의 절대 경로 구하기
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

// 감시를 시작하는 메서드
func (cw *ConfigWatcher) Start() error {
	configDir := filepath.Dir(cw.configPath) // 설정 파일이 위치한 디렉토리 경로
	err := cw.watcher.Add(configDir)         // 해당 디렉토리 감시 시작
	if err != nil {
		return err
	}

	go cw.watchLoop() // 이벤트 감지 루프를 고루틴으로 실행
	log.Printf("Started watching config file: %s", cw.configPath)

	return nil
}

// 파일 변경 이벤트를 감지하는 루프
func (cw *ConfigWatcher) watchLoop() {
	debounce := time.NewTimer(0) // 디바운스 타이머 생성
	debounce.Stop()              // 바로 멈춤

	for {
		select {
		case event, ok := <-cw.watcher.Events: // 파일 시스템 이벤트 수신
			if !ok {
				return
			}

			eventPath, _ := filepath.Abs(event.Name)
			// 설정 파일에 대한 쓰기/생성 이벤트만 감지
			if eventPath == cw.configPath && (event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create) {
				log.Printf("Config file event detected: %s %s", event.Op, event.Name)
				debounce.Reset(500 * time.Millisecond) // 0.5초 후에 reloadConfig 실행
			}

		case <-debounce.C:
			cw.reloadConfig() // 디바운스 타이머 만료 시 설정 재로드

		case err, ok := <-cw.watcher.Errors: // 감시 중 에러 발생 시
			if !ok {
				return
			}
			log.Printf("Config watcher error: %v", err)
		}
	}
}

// 설정 파일을 다시 읽고 서버에 반영하는 메서드
func (cw *ConfigWatcher) reloadConfig() {
	log.Println("Config file changed, reloading...")

	newConfig, err := LoadConfig(cw.configPath) // 설정 파일 재로드
	if err != nil {
		log.Printf("Failed to reload config: %v", err)
		return
	}

	cw.server.Reload(newConfig) // 서버에 새로운 설정 반영
}

// 감시 중지 메서드
func (cw *ConfigWatcher) Stop() {
	if cw.watcher != nil {
		cw.watcher.Close() // watcher 종료
	}
}
