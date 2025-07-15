`snmpclient` 패키지는 SNMP 통신을 위한 두 가지 주요 기능을 제공합니다: 주기적인 SNMP WALK 스케줄러와 독립적인 SNMP TRAP 수신 서버입니다. 기존의 HTTP 요청 기반 WALK 기능은 제거되었으며, 이제 설정 파일을 통해 정의된 일정에 따라 자동으로 실행됩니다. 모든 SNMP 데이터(GET, WALK, TRAP)는 처리 후 Kafka로 발행되고 `snmp_data` 데이터베이스 테이블에 저장됩니다.

### 아키텍처 변경 사항

기존의 단일 `SNMPClient` 구조체는 두 개의 독립적인 구성 요소로 분리되었습니다:

1.  **`SNMPClient`**: 주기적인 WALK 작업과 HTTP API를 통한 GET 요청을 담당합니다.
2.  **`TrapServer`**: SNMP TRAP 메시지 수신 및 처리를 전담합니다.

이러한 분리를 통해 WALK 스케줄러와 TRAP 서버는 독립적으로 설정되고 실행될 수 있습니다.

### 주요 구성 요소 및 역할

1.  **`SNMPClient` (snmp.go)**:
    *   주기적인 SNMP WALK 작업을 수행하고, HTTP 기반 GET 요청을 처리하는 클라이언트입니다.
    *   **`NewSNMPClient`**: `SNMPClient` 인스턴스를 생성합니다.
    *   **`StartWalkScheduler`**: `config.yaml`의 `snmp.term`에 지정된 시간(초) 간격으로 주기적인 WALK를 실행하는 스케줄러를 시작합니다. `term`이 0 이하이면 스케줄러는 비활성화됩니다.
    *   **`GetV3`**: 특정 OID에 대한 값을 가져오는 SNMP GET 요청을 수행합니다. 이 기능은 여전히 `/api/snmp/get` HTTP 엔드포인트를 통해 외부에서 호출할 수 있습니다.
    *   **`WalkV3`**: 특정 OID를 시작으로 하위 모든 OID를 순회하며 값을 가져오는 SNMP WALK 요청을 수행합니다. 이 함수는 이제 `StartWalkScheduler`에 의해서만 내부적으로 호출됩니다.
    *   **`Stop`**: WALK 스케줄러를 정상적으로 종료합니다.

2.  **`TrapServer` (snmp.go)**:
    *   SNMP TRAP 메시지를 수신하고 처리하는 독립적인 서버입니다.
    *   **`NewTrapServer`**: `TrapServer` 인스턴스를 생성합니다. `config.yaml`의 `snmptrap` 섹션에서 설정을 읽습니다.
    *   **`Start`**: TRAP 리스너를 시작합니다. 설정에서 `enabled: true`일 경우에만 활성화됩니다.
    *   **`handleTrap`**: 수신된 TRAP 메시지를 처리하고 `processResults`로 전달합니다.
    *   **`Stop`**: TRAP 리스너를 정상적으로 종료합니다.

<!-- 3.  **`SNMPHandler` (api.go)**:
    *   HTTP 요청을 받아 SNMP GET 작업을 수행하는 핸들러입니다.
    *   **`HandleWalk`** 함수와 `/api/snmp/walk` 경로는 **제거되었습니다.**
    *   **`HandleGet`** 함수는 이전과 동일하게 작동하며, `/api/snmp/get` 엔드포인트를 통해 GET 요청을 처리합니다. -->

4.  **`processResults` (snmp.go 내의 private 함수)**:
    *   모든 SNMP 작업(GET, WALK, TRAP)의 결과를 처리하는 중앙 집중식 함수입니다.
    *   수신된 데이터를 Kafka와 데이터베이스에 순차적으로 저장합니다.
    *   **데이터 처리 순서**:
        1.  결과 데이터를 Kafka의 해당 토픽(예: `snmp.walk.<target>`)으로 발행합니다.
        2.  Kafka 발행 성공 여부와 관계없이, `saveSNMPDataToDB` 함수를 호출하여 동일한 데이터를 `snmp_data` 데이터베이스 테이블에 저장합니다.

