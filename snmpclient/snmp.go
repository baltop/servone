package snmpclient

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"servone/config"
	"servone/kafka"
	"sync"
	"time"

	"github.com/gosnmp/gosnmp"
)

// SNMPClient handles periodic SNMP WALK operations.
type SNMPClient struct {
	config         *config.SNMPConfig
	db             *sql.DB
	kafkaPublisher KafkaPublisherInterface
	stop           chan struct{}
	wg             sync.WaitGroup
}

// TrapServer handles incoming SNMP traps.
type TrapServer struct {
	config         *config.SNMPTrapConfig
	db             *sql.DB
	kafkaPublisher KafkaPublisherInterface
	listener       *gosnmp.TrapListener
}

type KafkaPublisherInterface interface {
	Publish(topic string, data map[string]interface{}) error
}

// NewSNMPClient creates a new client for GET and periodic WALK operations.
func NewSNMPClient(cfg *config.SNMPConfig, db *sql.DB, kafkaPublisher KafkaPublisherInterface) *SNMPClient {
	return &SNMPClient{
		config:         cfg,
		db:             db,
		kafkaPublisher: kafkaPublisher,
		stop:           make(chan struct{}),
	}
}

// StartWalkScheduler starts the periodic SNMP WALK operation in a goroutine.
func (c *SNMPClient) StartWalkScheduler() {
	if c.config.Term <= 0 {
		log.Println("SNMP walk scheduler is disabled (term is zero or negative).")
		return
	}

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		log.Printf("Starting SNMP walk scheduler with term %d seconds", c.config.Term)
		ticker := time.NewTicker(time.Duration(c.config.Term) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				log.Println("Executing scheduled SNMP walk.")
				for _, target := range c.config.Targets {
					if err := c.WalkV3(target, c.config.RootOid); err != nil {
						log.Printf("Scheduled SNMP walk failed for target %s: %v", target, err)
					}
				}
			case <-c.stop:
				log.Println("Stopping SNMP walk scheduler.")
				return
			}
		}
	}()
}

// Stop gracefully stops the SNMP client and its scheduler.
func (c *SNMPClient) Stop() {
	close(c.stop)
	c.wg.Wait()
	log.Println("SNMP client stopped.")
}

// GetV3 performs SNMP v3 GET operation
func (c *SNMPClient) GetV3(target string, oids []string) error {
	g := &gosnmp.GoSNMP{
		Target:             target,
		Port:               uint16(c.config.Port),
		Version:            gosnmp.Version3,
		SecurityModel:      gosnmp.UserSecurityModel,
		MsgFlags:           gosnmp.AuthPriv,
		SecurityParameters: getSecurityParams(c.config.Username, c.config.AuthProtocol, c.config.AuthPassphrase, c.config.PrivProtocol, c.config.PrivPassphrase),
		Timeout:            time.Duration(c.config.Timeout) * time.Second,
		Retries:            c.config.Retries,
	}

	err := g.Connect()
	if err != nil {
		return fmt.Errorf("connect failed: %w", err)
	}
	defer g.Conn.Close()

	result, err := g.Get(oids)
	if err != nil {
		return fmt.Errorf("get failed: %w", err)
	}

	processResults("get", target, result.Variables, c.kafkaPublisher, c.db)
	return nil
}

// WalkV3 performs SNMP v3 WALK operation
func (c *SNMPClient) WalkV3(target string, rootOid string) error {
	g := &gosnmp.GoSNMP{
		Target:             target,
		Port:               uint16(c.config.Port),
		Version:            gosnmp.Version3,
		SecurityModel:      gosnmp.UserSecurityModel,
		MsgFlags:           gosnmp.AuthPriv,
		SecurityParameters: getSecurityParams(c.config.Username, c.config.AuthProtocol, c.config.AuthPassphrase, c.config.PrivProtocol, c.config.PrivPassphrase),
		Timeout:            time.Duration(c.config.Timeout) * time.Second,
		Retries:            c.config.Retries,
	}

	err := g.Connect()
	if err != nil {
		return fmt.Errorf("connect failed: %w", err)
	}
	defer g.Conn.Close()

	var pdus []gosnmp.SnmpPDU
	err = g.Walk(rootOid, func(pdu gosnmp.SnmpPDU) error {
		pdus = append(pdus, pdu)
		return nil
	})

	if err != nil {
		return fmt.Errorf("walk failed: %w", err)
	}

	processResults("walk", target, pdus, c.kafkaPublisher, c.db)
	return nil
}

// NewTrapServer creates a new server for receiving SNMP traps.
func NewTrapServer(cfg *config.SNMPTrapConfig, kafkaPublisher KafkaPublisherInterface, db *sql.DB) *TrapServer {
	return &TrapServer{
		config:         cfg,
		kafkaPublisher: kafkaPublisher,
		db:             db,
	}
}

