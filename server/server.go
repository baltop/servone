// 동적으로 라우트와 응답을 처리하는 HTTP 서버 구현
package server

// 필요한 패키지 임포트
import (
	"bytes"         // 바이트 버퍼 및 변환
	"context"       // 컨텍스트 제어
	"encoding/json" // JSON 파싱 및 인코딩
	"io"            // 요청 바디 읽기
	"log"           // 로그 출력
	"net/http"      // HTTP 서버
	"regexp"        // 정규표현식
	"servone/config"
	"servone/db"
	"servone/kafka" // Kafka 퍼블리셔 인터페이스
	"strconv"       // 문자열-숫자 변환
	"strings"       // 문자열 처리
	"sync"          // 동기화
	"text/template" // 템플릿 처리
	"time"          // 시간 관련

	"servone/snmpclient"

	"github.com/gorilla/mux" // 라우터
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// 동적으로 라우트와 응답을 처리하는 서버 구조체
type DynamicServer struct {
	config      *config.Config            // 현재 서버 설정
	router      *mux.Router               // HTTP 라우터
	server      *http.Server              // HTTP 서버 인스턴스
	pathVars    map[string]*regexp.Regexp // 경로 변수에 대한 정규표현식 매핑
	publisher   kafka.KafkaPublisherInterface
	templates   map[string]*template.Template // 템플릿 캐시
	templateMux sync.RWMutex                  // 템플릿 캐시 동기화
	configMux   sync.RWMutex                  // 설정 변경 동기화
}

// DynamicServer 생성자 함수
// config: 서버 설정 구조체
func NewDynamicServer(cfg *config.Config, publisher kafka.KafkaPublisherInterface) *DynamicServer {
	ds := &DynamicServer{
		config:    cfg,
		router:    mux.NewRouter(), // 새로운 라우터 생성
		pathVars:  make(map[string]*regexp.Regexp),
		publisher: publisher,
		templates: make(map[string]*template.Template),
	}

	ds.setupRoutes() // 라우트 설정

	ds.server = &http.Server{
		Addr:         ds.config.Rest.Host + ":" + ds.config.Rest.Port, // 서버 주소 및 포트 지정
		Handler:      ds.router,                                       // 라우터를 핸들러로 지정
		ReadTimeout:  30 * time.Second,                                // 읽기 타임아웃
		WriteTimeout: 30 * time.Second,                                // 쓰기 타임아웃
		IdleTimeout:  120 * time.Second,                               // 유휴 타임아웃
	}

	return ds
}

// 설정에 정의된 모든 엔드포인트를 라우터에 등록
func (ds *DynamicServer) setupRoutes() {
	for _, endpoint := range ds.config.Rest.Endpoints {
		ds.addRoute(endpoint)
	}
	ds.router.Handle("/metrics", promhttp.Handler()).Methods("GET")
	log.Printf("Added route: GET /metrics")
}

// 단일 엔드포인트를 라우터에 등록하는 함수
func (ds *DynamicServer) addRoute(endpoint config.EndpointConfig) {
	path := endpoint.Path                      // 엔드포인트 경로
	method := strings.ToUpper(endpoint.Method) // HTTP 메서드 대문자화

	handler := ds.createHandler(endpoint) // 핸들러 함수 생성

	// 라우트 등록
	ds.router.HandleFunc(path, handler).Methods(method)

	log.Printf("Added route: %s %s", method, path) // 라우트 등록 로그
}

// 엔드포인트별 요청을 처리하는 핸들러 함수 생성
func (ds *DynamicServer) createHandler(endpoint config.EndpointConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r) // URL 경로 변수 추출

		var bodyBytes []byte
		var err error
		if r.Body != nil {
			// 요청 크기 제한 (10MB)
			r.Body = http.MaxBytesReader(w, r.Body, 10*1024*1024)
			
			bodyBytes, err = io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
				return
			}
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes)) // 바디 복원(다중 사용 가능하게)
		}

		// Check if this is an SNMP endpoint
		if strings.HasPrefix(endpoint.Path, "/api/snmp/") && r.Method == "POST" {
			// Handle SNMP operations
			if endpoint.Path == "/api/snmp/get" {
				snmpclient.HandleSNMPRequest(w, r, "get")
				return
			} else if endpoint.Path == "/api/snmp/walk" {
				snmpclient.HandleSNMPRequest(w, r, "walk")
				return
			}
		}

		// POST 요청이면서 JSON 바디가 있을 때 구조화된 로그 출력
		if r.Method == "POST" && len(bodyBytes) > 0 {
			var jsonData map[string]interface{}
			if err := json.Unmarshal(bodyBytes, &jsonData); err == nil {
				logPayload := make(map[string]interface{})
				logPayload["url"] = endpoint.Path
				logPayload["data"] = jsonData
				for k, v := range vars {
					logPayload[k] = v
				}
				if logBytes, logErr := json.Marshal(logPayload); logErr == nil {
					log.Printf("Request Log: %s", string(logBytes)) // JSON 형태로 로그 출력
				}

				// 데이터베이스에 저장 (에러 로깅 추가)
				go func() {
					// Only save to database if DbPool is initialized
					if db.DbPool != nil {
						if err := db.SaveToDBWithError(endpoint.Path, jsonData, vars, ds.publisher); err != nil {
							log.Printf("Failed to save to database: %v", err)
						}
					}
				}()

			} else {
				// JSON 파싱 실패 시 일반 텍스트로 로그
				log.Printf("%s %s - %d | Request body: %s", r.Method, r.URL.Path, endpoint.Response.Status, string(bodyBytes))
			}
		} else if len(bodyBytes) > 0 {
			// POST 외의 요청에서 바디가 있을 때 로그
			log.Printf("%s %s - %d | Request body: %s", r.Method, r.URL.Path, endpoint.Response.Status, string(bodyBytes))
		} else {
			// 바디가 없는 요청 로그
			log.Printf("%s %s - %d", r.Method, r.URL.Path, endpoint.Response.Status)
		}

		// 설정에 정의된 헤더를 응답에 추가
		for key, value := range endpoint.Response.Headers {
			w.Header().Set(key, value)
		}

		w.WriteHeader(endpoint.Response.Status) // 응답 상태 코드 설정

		// 템플릿 처리 후 응답 본문 작성
		body := ds.processTemplate(endpoint.Response.Body, vars)
		w.Write([]byte(body))
	}
}