### 새로운 전체적인 플로우

1.  **초기화 (Application Startup)**:
    *   애플리케이션이 시작될 때, `config.yaml`에서 `snmp`와 `snmptrap` 설정을 로드합니다.
    *   `kafka.NewKafkaPublisher`를 통해 Kafka 발행기 인스턴스를 생성합니다.
    *   `db.InitDB`를 통해 데이터베이스 연결 풀을 초기화합니다.
    *   `snmpclient.NewSNMPClient`를 호출하여 `SNMPClient` 인스턴스를 생성합니다.
    *   `snmpClient.StartWalkScheduler()`를 호출하여 백그라운드에서 주기적인 WALK 작업을 시작합니다.
    *   `snmpclient.NewTrapServer`를 호출하여 `TrapServer` 인스턴스를 생성합니다.
    *   `trapServer.Start()`를 호출하여 백그라운드에서 TRAP 리스너를 시작합니다.
    *   `SNMPHandler`는 `/api/snmp/get` 경로만 HTTP 서버에 등록합니다.

2.  **주기적인 SNMP WALK 처리 플로우 (자동 실행)**:
    *   `StartWalkScheduler`에서 생성된 타이머가 `snmp.term` 초마다 신호를 보냅니다.
    *   타이머 신호가 발생하면, 스케줄러는 `snmp.targets` 설정에 있는 모든 대상에 대해 `WalkV3` 함수를 순차적으로 호출합니다.
    *   `WalkV3`는 대상 장비에 SNMP WALK 요청을 보냅니다.
    *   WALK 작업이 완료되면, `processResults` 함수를 호출하여 결과를 처리합니다.
    *   `processResults`는 다음을 수행합니다:
        1.  WALK 결과를 Kafka의 `snmp.walk.<target>` 토픽으로 발행합니다.
        2.  WALK 결과를 `snmp_data` 테이블에 저장합니다. `host` 필드에는 대상 장비의 주소가, `data` 필드에는 전체 결과가 JSON 형식으로 저장됩니다.

3.  **SNMP TRAP 수신 처리 플로우**:
    *   `TrapServer`는 `snmptrap.host`와 `snmptrap.port`에서 UDP 트래픽을 리스닝���니다.
    *   네트워크 장비에서 SNMP TRAP 메시지를 보내면, `handleTrap` 콜백 함수가 이를 수신합니다.
    *   `handleTrap`은 `processResults` 함수를 호출하여 TRAP 데이터를 처리합니다.
    *   `processResults`는 다음을 수행합니다:
        1.  TRAP 데이터를 Kafka의 `snmp.trap.<source_ip>` 토픽으로 발행합니다.
        2.  TRAP 데이터를 `snmp_data` 테이블에 저장합니다. `host` 필드에는 TRAP을 보낸 장비의 IP 주소가, `data` 필드에는 전체 결과가 JSON 형식으로 저장됩니다.

<!-- 4.  **SNMP GET 요청 처리 플로우 (HTTP 요청 기반)**:
    *   이 플로우는 이전과 거의 동일하게 유지됩니다.
    *   클라이언트가 `/api/snmp/get` 엔드포인트로 `target`과 `oids`를 포함한 POST 요청을 보냅니다.
    *   `SNMPHandler.HandleGet` 함수가 요청을 받아 `SNMPClient.GetV3`를 호출합니다.
    *   `GetV3`가 장비로부터 응답을 받으면, `processResults` 함수를 호출하여 결과를 처리합니다.
    *   `processResults`는 GET 결과를 Kafka에 발행하고 데이터베이스에 저장한 후, HTTP 핸들러는 클라이언트에게 성공 응답을 반환합니다. -->

### 요약

`snmpclient` 패키지는 이제 HTTP API와 완전히 분리된, 설정 기반의 자동화된 SNMP WALK 기능을 제공합니다. TRAP 수신 기능 또한 별도의 서버로 분리되어 명확성과 확장성이 향상되었습니다. 모든 종류의 SNMP 데이터는 Kafka 발행과 데이터베이스 저장을 통해 이중으로 기록되어 데이터 파이프라인의 안정성을 높입니다.
