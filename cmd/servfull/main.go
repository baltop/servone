// 프로그램의 진입점(main 함수) 및 서버 실행/종료 제어
package main

// 필요한 표준 라이브러리 임포트
import (
	"context"   // 컨텍스트(취소, 타임아웃 등) 제어용
	"log"       // 로그 출력용
	"os"        // 운영체제 기능(파일, 시그널 등)
	"os/signal" // OS 시그널 처리용
	"syscall"   // 시스템 콜 상수
	"time"      // 시간 관련 기능

	"servone/coap"
	"servone/config"
	"servone/db"
	"servone/kafka"
	"servone/mqttclient"
	"servone/server"
	"servone/snmpclient"
)

func main() {
	configPath := "config.yaml" // 사용할 설정 파일 경로

	// 설정 파일 로드
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err) // 설정 파일 로드 실패 시 프로그램 종료
	}

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

	// MQTT Client 생성 및 연결
	mqttClient, err := mqttclient.NewMQTTClient(cfg.MQTT.Broker, cfg.MQTT.ClientID, publisher, cfg)
	if err != nil {
		log.Fatalf("Failed to create MQTT client: %v", err)
	}
	defer mqttClient.Disconnect()
	if err := mqttClient.Subscribe("#"); err != nil {
		log.Fatalf("Failed to subscribe to MQTT topics: %v", err)
	}

	// 동적으로 설정을 반영하는 서버 인스턴스 생성

	coapServer := coap.NewCoapServer(cfg, publisher)

	// SNMP Client for GET and periodic WALK
	snmpClient := snmpclient.NewSNMPClient(&cfg.SNMP, db.DbPool, publisher)
	snmpClient.StartWalkScheduler() // Start the periodic walk
	defer snmpClient.Stop()

	// SNMP Trap Server
	trapServer := snmpclient.NewTrapServer(&cfg.SNMPTrap, publisher, db.DbPool)
	if err := trapServer.Start(); err != nil {
		log.Fatalf("Failed to start SNMP trap server: %v", err)
	}
	defer trapServer.Stop()

	// Set global SNMP client for HTTP handlers that still exist (e.g., GET)
	snmpclient.SetGlobalSNMPClient(snmpClient)

	server := server.NewDynamicServer(cfg, publisher)

	// 설정 파일 변경 감시를 위한 watcher 생성
	watcher, err := config.NewConfigWatcher(configPath, server, coapServer)
	if err != nil {
		log.Fatalf("Failed to create config watcher: %v", err)
	}

	// 설정 파일 변경 감시 시작
	err = watcher.Start()
	if err != nil {
		log.Fatalf("Failed to start config watcher: %v", err)
	}
	defer watcher.Stop() // main 함수 종료 시 watcher 정리

	// OS 시그널(종료 신호) 수신을 위한 채널 생성
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM) // Ctrl+C, 종료 시그널 감지

	// 서버 실행을 별도 고루틴에서 시작
	go func() {
		if err := server.Start(); err != nil {
			log.Fatalf("Server failed to start: %v", err) // 서버 실행 실패 시 종료
		}
	}()

	<-sigChan                              // 시그널이 들어올 때까지 대기
	log.Println("Shutting down server...") // 종료 로그 출력

	// Graceful shutdown 시작
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// CoAP 서버 종료
	coapServer.Stop()

	// HTTP 서버 종료
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("All services stopped successfully") // 모든 서비스 종료 완료
}
