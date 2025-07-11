package coap

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"sync"

	"servone/config"
	"servone/db"
	"servone/kafka"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/plgd-dev/go-coap/v3"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/mux"
)

// CoapServer 구조체는 CoAP 서버의 상태를 관리합니다.
type CoapServer struct {
	config    *config.Config
	router    *mux.Router
	publisher kafka.KafkaPublisherInterface
	stopChan  chan struct{}
	listening bool
	mu        sync.Mutex
}

// NewCoapServer는 새로운 CoapServer 인스턴스를 생성하고 초기화합니다.
func NewCoapServer(cfg *config.Config, publisher kafka.KafkaPublisherInterface) *CoapServer {
	cs := &CoapServer{
		config:    cfg,
		router:    mux.NewRouter(),
		publisher: publisher,
		stopChan:  make(chan struct{}),
	}

	cs.setupRoutes()
	cs.start()

	return cs
}

// start starts the CoAP server
func (cs *CoapServer) start() {
	addr := cs.config.Coap.Host + ":" + cs.config.Coap.Port

	go func() {
		log.Printf("Starting CoAP server on %s", addr)
		cs.mu.Lock()
		cs.listening = true
		cs.mu.Unlock()

		if err := coap.ListenAndServe("udp", addr, cs.router); err != nil {
			// Check if server was stopped intentionally
			select {
			case <-cs.stopChan:
				log.Println("CoAP server stopped")
			default:
				log.Printf("CoAP server error: %v", err)
			}
		}

		cs.mu.Lock()
		cs.listening = false
		cs.mu.Unlock()
	}()
}

// setupRoutes는 설정 파일에 정의된 엔���포인트를 기반으로 CoAP 라우트를 설정합니다.
func (cs *CoapServer) setupRoutes() {
	for _, endpoint := range cs.config.Endpoints {
		cs.addRoute(endpoint)
	}
}

// addRoute는 단일 엔드포인트에 대한 CoAP 라우트를 추가합니다.
func (cs *CoapServer) addRoute(endpoint config.EndpointConfig) {
	path := endpoint.Path
	method := strings.ToUpper(endpoint.Method)

	cs.router.HandleFunc(path, cs.createHandler(endpoint, method))
	log.Printf("Added CoAP route: %s %s", method, path)
}

// createHandler는 CoAP 요청을 처리하는 핸들러 함수를 생성합니다.
func (cs *CoapServer) createHandler(endpoint config.EndpointConfig, method string) mux.HandlerFunc {
	return func(w mux.ResponseWriter, r *mux.Message) {
		if r.Code().String() != method {
			w.SetResponse(codes.MethodNotAllowed, message.TextPlain, bytes.NewReader([]byte("Method Not Allowed")))
			return
		}

		// path, err := r.Options().Path()
		// if err != nil {
		// 	log.Printf("Cannot get path: %v", err)
		// 	w.SetResponse(codes.InternalServerError, message.TextPlain, bytes.NewReader([]byte("Cannot get path")))
		// 	return
		// }

		if r.Body() == nil {
			log.Printf("CoAP %s %s - %d | No request body", r.Code(), endpoint.Path, endpoint.Response.Status)
			return
		}

		bodyBytes, err := io.ReadAll(r.Body())
		if err != nil {
			log.Printf("Cannot read body: %v", err)
			w.SetResponse(codes.InternalServerError, message.TextPlain, bytes.NewReader([]byte("Cannot read body")))
			return
		}

		if r.Code() == codes.POST && len(bodyBytes) > 0 {
			var jsonData map[string]interface{}

			// bodyBytes가 JSON 형식이 아니면 "data" 필드에 원본 바이트를 저장
			if err := json.Unmarshal(bodyBytes, &jsonData); err != nil {
				jsonData = map[string]interface{}{"coapbody": string(bodyBytes)}
			}
			// if logBytes, logErr := json.Marshal(logPayload); logErr == nil {
			// 	log.Printf("CoAP Request Log: %s", string(logBytes))
			// }
			// Save to database and publish to Kafka
			go func() {
				receivedTime := time.Now().UnixNano()
				if err := db.SaveCoapMessage(endpoint.Path, string(bodyBytes), r.Code().String(), receivedTime); err != nil {
					log.Printf("Failed to save CoAP message to database: %v", err)
				}

				// Publish to Kafka
				kafkaPayload := map[string]interface{}{
					"path":     endpoint.Path,
					"method":   r.Code().String(),
					"data":     jsonData,
					"received": receivedTime,
				}
				if err := cs.publisher.Publish("coap"+endpoint.Path, kafkaPayload); err != nil {
					log.Printf("Failed to publish CoAP message to Kafka: %v", err)
				}
			}()

		} else if len(bodyBytes) > 0 {
			log.Printf("CoAP %s %s - %d | Request body: %s", r.Code(), endpoint.Path, endpoint.Response.Status, string(bodyBytes))
		} else {
			log.Printf("CoAP %s %s - %d", r.Code(), endpoint.Path, endpoint.Response.Status)
		}

		body := cs.processTemplate(endpoint.Response.Body, nil)

		var mediaType message.MediaType = message.TextPlain
		if contentType, ok := endpoint.Response.Headers["Content-Type"]; ok {
			if strings.EqualFold(contentType, "application/json") {
				mediaType = message.AppJSON
			}
		}

		w.SetResponse(codes.Code(endpoint.Response.Status), mediaType, bytes.NewReader([]byte(body)))
	}
}

// processTemplate은 응답 본문의 템플릿을 처리합니다.
func (cs *CoapServer) processTemplate(body string, vars map[string]string) string {
	if !strings.Contains(body, "{{") {
		return body
	}

	tmpl, err := template.New("response").Parse(body)
	if err != nil {
		log.Printf("Template parse error: %v", err)
		return body
	}

	data := make(map[string]interface{})
	for k, v := range vars {
		if id, err := strconv.Atoi(v); err == nil {
			data[k] = id
		} else {
			data[k] = v
		}
	}

	data["timestamp"] = time.Now().UTC().Format(time.RFC3339)

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		log.Printf("Template execute error: %v", err)
		return body
	}

	return buf.String()
}

// Stop은 CoAP 서버를 중지합니다.
// Stop gracefully stops the CoAP server
func (cs *CoapServer) Stop() {
	log.Println("Stopping CoAP server...")
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if cs.listening {
		close(cs.stopChan)
		// Note: go-coap v3 doesn't provide a way to stop a running server
		// The server will stop when the main program exits
		cs.listening = false
	}
}

// Reload는 새로운 설정으로 CoAP 서버를 재시작합니다.
func (cs *CoapServer) Reload(newConfig *config.Config) {
	log.Println("Reloading CoAP server...")

	// Stop the current server
	cs.Stop()

	// Update configuration
	cs.config = newConfig
	cs.router = mux.NewRouter()
	cs.setupRoutes()
	cs.stopChan = make(chan struct{})

	// Start the server again
	cs.start()
}
