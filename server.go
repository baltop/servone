package main

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/gorilla/mux"
)

type DynamicServer struct {
	config   *Config
	router   *mux.Router
	server   *http.Server
	pathVars map[string]*regexp.Regexp
}

func NewDynamicServer(config *Config) *DynamicServer {
	ds := &DynamicServer{
		config:   config,
		router:   mux.NewRouter(),
		pathVars: make(map[string]*regexp.Regexp),
	}
	
	ds.setupRoutes()
	
	ds.server = &http.Server{
		Addr:    ds.config.Server.Host + ":" + ds.config.Server.Port,
		Handler: ds.router,
	}
	
	return ds
}

func (ds *DynamicServer) setupRoutes() {
	for _, endpoint := range ds.config.Endpoints {
		ds.addRoute(endpoint)
	}
}

func (ds *DynamicServer) addRoute(endpoint EndpointConfig) {
	path := endpoint.Path
	method := strings.ToUpper(endpoint.Method)
	
	handler := ds.createHandler(endpoint)
	
	if strings.Contains(path, "{") {
		ds.router.HandleFunc(path, handler).Methods(method)
	} else {
		ds.router.HandleFunc(path, handler).Methods(method)
	}
	
	log.Printf("Added route: %s %s", method, path)
}

func (ds *DynamicServer) createHandler(endpoint EndpointConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		
		var requestBody string
		if r.Body != nil {
			bodyBytes, err := io.ReadAll(r.Body)
			if err == nil && len(bodyBytes) > 0 {
				requestBody = string(bodyBytes)
				r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}
		}
		
		for key, value := range endpoint.Response.Headers {
			w.Header().Set(key, value)
		}
		
		w.WriteHeader(endpoint.Response.Status)
		
		body := ds.processTemplate(endpoint.Response.Body, vars)
		w.Write([]byte(body))
		
		if requestBody != "" {
			log.Printf("%s %s - %d | Request body: %s", r.Method, r.URL.Path, endpoint.Response.Status, requestBody)
		} else {
			log.Printf("%s %s - %d", r.Method, r.URL.Path, endpoint.Response.Status)
		}
	}
}

func (ds *DynamicServer) processTemplate(body string, vars map[string]string) string {
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

func (ds *DynamicServer) Start() error {
	log.Printf("Starting server on %s", ds.server.Addr)
	return ds.server.ListenAndServe()
}

func (ds *DynamicServer) Reload(newConfig *Config) {
	log.Println("Reloading server configuration...")
	
	ds.config = newConfig
	ds.router = mux.NewRouter()
	ds.pathVars = make(map[string]*regexp.Regexp)
	
	ds.setupRoutes()
	ds.server.Handler = ds.router
	
	log.Println("Server configuration reloaded successfully")
}