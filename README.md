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
