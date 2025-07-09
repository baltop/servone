// 설정 파일을 처리하는 패키지
package main

// yaml 파일 파싱을 위한 패키지와 파일 입출력 패키지 임포트
import (
	"io" // 파일 읽기용
	"os" // 파일 열기용

	"gopkg.in/yaml.v3" // YAML 파싱용
)

// 전체 설정을 담는 최상위 구조체
type Config struct {
	Database  DatabaseConfig   `yaml:"database"` // 데이터베이스 관련 설정
	Server    ServerConfig     `yaml:"server"`   // 서버 관련 설정
	Kafka     KafkaConfig      `yaml:"kafka"`
	MQTT      MQTTConfig       `yaml:"mqtt"`
	Endpoints []EndpointConfig `yaml:"endpoints"` // 엔드포인트(라우트) 설정 목록
}

// MQTT 관련 설정 구조체
type MQTTConfig struct {
	Broker   string `yaml:"broker"`
	ClientID string `yaml:"client_id"`
}

// 데이터베이스 관련 설정 구조체
type DatabaseConfig struct {
	ConnectionString string `yaml:"connection_string"` // 데이터베이스 연결 문자열
}

// 서버 관련 설정 구조체
type ServerConfig struct {
	Port string `yaml:"port"` // 서버 포트 번호
	Host string `yaml:"host"` // 서버 호스트 주소
}

// Kafka 관련 설정 구조체
type KafkaConfig struct {
	Brokers []string `yaml:"brokers"`
}

// 각 엔드포인트(라우트)별 설정 구조체
type EndpointConfig struct {
	Path     string         `yaml:"path"`     // 엔드포인트 경로
	Method   string         `yaml:"method"`   // HTTP 메서드(GET, POST 등)
	Response ResponseConfig `yaml:"response"` // 응답 설정
}

// 엔드포인트 응답 설정 구조체
type ResponseConfig struct {
	Status  int               `yaml:"status"`  // HTTP 상태 코드
	Body    string            `yaml:"body"`    // 응답 본문
	Headers map[string]string `yaml:"headers"` // 응답 헤더
}

// 설정 파일을 읽어 Config 구조체로 반환하는 함수
// filename: 읽을 설정 파일 경로
func LoadConfig(filename string) (*Config, error) {
	file, err := os.Open(filename) // 파일 열기
	if err != nil {
		return nil, err // 파일 열기 실패 시 에러 반환
	}
	defer file.Close() // 함수 종료 시 파일 닫기

	data, err := io.ReadAll(file) // 파일 전체 내용 읽기
	if err != nil {
		return nil, err // 읽기 실패 시 에러 반환
	}

	var config Config                   // 파싱 결과를 담을 구조체
	err = yaml.Unmarshal(data, &config) // YAML 파싱
	if err != nil {
		return nil, err // 파싱 실패 시 에러 반환
	}

	return &config, nil // 파싱된 설정 반환
}
