# Servone - Product Requirements Document (PRD)

## 1. 제품 개요

### 1.1 제품명
**Servone** - Multi-Protocol Integration Server

### 1.2 제품 설명
Servone은 다양한 통신 프로토콜(HTTP REST API, CoAP, MQTT, SNMP)을 통합하여 데이터를 수집하고, PostgreSQL에 저장하며, Kafka로 스트리밍하는 Go 언어 기반의 멀티 프로토콜 통합 서버입니다.

### 1.3 대상 사용자
- IoT 디바이스 통합이 필요한 개발팀
- 네트워크 모니터링 시스템 운영자
- 다중 프로토콜 게이트웨이가 필요한 엔터프라이즈 환경

### 1.4 핵심 가치
- **프로토콜 통합**: 하나의 서버에서 HTTP, CoAP, MQTT, SNMP 지원
- **유연한 설정**: YAML 기반 동적 엔드포인트 구성 및 핫 리로드
- **데이터 파이프라인**: 실시간 데이터 수집 → 저장 → 스트리밍

---

## 2. 목표 및 목적

### 2.1 비즈니스 목표
1. 다양한 IoT 디바이스 및 네트워크 장비와의 통합 단순화
2. 단일 진입점을 통한 데이터 수집 및 처리 일원화
3. 실시간 데이터 스트리밍을 통한 빠른 데이터 활용

### 2.2 기술 목표
1. 고성능 비동기 처리를 통한 확장성 확보
2. 무중단 설정 변경(Hot Reload) 지원
3. 표준 프로토콜 준수를 통한 호환성 보장

---

## 3. 주요 기능

### 3.1 동적 HTTP REST API 서버

| 기능 | 설명 |
|------|------|
| 동적 엔드포인트 | config.yaml에서 정의한 엔드포인트 자동 생성 |
| 경로 파라미터 | `/api/products/{id}` 형태의 동적 경로 지원 |
| 템플릿 응답 | `{{.timestamp}}`, `{{.variable}}` 치환 |
| 핫 리로드 | 설정 파일 변경 시 500ms 디바운스 후 자동 적용 |
| 메트릭 노출 | Prometheus 형식의 `/metrics` 엔드포인트 |

**지원 메서드**: GET, POST, PUT, DELETE

### 3.2 CoAP 프로토콜 서버

| 기능 | 설명 |
|------|------|
| UDP 기반 CoAP | RFC 7252 표준 준수 |
| 동적 라우팅 | 설정 파일 기반 엔드포인트 생성 |
| 메시지 처리 | JSON 페이로드 파싱 및 저장 |
| Kafka 연동 | `coap.{path}` 형태 토픽으로 발행 |

**기본 포트**: 5683

### 3.3 MQTT 클라이언트

| 기능 | 설명 |
|------|------|
| 와일드카드 구독 | `#` 패턴으로 모든 토픽 구독 |
| QoS 지원 | Level 1 (At least once) |
| 데이터 브리징 | MQTT → PostgreSQL + Kafka |
| 토픽 매핑 | `mq.{original_topic}` 형태로 Kafka 발행 |

### 3.4 SNMP 클라이언트 및 트랩 서버

| 기능 | 설명 |
|------|------|
| SNMP v3 | USM 기반 인증 (MD5, SHA, SHA256 등) |
| 암호화 | DES, AES, AES256 등 Privacy 프로토콜 |
| GET 연산 | 단일 OID 값 조회 |
| WALK 연산 | 주기적 서브트리 순회 (설정 가능한 주기) |
| 트랩 수신 | 인바운드 SNMP 트랩 처리 |

**SNMP 포트**: 161, **트랩 포트**: 1162

### 3.5 데이터 저장소

| 기능 | 설명 |
|------|------|
| PostgreSQL | 모든 프로토콜 데이터 영구 저장 |
| 테이블 분리 | 프로토콜별 전용 테이블 운영 |
| 커넥션 풀링 | 25개 최대 연결, 5분 수명 |
| 자동 모니터링 | 10초 간격 커넥션 상태 체크 |

