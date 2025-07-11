# SNMP Client Package

This package provides SNMP v3 functionality for the servone application, supporting GET, WALK, and TRAP operations.

## Features

- **SNMP v3 Support**: Full support for SNMP version 3 with authentication and privacy
- **GET Operations**: Retrieve specific OID values from SNMP agents
- **WALK Operations**: Walk through SNMP MIB trees
- **TRAP Receiver**: Listen for and process SNMP v3 traps
- **Database Storage**: All SNMP data is stored in PostgreSQL
- **Kafka Publishing**: SNMP data is published to Kafka topics

## Configuration

Add the following to your `config.yaml`:

```yaml
snmp:
  port: 161
  timeout: 5
  retries: 3
  username: "snmpuser"
  auth_protocol: "SHA256"  # Options: MD5, SHA, SHA224, SHA256, SHA384, SHA512
  auth_passphrase: "authpassword"
  priv_protocol: "AES256"  # Options: DES, AES, AES192, AES256, AES192C, AES256C
  priv_passphrase: "privpassword"
  trap_enabled: true
  trap_host: "0.0.0.0"
  trap_port: 162
```

## API Endpoints

### SNMP GET
```bash
POST /api/snmp/get
Content-Type: application/json

{
  "target": "192.168.1.1",
  "oids": [".1.3.6.1.2.1.1.1.0", ".1.3.6.1.2.1.1.3.0"]
}
```

### SNMP WALK
```bash
POST /api/snmp/walk
Content-Type: application/json

{
  "target": "192.168.1.1",
  "root_oid": ".1.3.6.1.2.1.1"
}
```

## Database Schema

SNMP messages are stored in the `snmp_messages` table:

```sql
CREATE TABLE snmp_messages (
    id SERIAL PRIMARY KEY,
    operation TEXT NOT NULL,
    source TEXT NOT NULL,
    data JSONB,
    created_at BIGINT
);
```

## Kafka Topics

- GET operations: `snmp.get.<sanitized_target>`
- WALK operations: `snmp.walk.<sanitized_target>`
- TRAP operations: `snmp.trap.<sanitized_source>`

## Testing

Run tests with:
```bash
go test ./snmpclient
```

Note: Some tests require a running PostgreSQL instance.