// 응답 본문에 템플릿({{변수}})이 있을 경우 변수 치환 처리
func (ds *DynamicServer) processTemplate(body string, vars map[string]string) string {
	if !strings.Contains(body, "{{") {
		return body // 템플릿이 없으면 그대로 반환
	}

	// 템플릿 캐시 확인
	ds.templateMux.RLock()
	tmpl, exists := ds.templates[body]
	ds.templateMux.RUnlock()

	if !exists {
		// 캐시에 없으면 파싱하고 저장
		var err error
		tmpl, err = template.New("response").Parse(body)
		if err != nil {
			log.Printf("Template parse error: %v", err)
			return body
		}

		ds.templateMux.Lock()
		ds.templates[body] = tmpl
		ds.templateMux.Unlock()
	}

	data := make(map[string]interface{})
	for k, v := range vars {
		// 숫자형 변수는 int로 변환하여 템플릿에 전달
		if id, err := strconv.Atoi(v); err == nil {
			data[k] = id
		} else {
			data[k] = v
		}
	}

	data["timestamp"] = time.Now().UTC().Format(time.RFC3339) // 현재 UTC 타임스탬프 추가

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil { // 템플릿 실행
		log.Printf("Template execute error: %v", err)
		return body
	}

	return buf.String() // 치환된 결과 반환
}

// 서버를 시작하는 함수
func (ds *DynamicServer) Start() error {
	log.Printf("Starting server on %s", ds.server.Addr) // 서버 시작 로그
	return ds.server.ListenAndServe()                   // HTTP 서버 실행
}

// 설정 변경 시 서버 라우트 및 핸들러를 재설정하는 함수
func (ds *DynamicServer) Reload(newConfig *config.Config) {
	log.Println("Reloading server configuration...") // 재로드 시작 로그

	ds.configMux.Lock()
	defer ds.configMux.Unlock()

	ds.config = newConfig                         // 새로운 설정 반영
	newRouter := mux.NewRouter()                  // 새 라우터 생성
	ds.pathVars = make(map[string]*regexp.Regexp) // 경로 변수 맵 초기화

	// 임시 라우터에 라우트 설정
	ds.router = newRouter
	ds.setupRoutes()              // 라우트 재설정
	ds.server.Handler = ds.router // 서버 핸들러 갱신

	// 템플릿 캐시 초기화
	ds.templateMux.Lock()
	ds.templates = make(map[string]*template.Template)
	ds.templateMux.Unlock()

	log.Println("Server configuration reloaded successfully") // 재로드 완료 로그
}

// Shutdown gracefully shuts down the server
func (ds *DynamicServer) Shutdown(ctx context.Context) error {
	return ds.server.Shutdown(ctx)
}
