`snmpclient` 패키지는 SNMP(Simple Network Management Protocol) 통신을 위한 클라이언트 및 핸들러 기능을 제공합니다. 주요 기능은 SNMP GET, WALK 요청 처리 및 SNMP TRAP 수신입니다. 이 패키지는 HTTP API를 통해 외부에서 SNMP 작업을 수행할 수 있도록 하며, 결과를 Kafka로 발행하고 (주석 처리되었지만) 데이터베이스에 저장하는 기능을 포함합니다.

### 주요 구성 요소 및 역할

1.  **`SNMPClient` (snmp.go)**:
    *   SNMP 통신을 직접 수행하는 핵심 클라이언트입니다.
    *   `gosnmp` 라이브러리를 사용하여 SNMP v3 프로토콜을 지원합니다.
    *   **`NewSNMPClient`**: `SNMPClient` 인스턴스를 생성하고, 설정에 따라 TRAP 리스너를 시작합니다.
    *   **`GetV3`**: 특정 OID(Object Identifier)에 대한 값을 가져오는 SNMP GET 요청을 수행합니다.
    *   **`WalkV3`**: 특정 OID를 시작으로 하위 모든 OID를 순회하며 값을 가져오는 SNMP WALK 요청을 수행합니다.
    *   **`startTrapListener`**: SNMP TRAP 메시지를 수신하는 리스너를 시작합니다. TRAP 메시지는 네트워크 장비에서 발생하는 비동기 이벤트(예: 오류, 상태 변경)를 알리는 데 사용됩니다.
    *   **`handleTrap`**: 수신된 TRAP 메시지를 처리합니다.
    *   **`getSecurityParams`**: SNMP v3 통신에 필요한 인증 및 암호화 매개변수(사용자 이름, 비밀번호, 인증/개인 정보 보호 프로토콜)를 설정합니다.
    *   **`processResults`**: SNMP GET/WALK 결과 또는 TRAP 데이터를 처리합니다. 현재는 이 데이터를 Kafka 토픽으로 발행하는 기능을 포함하고 있습니다. (주석 처리된 부분은 데이터베이스 저장 기능)
    *   **`getValueString`**: `gosnmp.SnmpPDU`의 값을 적절한 문자열 형식으로 변환합니다.
    *   **`Stop`**: TRAP 리스너를 포함하여 클라이언트를 정상적으로 종료합니다.

2.  **`SNMPHandler` (api.go)**:
    *   HTTP 요청을 받아 SNMP 작업을 수행하는 HTTP 핸들러입니다.
    *   `SNMPClient` 인스턴스를 사용하여 실제 SNMP 통신을 수행합니다.
    *   **`NewSNMPHandler`**: `SNMPHandler` 인스턴스를 생성합니다.
    *   **`HandleGet`**: `/api/snmp/get` 엔드포인트에 대한 POST 요청을 처리합니다. 요청 본문에서 `target`(대상 장비 IP/호스트명)과 `oids`(요청할 OID 목록)를 받아 `SNMPClient.GetV3`를 호출합니다.
    *   **`HandleWalk`**: `/api/snmp/walk` 엔드포인트에 대한 POST 요청을 처리합니다. 요청 본문에서 `target`과 `root_oid`(WALK를 시작할 OID)를 받아 `SNMPClient.WalkV3`를 호출합니다.
    *   **`RegisterRoutes`**: 주어진 `http.ServeMux`에 `/api/snmp/get` 및 `/api/snmp/walk` 경로를 등록합니다.

3.  **`handler.go`**:
    *   `api.go`의 `SNMPHandler`와 유사하게 HTTP 요청을 처리하지만, `GlobalSNMPClient`라는 전역 변수를 사용하는 방식입니다.
    *   **`SetGlobalSNMPClient`**: 전역 `SNMPClient` 인스턴스를 설정합니다.
    *   **`HandleSNMPRequest`**: HTTP 요청을 받아 `operation` 매개변수(get 또는 walk)에 따라 `handleGet` 또는 `handleWalk` 함수를 호출합니다.
    *   **`handleGet`**: HTTP 요청 데이터를 파싱하여 `GlobalSNMPClient.GetV3`를 호출합니다.
    *   **`handleWalk`**: HTTP 요청 데이터를 파싱하여 `GlobalSNMPClient.WalkV3`를 호출합니다.

    *참고: `api.go`와 `handler.go`는 기능적으로 유사한 HTTP 핸들링 로직을 포함하고 있습니다. `api.go`의 `SNMPHandler`는 `SNMPClient`를 구조체 필드로 포함하여 의존성 주입 방식으로 사용하고, `handler.go`는 전역 변수를 사용합니다. 일반적으로 의존성 주입 방식이 더 권장됩니다.*

### 전체적인 플로우 (HTTP 요청 기반)

1.  **초기화 (Application Startup)**:
    *   애플리케이션이 시작될 때, `config` 패키지에서 SNMP 관련 설정(`SNMPConfig`)을 로드합니다.
    *   `kafka` 패키지의 `KafkaPublisherInterface` 구현체와 `sql.DB` 인스턴스를 준비합니다.
    *   `snmpclient.NewSNMPClient` 함수를 사용하여 `SNMPClient` 인스턴스를 생성합니다. 이때, 설정에 `TrapEnabled`가 `true`이면 SNMP TRAP 리스너가 백그라운드 고루틴으로 시작됩니다.
    *   `snmpclient.NewSNMPHandler` 함수를 사용하여 `SNMPHandler` 인스턴스를 생성하고, 이 핸들러에 위에서 생성한 `SNMPClient` 인스턴스를 주입합니다.
    *   애플리케이션의 주 HTTP 서버(`http.ServeMux`)에 `SNMPHandler.RegisterRoutes`를 호출하여 `/api/snmp/get` 및 `/api/snmp/walk` 경로를 등록합니다. (또는 `handler.go`의 `HandleSNMPRequest`를 직접 등록할 수도 있습니다.)

