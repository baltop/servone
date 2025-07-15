package snmpclient

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
)

var GlobalSNMPClient *SNMPClient

// SetGlobalSNMPClient sets the global SNMP client instance
func SetGlobalSNMPClient(client *SNMPClient) {
	GlobalSNMPClient = client
}

// HandleSNMPRequest processes SNMP requests from HTTP endpoints
func HandleSNMPRequest(w http.ResponseWriter, r *http.Request, operation string) {
	if GlobalSNMPClient == nil {
		http.Error(w, "SNMP client not initialized", http.StatusInternalServerError)
		return
	}

	// Read request body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	var reqData map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &reqData); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Process based on operation type
	switch operation {
	case "get":
		handleGet(w, reqData)
	default:
		http.Error(w, "Unknown or unsupported SNMP operation", http.StatusBadRequest)
	}
}

func handleGet(w http.ResponseWriter, data map[string]interface{}) {
	target, ok := data["target"].(string)
	if !ok || target == "" {
		http.Error(w, "Target is required", http.StatusBadRequest)
		return
	}

	oidsRaw, ok := data["oids"].([]interface{})
	if !ok || len(oidsRaw) == 0 {
		http.Error(w, "OIDs are required", http.StatusBadRequest)
		return
	}

	// Convert OIDs to string slice
	oids := make([]string, len(oidsRaw))
	for i, oid := range oidsRaw {
		oidStr, ok := oid.(string)
		if !ok {
			http.Error(w, "Invalid OID format", http.StatusBadRequest)
			return
		}
		oids[i] = oidStr
	}

	// Execute SNMP GET
	err := GlobalSNMPClient.GetV3(target, oids)
	if err != nil {
		log.Printf("SNMP GET error: %v", err)
		response := map[string]interface{}{
			"status": "error",
			"error":  err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Success response
	response := map[string]interface{}{
		"status":  "success",
		"message": "SNMP GET operation completed",
		"target":  target,
		"oids":    oids,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// DEPRECATED: Walk is now handled by a periodic scheduler.
// func handleWalk(w http.ResponseWriter, data map[string]interface{}) {
// 	target, ok := data["target"].(string)
// 	if !ok || target == "" {
// 		http.Error(w, "Target is required", http.StatusBadRequest)
// 		return
// 	}

// 	rootOID, ok := data["root_oid"].(string)
// 	if !ok || rootOID == "" {
// 		http.Error(w, "root_oid is required", http.StatusBadRequest)
// 		return
// 	}

// 	// Execute SNMP WALK
// 	err := GlobalSNMPClient.WalkV3(target, rootOID)
// 	if err != nil {
// 		log.Printf("SNMP WALK error: %v", err)
// 		response := map[string]interface{}{
// 			"status": "error",
// 			"error":  err.Error(),
// 		}
// 		w.Header().Set("Content-Type", "application/json")
// 		w.WriteHeader(http.StatusInternalServerError)
// 		json.NewEncoder(w).Encode(response)
// 		return
// 	}

// 	// Success response
// 	response := map[string]interface{}{
// 		"status":   "success",
// 		"message":  "SNMP WALK operation completed",
// 		"target":   target,
// 		"root_oid": rootOID,
// 	}
// 	w.Header().Set("Content-Type", "application/json")
// 	json.NewEncoder(w).Encode(response)
// }