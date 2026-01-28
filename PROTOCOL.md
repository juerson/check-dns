# Protocol v2 Specification

## Overview

This document describes the v2 protocol used for communication between the Go proxy client and JavaScript WebSocket server.

## Protocol Constants

| Constant | Value | Description |
|----------|-------|-------------|
| MAGIC | 0xA5 | Frame magic number |
| VERSION | 2 | Protocol version |
| CMD_AUTH | 0x01 | Authentication command |
| CMD_OPEN | 0x02 | Open connection command |
| CMD_DATA | 0x03 | Data transfer command |
| CMD_CLOSE | 0x04 | Close connection command |
| FLAG_NONE | 0x00 | No flags set |

## Frame Structure

Every frame follows this structure:

```
+--------+--------+--------+--------+--------+--------+--------+--------+
| Byte 0 | Byte 1 | Byte 2 | Byte 3 |    Bytes 4-7 (4 bytes)          |
+--------+--------+--------+--------+--------+--------+--------+--------+
| MAGIC  |VERSION |  CMD   | FLAGS  |      PAYLOAD_LEN (Big-Endian)    |
| 0xA5   |   2    |  0x01- | 0x00   |         (32-bit unsigned)         |
|        |        |  0x04  |        |                                   |
+--------+--------+--------+--------+-----------------------------------+
|                    Payload (variable length)                          |
|                       ... N bytes ...                                 |
+-----------------------------------------------------------------------+
```

### Field Descriptions

- **MAGIC (1 byte)**: Always 0xA5, used to identify valid frames
- **VERSION (1 byte)**: Protocol version, currently 2
- **CMD (1 byte)**: Command type (AUTH, OPEN, DATA, CLOSE)
- **FLAGS (1 byte)**: Reserved for future extensions (currently 0x00)
- **PAYLOAD_LEN (4 bytes)**: Length of payload in bytes, big-endian unsigned 32-bit integer
- **PAYLOAD (variable)**: Command-specific data

## Connection Flow

```
Go Client                                JavaScript Server
    |                                            |
    |-------- WebSocket Connection ------------>|
    |                                            |
    |-------- AUTH Frame (PSK token) ---------->|
    |                                            | Verify token
    |<------- AUTH Success / CLOSE -------------|
    |                                            |
    |-------- OPEN Frame (target) ------------->|
    |                                            | Connect to target
    |<------- OPEN Success / CLOSE -------------|
    |                                            |
    |<======= Bidirectional DATA Frames ========>|
    |                                            |
    |-------- CLOSE Frame --------------------->|
    |<------- CLOSE Frame ----------------------|
    |                                            |
    X                                            X
```

## Command Details

### CMD_AUTH (0x01)

**Purpose**: Authenticate the client using a pre-shared key (PSK).

**Client → Server**:
- Payload: UTF-8 encoded authentication token

**Server → Client**:
- Success: Any non-CLOSE frame (typically AUTH frame with "OK")
- Failure: CLOSE frame with error message

**Example**:
```
Client sends:
[0xA5, 0x02, 0x01, 0x00, 0x00, 0x00, 0x00, 0x10, 's','e','c','r','e','t','-','t','o','k','e','n','-','1','2','3']
                                     ^^^^^^^^ Length: 16 bytes
```

### CMD_OPEN (0x02)

**Purpose**: Request to open a connection to a target host.

**Client → Server**:
- Format 1 (no initial payload): `"host:port"`
- Format 2 (with initial payload): `"host:port|<binary_data>"`

**Server → Client**:
- Success: OPEN frame with status message
- Failure: CLOSE frame with error message

**Example 1 - Simple OPEN**:
```
Payload: "example.com:443"
```

**Example 2 - OPEN with first payload (HTTP request)**:
```
Payload: "example.com:80|GET / HTTP/1.1\r\nHost: example.com\r\n\r\n"
         ^^^^^^^^^^^^^^^^ ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
         target           first payload to send to target
```

### CMD_DATA (0x03)

**Purpose**: Transfer data between client and target.

**Bidirectional**:
- Client → Server: Data from local application to forward to target
- Server → Client: Data received from target to forward to local application

**Example**:
```
Client sends HTTP request:
Payload: "GET /api/data HTTP/1.1\r\nHost: api.example.com\r\n\r\n"

Server sends HTTP response:
Payload: "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\n\r\n{...}"
```

### CMD_CLOSE (0x04)

**Purpose**: Signal connection closure.

**Either Direction**:
- Payload: Optional error message or reason (UTF-8 string)

**Common Close Reasons**:
- `AUTH_FAILED`: Authentication failed
- `NOT_AUTHENTICATED`: Received command before authentication
- `CONNECT_FAILED`: Failed to connect to target
- `UNKNOWN_CMD`: Unrecognized command
- Empty payload: Normal closure

## Error Handling

### Authentication Errors

1. Client sends AUTH with invalid token
2. Server responds with CLOSE frame containing "AUTH_FAILED"
3. Client logs error and terminates connection

### Connection Errors

1. Client sends OPEN for unreachable target
2. Server responds with CLOSE frame containing "CONNECT_FAILED"
3. Client returns proxy error to local application

### Protocol Errors

1. Invalid MAGIC or VERSION detected
2. Recipient logs error and closes connection
3. No frame is sent (connection is dropped)

## Proxy Modes

### SOCKS5 Mode

1. Client accepts SOCKS5 connection
2. Client performs SOCKS5 handshake locally
3. Client extracts target from SOCKS5 request
4. Client sends AUTH + OPEN frames
5. Client proxies data via DATA frames

