# 소스 코드 개선 제안 사항 (2025-12-20)

현재 프로젝트 `servone`의 소스 코드를 검토한 결과, 다음과 같은 개선 사항을 제안합니다.

## 1. 데이터베이스 의존성 주입(Dependency Injection) 완료
*   **현재 상태:** `db` 패키지에 전역 변수 `DbPool`과 메서드 기반의 `Database` 구조체가 혼재되어 있습니다. `server.go`는 여전히 전역 함수 `db.SaveToDBWithError`를 사용하고 있어 `InitDB` 호출 순서에 강하게 의존합니다.
*   **제안:** `server.go`의 `DynamicServer`가 `*db.Database` 인스턴스를 필드로 가지도록 변경하고, 전역 함수 대신 메서드(`ds.db.SaveToDBWithError`)를 호출하도록 수정하세요. 이를 통해 테스트 시 모의 객체(Mock) 사용이 쉬워지고 전역 상태 의존성을 제거할 수 있습니다.

## 2. 백그라운드 작업의 우아한 종료 (Graceful Shutdown)
*   **현재 상태:** `server.go`의 `createHandler`에서 `go func() { ... }`를 사용하여 DB 저장을 별도 고루틴으로 실행합니다(Fire-and-forget). 서버 종료 시 이 작업들이 완료되지 않고 강제 종료될 위험이 있습니다.
*   **제안:** `sync.WaitGroup`을 사용하여 백그라운드 작업의 시작과 종료를 추적하고, 서버 종료(`Shutdown`) 시 모든 DB 저장 작업이 완료될 때까지 기다리도록 구현해야 합니다.

## 3. 스레드 안전한(Thread-Safe) 라우터 리로드
*   **현재 상태:** `Reload` 메서드에서 `ds.server.Handler = ds.router`를 직접 수정하고 있습니다. Go의 `http.Server`는 실행 중에 `Handler`를 변경하는 것이 스레드 안전하지 않습니다(Race Condition 발생 가능).
*   **제안:** `ServeHTTP` 메서드를 가진 래퍼(Wrapper) 구조체를 만들고, 그 내부에서 `RWMutex`를 사용하여 라우터를 안전하게 교체하는 방식으로 변경해야 합니다.

## 4. 구조화된 로깅 (Structured Logging) 도입
*   **현재 상태:** 표준 `log` 패키지를 사용하며, 일부는 `json.Marshal`을 이용해 수동으로 JSON 로그를 만들고 있습니다.
*   **제안:** Go 1.21부터 도입된 `log/slog` 패키지를 사용하여 일관된 JSON 구조화 로깅을 적용하세요. 로그 파싱과 검색이 훨씬 용이해집니다.

## 5. 하드코딩된 설정의 외부화
*   **현재 상태:** DB 연결 풀 설정(`MaxOpenConns: 25` 등)과 HTTP 서버 타임아웃(`30s`)이 코드 내에 하드코딩되어 있습니다.
*   **제안:** `config.yaml` 및 `config.go`에 해당 설정 항목을 추가하여 환경에 맞게 튜닝할 수 있도록 변경하세요.

## 6. HTTP 에러 처리 강화
*   **현재 상태:** `createHandler`에서 `json.Unmarshal` 실패 시 에러 로그만 남기고, 클라이언트에게는 400 Bad Request 등을 반환하지 않을 수 있습니다.
*   **제안:** 잘못된 요청 본문(JSON 파싱 실패)이 들어오면 명확하게 `http.Error(w, "Invalid JSON", http.StatusBadRequest)`를 반환하도록 수정하세요.
