# Servone - 통합 프로토콜 서버

Servone은 다양한 프로토콜을 지원하는 유연하고 확장 가능한 통합 서버 애플리케이션입니다. 주요 목표는 단일 애플리케이션 내에서 HTTP, CoAP, MQTT, SNMP와 같은 여러 통신 프로토콜을 처리하는 중앙 집중식 솔루션을 제공하는 것입니다.

## 주요 기능

*   **다중 프로토콜 지원:** HTTP (REST API), CoAP, MQTT 및 SNMP 프로토콜을 기본적으로 지원합니다.
*   **동적 설정:** `config.yaml` 파일을 통해 서버의 동작을 동적으로 구성할 수 있습니다. 변경 사항은 서버 재시작 없이 실시간으로 반영됩니다.
*   **메시지 브로커 통합:** Kafka와의 통합을 통해 수신된 메시지를 안정적으로 처리하고 다른 시스템으로 전달할 수 있습니다.
*   **데이터베이스 로깅:** PostgreSQL 데이터베이스와의 통합을 통해 수신된 메시지와 장치 데이터를 저장하고 관리합니다.
*   **모니터링:** Prometheus 메트릭을 통해 서버의 상태와 성능을 모니터링할 수 있습니다.

## 지원 프로토콜

*   **HTTP (REST API):** 동적으로 생성된 RESTful API 엔드포인트를 제공합니다.
*   **CoAP:** 경량의 IoT 디바이스를 위한 CoAP 프로토콜을 지원합니다.
*   **MQTT:** 발행/구독 모델 기반의 MQTT 프로토콜을 지원하여 실시간 메시징을 처리합니다.
*   **SNMP:** 네트워크 장치를 모니터링하고 관리하기 위한 SNMP 프로토콜을 지원합니다.

## 아키텍처

Servone은 두 가지 실행 모드를 제공합니다.

*   `cmd/servfull`: 모든 기능을 포함하는 전체 서버를 실행합니다. 여기에는 HTTP, CoAP, MQTT, SNMP 서비스가 포함됩니다.
*   `cmd/servrest`: 경량화된 버전으로, HTTP (REST API) 및 SNMP 기능만 제공합니다.

이 프로젝트의 핵심은 `config.yaml` 파일을 통해 서버의 동작을 동적으로 제어하는 것입니다. 이 파일을 수정하면 서버를 재시작할 필요 없이 새로운 API 엔드포인트를 추가하거나 기존 엔드포인트를 변경할 수 있습니다. 서버는 파일 변경을 감지하고 실시간으로 구성을 리로드하여 변경 사항을 적용합니다.

## 시작하기

### 필수 구성 요소

