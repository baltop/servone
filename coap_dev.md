# CoAP 서버 개발 및 테스트 가이드

이 문서는 `coap.go`를 중심으로 CoAP 서버 기능을 개발하고 테스트할 때의 주요 원칙과 주의사항을 정리한 것입니다.

## 1. 핵심 아키텍처 원칙

### 의존성 주입 (Dependency Injection)

우리 서버의 핵심 원칙 중 하나는 의존성 주입입니다. `CoapServer`나 `DynamicServer`가 데이터베이스 커넥션 풀(`*sql.DB`)이나 Kafka 발행기(`KafkaPublisherInterface`) 같은 외부 서비스에 직접 의존하지 않고, 생성자를 통해 외부에서 주입받습니다.

**이유:**
- **테스트 용이성:** 실제 DB나 Kafka 클라이언트 대신, 테스트 코드에서 모의(Mock) 객체를 쉽게 주입할 수 있습니다. 이를 통해 외부 서비스의 상태와 관계없이 독립적이고 안정적인 단위 테스트가 가능해집니다.
- **유연성 및 확장성:** 나중에 다른 종류의 데이터베이스나 메시지 큐로 교체하더라도, 해당 서비스의 클라이언트를 생성해서 주입하기만 하면 되므로 코드 변경을 최소화할 수 있습니다.

**주의사항:**
- 새로운 의존성이 필요할 경우, 전��� 변수로 선언하지 말고 반드시 서버 구조체에 필드를 추가하고 생성자를 통해 주입하는 패턴을 따라야 합니다.

## 2. CoAP 핸들러 개발 시 주의사항

### `go-coap` 라이브러리 특성

- **요청 코드와 응답 코드:** CoAP 프로토콜에서는 클라이언트의 요청 메서드 코드(예: `POST` = `0.02`)와 서버의 성공 응답 코드(예: `Content` = `2.05`)가 다릅니다. 핸들러 내부에서 요청 메서드를 확인할 때는 `r.Code()`를 사용하고, 응답을 보낼 때는 `codes.Content`, `codes.Changed` 등과 같은 적절한 성공 코드를 사용해야 합니다.
- **상태 코드 설정:** 응답 코드를 설정할 때 `w.SetResponse(codes.Code(205), ...)`와 같이 정수 값을 `codes.Code`로 변환하여 사용합니다. 테스트 코드에서 응답 코드를 검증할 때도 이와 동일한 형태로 비교해야 합니다. (`assert.Equal(t, codes.Code(205), resp.Code())`)

## 3. 테스트 코드 작성 가이드

### 비동기 코드 테스트

- **`time.Sleep` 사용 금지:** CoAP 핸들러 내의 `saveToDB` 함수는 `go func() { ... }()` 형태로 비동기 호출됩니다. 이런 비동기 작업이 완료되기를 기다리기 위해 `time.Sleep()`을 사용하는 것은 테스트를 불안정하게 만듭니다. 실행 환경의 성능에 따라 ���스트가 실패할 수 있습니다.
- **채널(Channel) 사용:** 비동기 작업의 완료를 보장하기 위해 채널을 사용해야 합니다. `coap_test.go`의 `MockKafkaPublisher`는 `Publish` 함수가 호출되면 채널에 신호를 보냅니다. 테스트 코드는 이 채널로부터 신호가 올 때까지 대기함으로써, 비동기 작업이 완료되었음을 명확하게 확인한 후 검증을 수행합니다.

```go
// coap_test.go 예시
publishedChan := make(chan struct{}, 1)
mockPublisher := &MockKafkaPublisher{
    PublishFunc: func(...) {
        // ...
        publishedChan <- struct{}{} // 작업 완료 신호
    },
}

// ... 테스트 로직 ...

// 채널을 통해 대기
select {
case <-publishedChan:
    // 검증 로직
case <-time.After(2 * time.Second):
    t.Fatal("timed out")
}
```

### 데이터베이스 테스트

- `database_test.go`의 `setupTestDB` 함수는 테스트를 위한 격리된 DB 환경을 설정합니다.
- `t.Cleanup`을 사용하여 테스트가 끝난 후 `TRUNCATE` 명령으로 테이블 데이터를 삭제함으로써 각 테스트가 서로에게 영향을 주지 않도록 보장합니다.

## 4. Kafka 연동 시 주의사항

### 토픽 이름 규칙 (Topic Sanitization)

- Kafka 토픽 이름에는 사용할 수 있는 문자에 제약이 있습니다. URL 경로(`/`)와 같은 특수 문자는 허용되지 않습니다.
- `kafka.go`의 `sanitizeTopic(topic string)` 함수는 URL 경로와 같은 문자열을 Kafka 토픽 이름 규칙에 맞게 변환하고, 일관성을 위해 `bz.` 접두사를 붙입니다.
- **매우 중요:** Kafka로 메시지를 발행(`publisher.Publish`)하는 모든 코드에서는, 토픽 이름을 전달하기 전에 **반드시** `sanitizeTopic()` 함수를 호출해야 합니다. 이를 누락하면 Kafka 클라이언트에서 오류가 발생할 수 있습니다.
