 config.yaml에 정의된 대로 동적으로 엔드포인트가 생성되는 REST API 서버가 완성되었습니다.

  주요 기능:
  - config.yaml 파일에서 엔드포인트 설정을 읽어와 동적으로 라우트 생성
  - 파일 변경 감지로 실시간 설정 리로딩 (500ms 디바운스)
  - 경로 파라미터 지원 (/api/products/{id})
  - 템플릿 기반 응답 생성 (타임스탬프, 파라미터 값 등)
  - HTTP 메서드, 상태 코드, 헤더 설정 가능

  사용 방법:
  1. go mod download - 의존성 다운로드
  2. go run . - 서버 실행
  3. config.yaml 수정 시 자동으로 새 엔드포인트 적용

  테스트 엔드포인트:
  - GET /api/users - 사용자 목록
  - POST /api/users - 사용자 생성
  - GET /api/health - 서비스 상태 (현재 시간 포함)
  - GET /api/products/{id} - 특정 상품 조회

  config.yaml을 수정하면 새로운 엔드포인트가 즉시 반영됩니다.



----------

  db/database.go 파일의 끝에 다음 함수를 추가하여, 전역 DB 연결 풀(DbPool)을 사용해 Database 인스턴스를 생성하고 반환하도록 합니다.



   1 // GetDBInstance returns a new Database instance using the global DbPool.
   2 // This is a helper function for the transition to dependency injection.
   3 func GetDBInstance() *Database {
   4 if DbPool == nil {
   5 return nil
   6    }
   7 return &Database{db: DbPool}
   8 }



  2단계: coap/coap.go의 핸들러 수정


  이제 coap/coap.go 파일의 createHandler 함수 내부에 있는 go func() { ... }() 부분을 아래 코드로 교체합니다. 이 코드는 위에서 추가한 GetDBInstance() 함수를 사용하여 DB 인스턴스를 얻은 후, SaveCoapMessage
  메서드를 호출합니다.

  교체할 `coap.go` 코드:



    1 // coap/coap.go 의 createHandler 함수 내부
    2 // Save to database and publish to Kafka
    3 go func() {
    4                                   receivedTime := time.Now().UnixNano()
    5 
    6 // Get database instance and save message
    7 if dbInstance := db.GetDBInstance(); dbInstance != nil {
    8 if err := dbInstance.SaveCoapMessage(endpoint.Path, string(bodyBytes), r.Code().String(), receivedTime); err != nil {
    9                                                   log."Failed to save CoAP message to database: %v", err)
   10                                           }
   11           else {
   12                                           log.P"Failed to get database instance. Message not saved.")
   13                                   }
   14 
   15 // Publish to Kafka
   16                                   kafkaPayloamap[string]interface{}{
   17 "path":     endpoint.Path,
   18 "method":   r.Code().String(),
   19 "data":     jsonData,
   20 "received": receivedTime,
   21                                   }
   22 if err := cs.publisher.Publish("coap."+endpoint.Path, kafkaPayload); err != nil {
   23                                           log.P"Failed to publish CoAP message to Kafka: %v", err)
   24                                   }
   25                           }()



  이 두 가지를 적용하면 CoAP 메시지를 수신했을 때 (d *Database) SaveCoapMessage(...) 메서드가 정상적으로 호출되어 DB에 데이터가 저장될 것입니다.

----------------





postgres에서 mqtt payload 꺼낼때

SELECT id, topic, encode(payload, 'escape')::text , created_at FROM public.mqtt_messages
ORDER BY id DESC LIMIT 100


        docker kill -s SIGHUP [prometheus_container]


echo -n 'hello world' | coap post coap://localhost/hands/left        