*   [Go](https://golang.org/dl/) (버전 1.21 이상)
*   [PostgreSQL](https://www.postgresql.org/download/)
*   [Kafka](https://kafka.apache.org/downloads)
*   [MQTT Broker](https://mqtt.org/software/) (예: Mosquitto)

### 설치

1.  **저장소 복제:**
    ```sh
    git clone https://github.com/your-username/servone.git
    cd servone
    ```

2.  **의존성 설치:**
    ```sh
    go mod download
    ```

3.  **데이터베이스 설정:**
    *   PostgreSQL 데이터베이스를 생성합니다.
    *   `config.yaml` 파일의 `database.connection_string`을 사용자의 데이터베이스 정보로 수정합니다.

4.  **환경 설정:**
    *   `config.yaml` 파일을 프로젝트 루트에 생성하고 아래의 "설정 파일" 섹션을 참고하여 내용을 작성합니다.
    *   Kafka 및 MQTT 브로커가 실행 중인지 확인하고 `config.yaml`에 브로커 주소를 올바르게 설정합니다.

### 실행

*   **전체 기능 서버 실행 (`servfull`):**
    ```sh
    go run cmd/servfull/main.go
    ```

*   **REST API 서버만 실행 (`servrest`):**
    ```sh
    go run cmd/servrest/main.go
    ```

## 설정 파일 (`config.yaml`)

`config.yaml` 파일은 Servone 애플리케이션의 모든 설정을 관리합니다.

```yaml
database:
  connection_string: "postgresql://user:password@localhost:5432/dbname?sslmode=disable"

kafka:
  brokers:
    - "localhost:9092"

mqtt:
  broker: "tcp://localhost:1883"
  client_id: "servone-mqtt-client"

snmp:
  port: 1161
  timeout: 5
  retries: 3
  username: "snmpuser"
  auth_protocol: "SHA256"
  auth_passphrase: "authpassword"
  priv_protocol: "AES256"
  priv_passphrase: "privpassword"
  trap_enabled: true
  trap_host: "0.0.0.0"
  trap_port: 1162

server:
  host: "0.0.0.0"
  port: "8090"

rest:
  endpoints:
    - path: "/api/users"
      method: "GET"
      response:
        status: 200
        body: '{"users": [{"id": 1, "name": "John"}, {"id": 2, "name": "Jane"}]}'
        headers:
          Content-Type: "application/json"
    # ... 다른 REST 엔드포인트들

coap:
  host: "0.0.0.0"
  port: "5683"
  endpoints:
    - path: "/api/temperature"
      method: "POST"
      # ... CoAP 엔드포인트 설정
```

### 섹션 설명

*   `database`: PostgreSQL 데이터베이스 연결 정보를 설정합니다.
*   `kafka`: Kafka 브로커의 주소를 설정합니다.
*   `mqtt`: MQTT 브로커의 주소와 클라이언트 ID를 설정합니다.
*   `snmp`: SNMP 에이전트 및 트랩 수신 설정을 구성합니다.
*   `server`: HTTP 서버의 호스트와 포트를 설정합니다.
*   `rest`: 동적으로 생성할 REST API 엔드포인트를 정의합니다.
    *   `path`: API 경로 (예: `/api/users/{id}`)
    *   `method`: HTTP 메서드 (GET, POST, PUT, DELETE 등)
    *   `response`: HTTP 응답 설정
        *   `status`: 응답 상태 코드
        *   `body`: 응답 본문 (템플릿 지원: `{{.timestamp}}`, `{{.path_variable}}`)
        *   `headers`: 응답 헤더
*   `coap`: CoAP 서버 및 엔드포인트 설정을 정의합니다. `rest` 섹션과 유사한 구조를 가집니다.

## API 엔드포인트

다음은 `config.yaml`에 기본적으로 정의된 몇 가지 API 엔드포인트 예시입니다.

### REST API

*   **GET /api/users**
    *   설명: 모든 사용자 목록을 반환합니다.
    *   응답 예시:
        ```json
        {"users": [{"id": 1, "name": "John"}, {"id": 2, "name": "Jane"}]}
        ```

*   **POST /api/users**
    *   설명: 새로운 사용자를 생성합니다.
    *   응답 예시:
        ```json
        {"result": "success"}
        ```

*   **GET /api/health**
    *   설명: 서버의 상태와 현재 타임스탬프를 확인합니다.
    *   응답 예시:
        ```json
        {"status": "OK", "timestamp": "2023-10-27T10:00:00Z"}
        ```

*   **POST /api/snmp/get**
    *   설명: SNMP GET 작업을 트리거합니다.
    *   요청 본문 예시:
        ```json
        {
            "target": "192.168.1.1",
            "oids": ["1.3.6.1.2.1.1.1.0", "1.3.6.1.2.1.1.5.0"]
        }
        ```

### CoAP

*   **POST /api/temperature**
    *   설명: CoAP를 통해 온도 데이터를 전송합니다.
    *   페이로드 예시:
        ```json
        {"sensor_id": "temp-001", "value": 25.5}
        ```

## 테스트

프로젝트의 모든 테스트를 실행하려면 다음 명령어를 사용하세요.

```sh
go test ./...
```

## 개발 및 기여

이 프로젝트에 기여하고 싶으시다면, 다음 단계를 따라주세요.

1.  이 저장소를 포크(fork)합니다.
2.  새로운 기능이나 버그 수정을 위한 브랜치를 생성합니다 (`git checkout -b feature/your-feature-name`).
3.  코드를 수정하고, 변경 사항에 대한 테스트를 추가합니다.
4.  모든 테스트가 통과하는지 확인합니다.
5.  변경 사항을 커밋하고 브랜치에 푸시합니다.
6.  Pull Request를 생성하여 변경 사항을 설명하고 리뷰를 요청합니다.

## 유용한 명령어

### 데이터베이스 조회

*   **MQTT 메시지 조회:**
    ```sql
    SELECT id, topic, encode(payload, 'escape')::text, created_at FROM public.mqtt_messages ORDER BY id DESC LIMIT 100;
    ```

### CoAP 클라이언트

*   **CoAP POST 요청:**
    ```sh
    echo -n '{"key": "value"}' | coap post coap://localhost:5683/hands/left
    ```
