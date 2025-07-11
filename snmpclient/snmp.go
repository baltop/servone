package snmpclient

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"servone/config"
	"servone/kafka"
	"time"

	"github.com/gosnmp/gosnmp"
)

type SNMPClient struct {
	config         *config.SNMPConfig
	db             *sql.DB
	kafkaPublisher KafkaPublisherInterface
	trapListener   *gosnmp.TrapListener
}

type KafkaPublisherInterface interface {
	Publish(topic string, data map[string]interface{}) error
}

func NewSNMPClient(cfg *config.SNMPConfig, db *sql.DB, kafkaPublisher KafkaPublisherInterface) (*SNMPClient, error) {
	client := &SNMPClient{
		config:         cfg,
		db:             db,
		kafkaPublisher: kafkaPublisher,
	}

	// Start TRAP listener if enabled
	if cfg.TrapEnabled {
		if err := client.startTrapListener(); err != nil {
			return nil, fmt.Errorf("failed to start trap listener: %w", err)
		}
	}

	return client, nil
}

// GetV3 performs SNMP v3 GET operation
func (c *SNMPClient) GetV3(target string, oids []string) error {
	g := &gosnmp.GoSNMP{
		Target:             target,
		Port:               uint16(c.config.Port),
		Version:            gosnmp.Version3,
		SecurityModel:      gosnmp.UserSecurityModel,
		MsgFlags:           gosnmp.AuthPriv,
		SecurityParameters: c.getSecurityParams(),
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

	// Process and store results
	c.processResults("get", target, result.Variables)

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
		SecurityParameters: c.getSecurityParams(),
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

	// Process and store results
	c.processResults("walk", target, pdus)

	return nil
}

// startTrapListener starts the SNMP v3 TRAP listener
func (c *SNMPClient) startTrapListener() error {
	c.trapListener = gosnmp.NewTrapListener()
	c.trapListener.OnNewTrap = c.handleTrap

	// Configure listener parameters - Params expects a pointer to GoSNMP
	c.trapListener.Params = &gosnmp.GoSNMP{
		Version:            gosnmp.Version3,
		SecurityModel:      gosnmp.UserSecurityModel,
		MsgFlags:           gosnmp.AuthPriv,
		SecurityParameters: c.getSecurityParams(),
	}

	// Start listening in goroutine
	go func() {
		addr := fmt.Sprintf("%s:%d", c.config.TrapHost, c.config.TrapPort)
		log.Printf("Starting SNMP trap listener on %s", addr)

		err := c.trapListener.Listen(addr)
		if err != nil {
			log.Printf("SNMP trap listener error: %v", err)
		}
	}()

	return nil
}

// handleTrap processes incoming SNMP traps
func (c *SNMPClient) handleTrap(packet *gosnmp.SnmpPacket, addr *net.UDPAddr) {
	log.Printf("Received SNMP trap from %s", addr.String())

	// Process trap data
	c.processResults("trap", addr.String(), packet.Variables)
}

// getSecurityParams creates USM security parameters from config
func (c *SNMPClient) getSecurityParams() *gosnmp.UsmSecurityParameters {
	authProtocol := gosnmp.NoAuth
	privProtocol := gosnmp.NoPriv

	switch c.config.AuthProtocol {
	case "MD5":
		authProtocol = gosnmp.MD5
	case "SHA":
		authProtocol = gosnmp.SHA
	case "SHA224":
		authProtocol = gosnmp.SHA224
	case "SHA256":
		authProtocol = gosnmp.SHA256
	case "SHA384":
		authProtocol = gosnmp.SHA384
	case "SHA512":
		authProtocol = gosnmp.SHA512
	}

	switch c.config.PrivProtocol {
	case "DES":
		privProtocol = gosnmp.DES
	case "AES":
		privProtocol = gosnmp.AES
	case "AES192":
		privProtocol = gosnmp.AES192
	case "AES256":
		privProtocol = gosnmp.AES256
	case "AES192C":
		privProtocol = gosnmp.AES192C
	case "AES256C":
		privProtocol = gosnmp.AES256C
	}

	return &gosnmp.UsmSecurityParameters{
		UserName:                 c.config.Username,
		AuthenticationProtocol:   authProtocol,
		AuthenticationPassphrase: c.config.AuthPassphrase,
		PrivacyProtocol:          privProtocol,
		PrivacyPassphrase:        c.config.PrivPassphrase,
	}
}

// processResults stores SNMP results in database and publishes to Kafka
func (c *SNMPClient) processResults(operation string, source string, pdus []gosnmp.SnmpPDU) {
	receivedTime := time.Now().UnixNano()

	// Prepare data for storage
	var results []map[string]interface{}
	for _, pdu := range pdus {
		result := map[string]interface{}{
			"oid":   pdu.Name,
			"type":  pdu.Type.String(),
			"value": c.getValueString(pdu),
		}
		results = append(results, result)
	}

	data := map[string]interface{}{
		"operation": operation,
		"source":    source,
		"results":   results,
		"timestamp": receivedTime,
	}

	// Save to database
	// if err := db.SaveSNMPData(source, data, receivedTime); err != nil {

	// if err := db.SaveSNMPData(source, data, receivedTime); err != nil {
	// 	log.Printf("Failed to save SNMP data to DB: %v", err)
	// }

	// Publish to Kafka
	kafkaTopic := fmt.Sprintf("snmp.%s.%s", operation, kafka.SanitizeTopic(source))
	if err := c.kafkaPublisher.Publish(kafkaTopic, data); err != nil {
		log.Printf("Failed to publish SNMP data to Kafka: %v", err)
	}
}

// getValueString converts SNMP PDU value to string
func (c *SNMPClient) getValueString(pdu gosnmp.SnmpPDU) string {
	switch pdu.Type {
	case gosnmp.OctetString:
		return string(pdu.Value.([]byte))
	case gosnmp.ObjectIdentifier:
		return pdu.Value.(string)
	default:
		return fmt.Sprintf("%v", pdu.Value)
	}
}

// Stop gracefully stops the SNMP client
func (c *SNMPClient) Stop() {
	if c.trapListener != nil {
		c.trapListener.Close()
		log.Println("SNMP trap listener stopped")
	}
}