// Start starts the SNMP trap listener.
func (ts *TrapServer) Start() error {
	if !ts.config.Enabled {
		log.Println("SNMP trap server is disabled.")
		return nil
	}

	ts.listener = gosnmp.NewTrapListener()
	ts.listener.OnNewTrap = ts.handleTrap
	ts.listener.Params = &gosnmp.GoSNMP{
		Version:            gosnmp.Version3,
		SecurityModel:      gosnmp.UserSecurityModel,
		MsgFlags:           gosnmp.AuthPriv,
		SecurityParameters: getSecurityParams(ts.config.Username, ts.config.AuthProtocol, ts.config.AuthPassphrase, ts.config.PrivProtocol, ts.config.PrivPassphrase),
	}

	addr := fmt.Sprintf("%s:%d", ts.config.Host, ts.config.Port)
	log.Printf("Starting SNMP trap listener on %s", addr)

	// Start listening in a goroutine to not block the main thread.
	go func() {
		if err := ts.listener.Listen(addr); err != nil {
			log.Printf("SNMP trap listener error: %v", err)
		}
	}()

	return nil
}

// Stop gracefully stops the SNMP trap listener.
func (ts *TrapServer) Stop() {
	if ts.listener != nil {
		ts.listener.Close()
		log.Println("SNMP trap listener stopped.")
	}
}

// handleTrap processes incoming SNMP traps for the TrapServer.
func (ts *TrapServer) handleTrap(packet *gosnmp.SnmpPacket, addr *net.UDPAddr) {
	log.Printf("Received SNMP trap from %s", addr.String())
	processResults("trap", addr.String(), packet.Variables, ts.kafkaPublisher, ts.db)
}

// getSecurityParams creates USM security parameters from config.
func getSecurityParams(username, authProtocol, authPassphrase, privProtocol, privPassphrase string) *gosnmp.UsmSecurityParameters {
	authProto := gosnmp.NoAuth
	switch authProtocol {
	case "MD5":
		authProto = gosnmp.MD5
	case "SHA":
		authProto = gosnmp.SHA
	case "SHA224":
		authProto = gosnmp.SHA224
	case "SHA256":
		authProto = gosnmp.SHA256
	case "SHA384":
		authProto = gosnmp.SHA384
	case "SHA512":
		authProto = gosnmp.SHA512
	}

	privProto := gosnmp.NoPriv
	switch privProtocol {
	case "DES":
		privProto = gosnmp.DES
	case "AES":
		privProto = gosnmp.AES
	case "AES192":
		privProto = gosnmp.AES192
	case "AES256":
		privProto = gosnmp.AES256
	case "AES192C":
		privProto = gosnmp.AES192C
	case "AES256C":
		privProto = gosnmp.AES256C
	}

	return &gosnmp.UsmSecurityParameters{
		UserName:                 username,
		AuthenticationProtocol:   authProto,
		AuthenticationPassphrase: authPassphrase,
		PrivacyProtocol:          privProto,
		PrivacyPassphrase:        privPassphrase,
	}
}

// processResults processes SNMP data and publishes it to Kafka.
func processResults(operation string, source string, pdus []gosnmp.SnmpPDU, publisher KafkaPublisherInterface, db *sql.DB) {
	receivedTime := time.Now().UnixNano()

	var results []map[string]interface{}
	for _, pdu := range pdus {
		result := map[string]interface{}{
			"oid":   pdu.Name,
			"type":  pdu.Type.String(),
			"value": getValueString(pdu),
		}
		results = append(results, result)
	}

	data := map[string]interface{}{
		"operation": operation,
		"source":    source,
		"results":   results,
		"timestamp": receivedTime,
	}

	// Publish to Kafka
	kafkaTopic := fmt.Sprintf("snmp.%s.%s", operation, kafka.SanitizeTopic(source))
	if err := publisher.Publish(kafkaTopic, data); err != nil {
		log.Printf("Failed to publish SNMP data to Kafka: %v", err)
	}

	// Save to database
	if db != nil {
		if err := saveSNMPDataToDB(db, source, data, receivedTime); err != nil {
			log.Printf("Failed to save SNMP data to DB: %v", err)
		}
	}
}

// saveSNMPDataToDB saves SNMP data to the database.
func saveSNMPDataToDB(db *sql.DB, host string, data map[string]interface{}, receivedTime int64) error {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal SNMP data: %w", err)
	}

	insertSQL := `
	INSERT INTO snmp_data (host, data, created_at)
	VALUES ($1, $2, $3);`

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = db.ExecContext(ctx, insertSQL, host, dataJSON, receivedTime)
	if err != nil {
		return fmt.Errorf("failed to insert SNMP data: %w", err)
	}
	return nil
}

// getValueString converts SNMP PDU value to a string representation.
func getValueString(pdu gosnmp.SnmpPDU) string {
	switch pdu.Type {
	case gosnmp.OctetString:
		return string(pdu.Value.([]byte))
	case gosnmp.ObjectIdentifier:
		return pdu.Value.(string)
	default:
		return fmt.Sprintf("%v", pdu.Value)
	}
}