**테이블 구조**:
- `client_data`: HTTP REST 요청 데이터
- `mqtt_messages`: MQTT 메시지
- `coap_messages`: CoAP 메시지
- `snmp_data`: SNMP 연산 결과

### 3.6 Kafka 메시지 발행

| 기능 | 설명 |
|------|------|
| 토픽 정제 | URL 경로 → Kafka 토픽명 변환 |
| 자동 프리픽스 | `bz.` 접두사 자동 추가 |
| 비동기 발행 | 콜백 기반 5초 타임아웃 |
| 자동 토픽 생성 | 존재하지 않는 토픽 자동 생성 |

**토픽 정제 규칙**:
- `/api/users` → `bz.api.users`
- `/hands/left` → `bz.hands.left`

### 3.7 모니터링 (Prometheus)

| 메트릭 | 설명 |
|--------|------|
| `servone_http_requests_total` | HTTP 요청 카운터 (메서드, 엔드포인트, 상태별) |
| `servone_http_request_duration_seconds` | HTTP 요청 지연 히스토그램 |
| `servone_coap_requests_total` | CoAP 요청 카운터 |
| `servone_mqtt_messages_received_total` | MQTT 메시지 수신 카운터 |
| `servone_snmp_operations_total` | SNMP 연산 카운터 |
| `servone_db_*` | 데이터베이스 연결 및 연산 메트릭 |
| `servone_kafka_*` | Kafka 발행 메트릭 |

---

## 4. 기술 사양

### 4.1 시스템 요구사항

| 항목 | 사양 |
|------|------|
| 언어 | Go 1.24+ |
| OS | Linux, macOS, Windows |
| 메모리 | 최소 512MB (권장 2GB) |
| 네트워크 | HTTP(8090), CoAP(5683), SNMP Trap(1162) |

### 4.2 외부 의존성

| 서비스 | 버전 | 용도 |
|--------|------|------|
| PostgreSQL | 12+ | 데이터 저장 |
| Apache Kafka | 2.8+ | 메시지 스트리밍 |
| MQTT Broker | Mosquitto 등 | MQTT 메시지 브로커 |
| Prometheus | 2.x | 메트릭 수집 |

### 4.3 주요 라이브러리

| 라이브러리 | 용도 |
|------------|------|
| gorilla/mux | HTTP 라우팅 |
| plgd-dev/go-coap/v3 | CoAP 프로토콜 |
| eclipse/paho.mqtt.golang | MQTT 클라이언트 |
| gosnmp/gosnmp | SNMP 클라이언트 |
| twmb/franz-go | Kafka 클라이언트 |
| lib/pq | PostgreSQL 드라이버 |
| prometheus/client_golang | Prometheus 메트릭 |
| fsnotify/fsnotify | 파일 감시 |

### 4.4 아키텍처 패턴

1. **의존성 주입 (Dependency Injection)**
   - 모든 서비스는 생성자를 통해 의존성 주입
   - 전역 변수 사용 금지

2. **인터페이스 기반 설계**
   - `KafkaPublisherInterface` 등 외부 서비스 추상화
   - 테스트 시 모킹 용이

3. **Hot Reload 패턴**
   - `Reloadable` 인터페이스 구현
   - 설정 변경 시 무중단 적용

4. **Graceful Shutdown**
   - SIGINT/SIGTERM 시그널 처리
   - 30초 타임아웃 내 리소스 정리

---

## 5. 비기능적 요구사항

### 5.1 성능
- HTTP 요청 처리: < 100ms (P95)
- CoAP 메시지 처리: < 50ms (P95)
- Kafka 발행 타임아웃: 5초
- 데이터베이스 쿼리 타임아웃: 5초

### 5.2 확장성
- 수평 확장을 위한 상태 비저장 설계
- 비동기 I/O를 통한 동시 처리
- 커넥션 풀링을 통한 리소스 효율화

### 5.3 가용성
- Graceful Shutdown 지원
- 설정 핫 리로드를 통한 무중단 운영
- 연결 자동 재시도 (MQTT, DB)

### 5.4 관측성
- 구조화된 JSON 로깅
- Prometheus 메트릭 노출
- 프로토콜별 요청 추적