### HTTP CONNECT Mode

1. Client accepts HTTP CONNECT request
2. Client extracts target from CONNECT line
3. Client responds "200 Connection Established"
4. Client sends AUTH + OPEN frames
5. Client proxies data via DATA frames

### HTTP Proxy Mode

1. Client accepts HTTP request
2. Client extracts target from Host header
3. Client captures entire HTTP request
4. Client sends AUTH + OPEN with first payload (HTTP request)
5. Client proxies subsequent data via DATA frames

## Version Upgrade Path

### Upgrading to v3

To support future protocol versions:

1. **Update VERSION constant**:
   ```go
   const VERSION byte = 3
   ```

2. **Add version negotiation in parseFrame()**:
   ```go
   func parseFrame(buffer []byte) (cmd byte, flags byte, payload []byte, ok bool) {
       version := buffer[1]
       switch version {
       case 2:
           // v2 handling (backward compatible)
       case 3:
           // v3 new features
       default:
           return 0, 0, nil, false
       }
       // ...
   }
   ```

3. **Utilize FLAGS field**:
   ```go
   const (
       FLAG_COMPRESSED = 0x01
       FLAG_ENCRYPTED  = 0x02
       FLAG_FRAGMENTED = 0x04
   )
   ```

4. **Update frame handlers**:
   ```go
   if flags & FLAG_COMPRESSED != 0 {
       payload = decompress(payload)
   }
   ```

## Security Considerations

### Authentication

- Uses PSK (Pre-Shared Key) authentication via AUTH frame
- Token transmitted in cleartext over WebSocket
- **Recommendation**: Always use WSS (WebSocket Secure) in production

### TLS/ECH

- Client supports TLS 1.2+ with optional ECH (Encrypted Client Hello)
- ECH helps prevent SNI-based censorship
- Standard TLS configuration used by default

### Token Management

- Store AUTH_TOKEN in environment variables, not in code
- Rotate tokens periodically
- Use strong, randomly generated tokens (minimum 32 characters)

## Performance Considerations

### Buffer Sizes

- Default WebSocket read buffer: 32KB
- Default WebSocket write buffer: 32KB
- Adjust based on expected traffic patterns

### Frame Overhead

- Each frame adds 8 bytes overhead
- For large transfers, use maximum payload size
- For small messages, overhead is minimal

### Connection Pooling

- Not implemented in current version
- Future enhancement for connection reuse
- Would use FLAGS to multiplex multiple streams

## Testing

### Unit Tests

Run comprehensive frame encoding/decoding tests:
```bash
go test -v
```

### Integration Tests

Test with real JavaScript server:
```bash
# Start JavaScript server
# ...

# Start Go client
export SERVER_URL="wss://your-server.com/ws"
export AUTH_TOKEN="test-token-123"
export PROXY_MODE="socks5"
./proxy-client

# Test with curl through SOCKS5
curl --socks5 127.0.0.1:1080 http://example.com
```

## Examples

### Example 1: Simple SOCKS5 Connection

```
1. SOCKS5 client connects to 127.0.0.1:1080
2. Go client performs SOCKS5 handshake
3. Target extracted: example.com:80

WebSocket frames:
→ AUTH: [0xA5, 0x02, 0x01, 0x00, ...token...]
← AUTH: [0xA5, 0x02, 0x01, 0x00, 0x00, 0x00, 0x00, 0x02, 'O','K']
→ OPEN: [0xA5, 0x02, 0x02, 0x00, ...,'example.com:80']
← OPEN: [0xA5, 0x02, 0x02, 0x00, 0x00, 0x00, 0x00, 0x02, 'O','K']
→ DATA: [0xA5, 0x02, 0x03, 0x00, ..., 'GET / HTTP/1.1...']
← DATA: [0xA5, 0x02, 0x03, 0x00, ..., 'HTTP/1.1 200 OK...']
→ CLOSE: [0xA5, 0x02, 0x04, 0x00, 0x00, 0x00, 0x00, 0x00]
```

### Example 2: HTTP Proxy with First Payload

```
1. HTTP client sends: GET http://example.com/api HTTP/1.1
2. Go client extracts target: example.com:80
3. Go client captures first payload: full HTTP request

WebSocket frames:
→ AUTH: [token]
← AUTH: [OK]
→ OPEN: [0xA5, 0x02, 0x02, 0x00, ..., 'example.com:80|GET /api HTTP/1.1...']
                                        ^^^^^^^^^^^^^^^^ ^^^^^^^^^^^^^^^^^^^^^
                                        target           first payload
← OPEN: [OK]
← DATA: [HTTP response]
→ CLOSE: []
```

## Alignment with JavaScript Server

This Go client implementation fully aligns with JavaScript server specifications:

✓ Identical frame structure
✓ Same command definitions (0x01-0x04)
✓ Same OPEN format ("host:port|firstPayload")
✓ Same error handling flow (CLOSE on error)
✓ Compatible with Cloudflare Workers WebSocket API
✓ UTF-8 encoding for text payloads
✓ Big-endian byte order for integers

## References

- [RFC 6455 - The WebSocket Protocol](https://tools.ietf.org/html/rfc6455)
- [SOCKS Protocol Version 5](https://tools.ietf.org/html/rfc1928)
- [HTTP/1.1 CONNECT Method](https://tools.ietf.org/html/rfc7231#section-4.3.6)
- [Cloudflare Workers WebSocket](https://developers.cloudflare.com/workers/runtime-apis/websockets/)
