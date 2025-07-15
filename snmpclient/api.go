package snmpclient

import (
	"encoding/json"
	"net/http"
)

// SNMPHandler provides HTTP endpoints for SNMP operations
type SNMPHandler struct {
	client *SNMPClient
}

// NewSNMPHandler creates a new SNMP HTTP handler
func NewSNMPHandler(client *SNMPClient) *SNMPHandler {
	return &SNMPHandler{
		client: client,
	}
}

// HandleGet handles SNMP GET requests
func (h *SNMPHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Target string   `json:"target"`
		OIDs   []string `json:"oids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Target == "" || len(req.OIDs) == 0 {
		http.Error(w, "Target and OIDs are required", http.StatusBadRequest)
		return
	}

	err := h.client.GetV3(req.Target, req.OIDs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// HandleWalk handles SNMP WALK requests
// DEPRECATED: Walk is now handled by a periodic scheduler.
// func (h *SNMPHandler) HandleWalk(w http.ResponseWriter, r *http.Request) {
// 	if r.Method != http.MethodPost {
// 		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
// 		return
// 	}

// 	var req struct {
// 		Target  string `json:"target"`
// 		RootOID string `json:"root_oid"`
// 	}

// 	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
// 		http.Error(w, "Invalid request body", http.StatusBadRequest)
// 		return
// 	}

// 	if req.Target == "" || req.RootOID == "" {
// 		http.Error(w, "Target and root_oid are required", http.StatusBadRequest)
// 		return
// 	}

// 	err := h.client.WalkV3(req.Target, req.RootOID)
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}

// 	w.Header().Set("Content-Type", "application/json")
// 	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
// }

// RegisterRoutes registers SNMP HTTP routes
func (h *SNMPHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/snmp/get", h.HandleGet)
}