### 5.5 보안
- SNMP v3 인증/암호화 지원
- PostgreSQL SSL 연결 지원
- 요청 본문 크기 제한 (10MB)

---

## 6. 데이터 플로우

```
┌──────────────────────────────────────────────────────────────┐
│                        클라이언트 계층                         │
├─────────────┬─────────────┬─────────────┬────────────────────┤
│ HTTP Client │ CoAP Device │ MQTT Broker │ SNMP Agent/Trap    │
└──────┬──────┴──────┬──────┴──────┬──────┴─────────┬──────────┘
       │             │             │                │
       ▼             ▼             ▼                ▼
┌──────────────────────────────────────────────────────────────┐
│                        Servone 서버                           │
├─────────────┬─────────────┬─────────────┬────────────────────┤
│ HTTP Server │ CoAP Server │ MQTT Client │ SNMP Client/Trap   │
│ (Port 8090) │ (Port 5683) │ (Subscribe) │ (Port 161/1162)    │
└──────┬──────┴──────┬──────┴──────┬──────┴─────────┬──────────┘
       │             │             │                │
       └─────────────┴─────────────┴────────────────┘
                            │
                ┌───────────┴───────────┐
                │                       │
                ▼                       ▼
        ┌──────────────┐       ┌──────────────┐
        │  PostgreSQL  │       │    Kafka     │
        │  (저장소)     │       │  (스트리밍)   │
        └──────────────┘       └──────────────┘
                │
                ▼
        ┌──────────────┐
        │  Prometheus  │
        │  (모니터링)   │
        └──────────────┘
```

---

## 7. 배포 옵션

### 7.1 servfull (전체 기능)
모든 프로토콜 핸들러 포함:
- HTTP REST API
- CoAP 서버
- MQTT 클라이언트
- SNMP 클라이언트/트랩 서버

```bash
go run ./cmd/servfull/main.go
```

### 7.2 servrest (경량 버전)
HTTP REST API만 포함:
- HTTP REST API
- Kafka 발행

```bash
go run ./cmd/servrest/main.go
```

---

## 8. 설정 구조

```yaml
database:
  connection_string: "postgresql://user:pass@localhost:5432/db?sslmode=disable"

kafka:
  brokers:
    - "localhost:9092"

mqtt:
  broker: "tcp://localhost:1883"
  client_id: "servone-mqtt"

snmp:
  port: 161
  timeout: 5
  retries: 3
  username: "user"
  auth_protocol: "SHA"
  auth_passphrase: "auth_password"
  priv_protocol: "AES"
  priv_passphrase: "priv_password"
  term: 5  # WALK 주기 (초)
  targets:
    - "192.168.1.1"
  root_oid: "1.3.6.1.2.1.1"

snmptrap:
  enabled: true
  host: "0.0.0.0"
  port: 1162

rest:
  host: "0.0.0.0"
  port: "8090"
  endpoints:
    - path: "/api/health"
      method: "GET"
      response:
        status: 200
        body: '{"status": "ok", "timestamp": "{{.timestamp}}"}'
        headers:
          Content-Type: "application/json"

coap:
  host: "0.0.0.0"
  port: "5683"
  endpoints:
    - path: "/sensor/data"
      method: "POST"
      response:
        status: 201
```

---

## 9. 버전 히스토리

| 버전 | 날짜 | 주요 변경사항 |
|------|------|--------------|
| 1.0.0 | 현재 | 초기 릴리스 - HTTP, CoAP, MQTT, SNMP 통합 |

---

## 10. 용어 정의

| 용어 | 정의 |
|------|------|
| CoAP | Constrained Application Protocol - 제한된 네트워크용 경량 프로토콜 |
| MQTT | Message Queuing Telemetry Transport - IoT용 경량 메시징 프로토콜 |
| SNMP | Simple Network Management Protocol - 네트워크 장비 관리 프로토콜 |
| USM | User-based Security Model - SNMP v3 보안 모델 |
| Hot Reload | 서버 재시작 없이 설정 변경을 적용하는 기능 |
| Topic Sanitization | URL 경로를 Kafka 토픽명으로 변환하는 과정 |