2.  **SNMP GET 요청 처리 플로우**:
    *   클라이언트(예: 웹 UI, 다른 서비스)가 `/api/snmp/get` 엔드포인트로 POST 요청을 보냅니다.
    *   요청 본문에는 JSON 형식으로 `{"target": "192.168.1.1", "oids": ["1.3.6.1.2.1.1.1.0", "1.3.6.1.2.1.1.5.0"]}`와 같은 데이터가 포함됩니다.
    *   `SNMPHandler.HandleGet` (또는 `handler.go`의 `handleGet`) 함수가 이 요청을 받습니다.
    *   함수는 요청 본문을 파싱하여 `target`과 `oids`를 추출합니다.
    *   추출된 정보를 사용하여 `SNMPClient.GetV3(target, oids)`를 호출합니다.
    *   `SNMPClient.GetV3`는 `gosnmp` 라이브러리를 사용하여 대상 장비에 SNMP GET 요청을 보냅니다. 이때, `SNMPConfig`에 정의된 포트, 타임아웃, 재시도 횟수, SNMP v3 보안 매개변수(인증/암호화)가 사용됩니다.
    *   장비로부터 응답을 받으면, `processResults` 함수를 호출하여 결과를 처리합니다.
    *   `processResults`는 수신된 PDU(Protocol Data Unit) 변수들을 파싱하여 `operation`(get), `source`(target), `results`(OID, 타입, 값), `timestamp`를 포함하는 맵을 생성합니다.
    *   이 맵 데이터를 `kafkaPublisher.Publish`를 통해 Kafka의 `snmp.get.<sanitized_target>` 토픽으로 발행합니다.
    *   HTTP 핸들러는 클라이언트에게 성공 또는 실패 응답을 JSON 형식으로 반환합니다.

3.  **SNMP WALK 요청 처리 플로우**:
    *   클라이언트가 `/api/snmp/walk` 엔드포인트로 POST 요청을 보냅니다.
    *   요청 본문에는 JSON 형식으로 `{"target": "192.168.1.1", "root_oid": "1.3.6.1.2.1.1"}`와 같은 데이터가 포함됩니다.
    *   `SNMPHandler.HandleWalk` (또는 `handler.go`의 `handleWalk`) 함수가 이 요청을 받습니다.
    *   함수는 요청 본문을 파싱하여 `target`과 `root_oid`를 추출합니다.
    *   추출된 정보를 사용하여 `SNMPClient.WalkV3(target, root_oid)`를 호출합니다.
    *   `SNMPClient.WalkV3`는 `gosnmp` 라이브러리를 사용하여 대상 장비에 SNMP WALK 요청을 보냅니다.
    *   WALK 작업이 완료되면, `processResults` 함수를 호출하여 결과를 처리합니다.
    *   `processResults`는 수신된 PDU 변수들을 파싱하여 `operation`(walk), `source`(target), `results`(OID, 타입, 값), `timestamp`를 포함하는 맵을 생성합니다.
    *   이 맵 데이터를 `kafkaPublisher.Publish`를 통해 Kafka의 `snmp.walk.<sanitized_target>` 토픽으로 발행합니다.
    *   HTTP 핸들러는 클라이언트에게 성공 또는 실패 응답을 JSON 형식으로 반환합니다.

4.  **SNMP TRAP 수신 처리 플로우**:
    *   `SNMPClient` 초기화 시 `TrapEnabled`가 `true`로 설정되어 있으면, `startTrapListener` 함수가 백그라운드에서 지정된 `TrapHost`와 `TrapPort`에서 UDP 트래픽을 리스닝합니다.
    *   네트워크 장비에서 SNMP TRAP 메시지를 이 리스닝 포트로 보냅니다.
    *   `trapListener.OnNewTrap`에 등록된 `handleTrap` 콜백 함수가 TRAP 메시지를 수신합니다.
    *   `handleTrap` 함수는 수신된 TRAP 패킷의 변수들을 추출하고, `processResults` 함수를 호출하여 처리합니다.
    *   `processResults`는 TRAP 데이터를 파싱하여 `operation`(trap), `source`(TRAP을 보낸 장비의 IP 주소), `results`(OID, 타입, 값), `timestamp`를 포함하는 맵을 생성합니다.
    *   이 맵 데이터를 `kafkaPublisher.Publish`를 통해 Kafka의 `snmp.trap.<sanitized_source_ip>` 토픽으로 발행합니다.

### 요약

`snmpclient` 패키지는 SNMP GET, WALK, TRAP 기능을 통합하여 제공합니다. 외부에서는 HTTP API를 통해 SNMP GET/WALK 요청을 시작할 수 있으며, 내부적으로는 SNMP TRAP을 수신 대기합니다. 모든 SNMP 통신 결과(GET, WALK, TRAP)는 Kafka 메시지 브로커로 발행되어 다른 서비스에서 실시간으로 소비하거나, (주석 처리된) 데이터베이스에 저장될 수 있습니다. 이는 네트워크 장비의 상태 모니터링 및 이벤트 처리를 위한 핵심 구성 요소로 보입니다.