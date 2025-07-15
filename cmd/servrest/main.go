// REST API 서버만 실행하는 프로그램 진입점
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"servone/config"
	"servone/db"
	"servone/kafka"
	"servone/server"
)

func main() {
	configPath := "config.yaml"

	// 설정 파일 로드
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 데이터베이스 초기화
	if err := db.InitDB(cfg.Database.ConnectionString); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer func() {
		if err := db.CloseDB(); err != nil {
			log.Printf("Failed to close database: %v", err)
		}
	}()

	db.SetupDatabase()

	// Kafka Publisher 생성
	publisher, err := kafka.NewKafkaPublisher(cfg.Kafka.Brokers)
	if err != nil {
		log.Fatalf("Failed to create kafka publisher: %v", err)
	}
	defer publisher.Close()

	// HTTP REST API 서버만 생성
	server := server.NewDynamicServer(cfg, publisher)

	// 설정 파일 변경 감시를 위한 watcher 생성 (CoAP 서버는 nil로 전달)
	watcher, err := config.NewConfigWatcher(configPath, server, nil)
	if err != nil {
		log.Fatalf("Failed to create config watcher: %v", err)
	}

	// 설정 파일 변경 감시 시작
	err = watcher.Start()
	if err != nil {
		log.Fatalf("Failed to start config watcher: %v", err)
	}
	defer watcher.Stop()

	// OS 시그널 수신을 위한 채널 생성
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// REST API 서버 실행
	go func() {
		log.Println("Starting REST API server on port", cfg.Rest.Port)
		if err := server.Start(); err != nil {
			log.Fatalf("REST API server failed to start: %v", err)
		}
	}()

	<-sigChan
	log.Println("Shutting down REST API server...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// HTTP 서버 종료
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("REST API server stopped successfully")
}
