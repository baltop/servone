package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"log"
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
	config    *Config
	router    *mux.Router
	publisher KafkaPublisherInterface
	db        *sql.DB
}

// NewCoapServer는 새로운 CoapServer 인스턴스를 생성하고 초기화합니다.
func NewCoapServer(config *Config, publisher KafkaPublisherInterface, db *sql.DB) *CoapServer {
	cs := &CoapServer{
		config:    config,
		router:    mux.NewRouter(),
		publisher: publisher,
		db:        db,
	}

	cs.setupRoutes()

	go func() {
		log.Printf("Starting CoAP server on %s:%s", cs.config.Coap.Host, cs.config.Coap.Port)
		if err := coap.ListenAndServe("udp", cs.config.Coap.Host+":"+cs.config.Coap.Port, cs.router); err != nil {
			log.Fatalf("Failed to start CoAP server: %v", err)
		}
	}()

	return cs
}

// setupRoutes는 설정 파일에 정의된 엔드포인트를 기반으로 CoAP 라우트를 설정합니다.
func (cs *CoapServer) setupRoutes() {
	for _, endpoint := range cs.config.Endpoints {
		cs.addRoute(endpoint)
	}
}

// addRoute는 단일 엔드포인트에 대한 CoAP 라우트를 추가합니다.
func (cs *CoapServer) addRoute(endpoint EndpointConfig) {
	path := endpoint.Path
	method := strings.ToUpper(endpoint.Method)

	cs.router.HandleFunc(path, cs.createHandler(endpoint, method))
	log.Printf("Added CoAP route: %s %s", method, path)
}

// createHandler는 CoAP 요청을 처리하는 핸들러 함수를 생성합니다.
func (cs *CoapServer) createHandler(endpoint EndpointConfig, method string) mux.HandlerFunc {
	return func(w mux.ResponseWriter, r *mux.Message) {
		if r.Code().String() != method {
			w.SetResponse(codes.MethodNotAllowed, message.TextPlain, bytes.NewReader([]byte("Method Not Allowed")))
			return
		}

		path, err := r.Options().Path()
		if err != nil {
			log.Printf("Cannot get path: %v", err)
			w.SetResponse(codes.InternalServerError, message.TextPlain, bytes.NewReader([]byte("Cannot get path")))
			return
		}

		var bodyBytes []byte
		if r.Body() != nil {
			bodyBytes, err = io.ReadAll(r.Body())
			if err != nil {
				log.Printf("Cannot read body: %v", err)
				w.SetResponse(codes.InternalServerError, message.TextPlain, bytes.NewReader([]byte("Cannot read body")))
				return
			}
		}

		if r.Code() == codes.POST && len(bodyBytes) > 0 {
			var jsonData map[string]interface{}
			if err := json.Unmarshal(bodyBytes, &jsonData); err == nil {
				logPayload := make(map[string]interface{})
				logPayload["url"] = endpoint.Path
				logPayload["data"] = jsonData
				logPayload["path_params"] = path

				if logBytes, logErr := json.Marshal(logPayload); logErr == nil {
					log.Printf("CoAP Request Log: %s", string(logBytes))
				}
				go func() { saveToDB(cs.db, endpoint.Path, jsonData, nil, cs.publisher) }()

			} else {
				log.Printf("CoAP %s %s - %d | Request body: %s", r.Code(), endpoint.Path, endpoint.Response.Status, string(bodyBytes))
			}
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
func (cs *CoapServer) Stop() {
	log.Println("Stopping CoAP server is not supported in the current implementation.")
}

// Reload는 새로운 설정으로 CoAP 서버를 재시작합니다.
func (cs *CoapServer) Reload(newConfig *Config) {
	log.Println("Reloading CoAP server is not supported in the current implementation.")
}

