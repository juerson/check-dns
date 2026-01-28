package main

import (
    "crypto/tls"
    "encoding/binary"
    "fmt"
    "io"
    "log"
    "net"
    "net/http"
    "os"
    "strings"
    "time"

    "github.com/gorilla/websocket"
)

// Protocol v2 constants
const (
    MAGIC   byte = 0xA5
    VERSION byte = 2

    // Frame commands
    CMD_AUTH  byte = 0x01
    CMD_OPEN  byte = 0x02
    CMD_DATA  byte = 0x03
    CMD_CLOSE byte = 0x04

    // Flags (reserved for future use)
    FLAG_NONE byte = 0x00
)

// Configuration
var (
    serverURL  string
    authToken  string
    listenAddr string
    proxyMode  string // "socks5", "http", "connect"
)

func main() {
    // Read configuration from environment variables
    serverURL = getEnv("SERVER_URL", "wss://example.com/ws")
    authToken = getEnv("AUTH_TOKEN", "your-secret-token")
    listenAddr = getEnv("LISTEN_ADDR", "127.0.0.1:1080")
    proxyMode = getEnv("PROXY_MODE", "socks5")

    log.Printf("Starting proxy client (Protocol v2)")
    log.Printf("Server: %s", serverURL)
    log.Printf("Listen: %s", listenAddr)
    log.Printf("Mode: %s", proxyMode)

    listener, err := net.Listen("tcp", listenAddr)
    if err != nil {
        log.Fatalf("Failed to listen: %v", err)
    }
    defer listener.Close()

    log.Printf("Proxy server listening on %s", listenAddr)

    for {
        conn, err := listener.Accept()
        if err != nil {
            log.Printf("Accept error: %v", err)
            continue
        }

        go handleConnection(conn)
    }
}

func handleConnection(conn net.Conn) {
    defer conn.Close()

    switch proxyMode {
    case "socks5":
        handleSOCKS5(conn)
    case "http":
        handleHTTP(conn)
    case "connect":
        handleHTTPConnect(conn)
    default:
        log.Printf("Unknown proxy mode: %s", proxyMode)
    }
}

// SOCKS5 proxy handler
func handleSOCKS5(conn net.Conn) {
    // SOCKS5 handshake
    buf := make([]byte, 256)
    n, err := conn.Read(buf)
    if err != nil {
        log.Printf("SOCKS5 read error: %v", err)
        return
    }

    if n < 2 || buf[0] != 0x05 {
        log.Printf("Invalid SOCKS5 request")
        return
    }

    // No authentication required
    conn.Write([]byte{0x05, 0x00})

    // Read connection request
    n, err = conn.Read(buf)
    if err != nil {
        log.Printf("SOCKS5 read request error: %v", err)
        return
    }

    if n < 7 || buf[0] != 0x05 || buf[1] != 0x01 {
        log.Printf("Invalid SOCKS5 connection request")
        return
    }

    var host string
    var port uint16

    switch buf[3] {
    case 0x01: // IPv4
        host = fmt.Sprintf("%d.%d.%d.%d", buf[4], buf[5], buf[6], buf[7])
        port = binary.BigEndian.Uint16(buf[8:10])
    case 0x03: // Domain name
        addrLen := int(buf[4])
        host = string(buf[5 : 5+addrLen])
        port = binary.BigEndian.Uint16(buf[5+addrLen : 7+addrLen])
    case 0x04: // IPv6
        host = net.IP(buf[4:20]).String()
        port = binary.BigEndian.Uint16(buf[20:22])
    default:
        log.Printf("Unsupported address type: %d", buf[3])
        conn.Write([]byte{0x05, 0x08, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
        return
    }

    target := fmt.Sprintf("%s:%d", host, port)
    log.Printf("SOCKS5 connecting to %s", target)

    // Send success response
    conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0})

    // Start tunnel with empty first payload
    handleTunnel(conn, target, nil)
}

// HTTP CONNECT proxy handler
func handleHTTPConnect(conn net.Conn) {
    buf := make([]byte, 4096)
    n, err := conn.Read(buf)
    if err != nil {
        log.Printf("HTTP CONNECT read error: %v", err)
        return
    }

    request := string(buf[:n])
    lines := strings.Split(request, "\r\n")
    if len(lines) == 0 {
        log.Printf("Invalid HTTP CONNECT request")
        return
    }

    parts := strings.Split(lines[0], " ")
    if len(parts) < 2 || parts[0] != "CONNECT" {
        log.Printf("Not a CONNECT request")
        return
    }

    target := parts[1]
    log.Printf("HTTP CONNECT to %s", target)

    // Send success response
    conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

    // Start tunnel with empty first payload
    handleTunnel(conn, target, nil)
}

// HTTP proxy handler
func handleHTTP(conn net.Conn) {
    buf := make([]byte, 4096)
    n, err := conn.Read(buf)
    if err != nil {
        log.Printf("HTTP read error: %v", err)
        return
    }

    firstPayload := buf[:n]
    request := string(firstPayload)
    lines := strings.Split(request, "\r\n")
    if len(lines) == 0 {
        log.Printf("Invalid HTTP request")
        return
    }

    parts := strings.Split(lines[0], " ")
    if len(parts) < 2 {
        log.Printf("Invalid HTTP request line")
        return
    }

    // Extract host and port from request
    var target string
    for _, line := range lines {
        if strings.HasPrefix(strings.ToLower(line), "host:") {
            host := strings.TrimSpace(line[5:])
            if !strings.Contains(host, ":") {
                host += ":80"
            }
            target = host
            break
        }
    }

    if target == "" {
        log.Printf("No Host header found")
        return
    }

    log.Printf("HTTP proxy to %s", target)

    // Start tunnel with first HTTP request as payload
    handleTunnel(conn, target, firstPayload)
}

// Protocol v2: handleTunnel implements WebSocket communication with frame protocol
func handleTunnel(clientConn net.Conn, target string, firstPayload []byte) {
    // Dial WebSocket connection with ECH support
    ws, err := dialWebSocketWithECH(serverURL)
    if err != nil {
        log.Printf("Failed to connect to server: %v", err)
        return
    }
    defer ws.Close()

    log.Printf("WebSocket connected, starting v2 protocol handshake")

    // Step 1: Send AUTH frame
    authFrame := makeFrame(CMD_AUTH, FLAG_NONE, []byte(authToken))
    err = ws.WriteMessage(websocket.BinaryMessage, authFrame)
    if err != nil {
        log.Printf("Failed to send AUTH frame: %v", err)
        return
    }
    log.Printf("Sent AUTH frame")

    // Step 2: Wait for AUTH response (or CLOSE on failure)
    _, authResponse, err := ws.ReadMessage()
    if err != nil {
        log.Printf("Failed to read AUTH response: %v", err)
        return
    }

    cmd, _, _, ok := parseFrame(authResponse)
    if !ok {
        log.Printf("Invalid AUTH response frame")
        return
    }

    if cmd == CMD_CLOSE {
        log.Printf("AUTH failed - server sent CLOSE")
        return
    }

    log.Printf("AUTH successful")

    // Step 3: Send OPEN frame with "host:port|firstPayload" format
    var openPayload []byte
    if firstPayload != nil && len(firstPayload) > 0 {
        openPayload = []byte(fmt.Sprintf("%s|", target))
        openPayload = append(openPayload, firstPayload...)
    } else {
        openPayload = []byte(target)
    }

    openFrame := makeFrame(CMD_OPEN, FLAG_NONE, openPayload)
    err = ws.WriteMessage(websocket.BinaryMessage, openFrame)
    if err != nil {
        log.Printf("Failed to send OPEN frame: %v", err)
        return
    }
    log.Printf("Sent OPEN frame for %s", target)

    // Step 4: Wait for OPEN response
    _, openResponse, err := ws.ReadMessage()
    if err != nil {
        log.Printf("Failed to read OPEN response: %v", err)
        return
    }

    cmd, _, _, ok = parseFrame(openResponse)
    if !ok {
        log.Printf("Invalid OPEN response frame")
        return
    }

    if cmd == CMD_CLOSE {
        log.Printf("OPEN failed - server sent CLOSE")
        return
    }

    if cmd != CMD_OPEN {
        log.Printf("Unexpected response to OPEN: cmd=%d", cmd)
        return
    }

    log.Printf("OPEN successful, starting data transfer")

    // Step 5: Bidirectional data transfer with DATA frames
    done := make(chan bool, 2)

    // Goroutine 1: Client -> WebSocket (DATA frames)
    go func() {
        defer func() {
            done <- true
        }()

        buf := make([]byte, 32*1024)
        for {
            n, err := clientConn.Read(buf)
            if err != nil {
                if err != io.EOF {
                    log.Printf("Client read error: %v", err)
                }
                // Send CLOSE frame
                closeFrame := makeFrame(CMD_CLOSE, FLAG_NONE, nil)
                ws.WriteMessage(websocket.BinaryMessage, closeFrame)
                return
            }

            // Wrap data in DATA frame
            dataFrame := makeFrame(CMD_DATA, FLAG_NONE, buf[:n])
            err = ws.WriteMessage(websocket.BinaryMessage, dataFrame)
            if err != nil {
                log.Printf("WebSocket write error: %v", err)
                return
            }
        }
    }()

    // Goroutine 2: WebSocket -> Client (parse DATA frames)
    go func() {
        defer func() {
            done <- true
        }()

        for {
            _, message, err := ws.ReadMessage()
            if err != nil {
                log.Printf("WebSocket read error: %v", err)
                return
            }

            cmd, _, payload, ok := parseFrame(message)
            if !ok {
                log.Printf("Invalid frame received")
                return
            }

            switch cmd {
            case CMD_DATA:
                // Write payload to client
                _, err = clientConn.Write(payload)
                if err != nil {
                    log.Printf("Client write error: %v", err)
                    return
                }
            case CMD_CLOSE:
                log.Printf("Received CLOSE frame from server")
                return
            default:
                log.Printf("Unexpected frame command: %d", cmd)
            }
        }
    }()

    // Wait for either goroutine to finish
    <-done
    log.Printf("Tunnel closed for %s", target)
}

// makeFrame constructs a v2 protocol frame
// Frame structure:
//   Byte 0: MAGIC (0xA5)
//   Byte 1: VERSION (2)
//   Byte 2: CMD (AUTH=0x01, OPEN=0x02, DATA=0x03, CLOSE=0x04)
//   Byte 3: FLAGS (reserved)
//   Bytes 4-7: payload_len (32-bit big-endian)
//   Bytes 8+: payload
func makeFrame(cmd byte, flags byte, payload []byte) []byte {
    payloadLen := len(payload)
    frame := make([]byte, 8+payloadLen)

    frame[0] = MAGIC
    frame[1] = VERSION
    frame[2] = cmd
    frame[3] = flags
    binary.BigEndian.PutUint32(frame[4:8], uint32(payloadLen))

    if payloadLen > 0 {
        copy(frame[8:], payload)
    }

    return frame
}

// parseFrame parses a v2 protocol frame
// Returns: cmd, flags, payload, ok
func parseFrame(buffer []byte) (cmd byte, flags byte, payload []byte, ok bool) {
    if len(buffer) < 8 {
        return 0, 0, nil, false
    }

    // Verify MAGIC
    if buffer[0] != MAGIC {
        log.Printf("Invalid MAGIC: expected 0x%02X, got 0x%02X", MAGIC, buffer[0])
        return 0, 0, nil, false
    }

    // Verify VERSION
    version := buffer[1]
    if version != VERSION {
        log.Printf("Unsupported VERSION: expected %d, got %d", VERSION, version)
        return 0, 0, nil, false
    }

    cmd = buffer[2]
    flags = buffer[3]
    payloadLen := binary.BigEndian.Uint32(buffer[4:8])

    if len(buffer) < int(8+payloadLen) {
        log.Printf("Incomplete frame: expected %d bytes, got %d", 8+payloadLen, len(buffer))
        return 0, 0, nil, false
    }

    if payloadLen > 0 {
        payload = buffer[8 : 8+payloadLen]
    }

    return cmd, flags, payload, true
}

// dialWebSocketWithECH establishes WebSocket connection with ECH support
func dialWebSocketWithECH(wsURL string) (*websocket.Conn, error) {
    // Configure TLS with ECH support (Encrypted Client Hello)
    tlsConfig := &tls.Config{
        MinVersion: tls.VersionTLS12,
        // ECH support would require custom implementation or library
        // For now, use standard TLS configuration
    }

    dialer := &websocket.Dialer{
        TLSClientConfig:  tlsConfig,
        HandshakeTimeout: 15 * time.Second,
        ReadBufferSize:   32 * 1024,
        WriteBufferSize:  32 * 1024,
    }

    header := http.Header{}
    header.Set("User-Agent", "ProxyClient/2.0")

    conn, _, err := dialer.Dial(wsURL, header)
    if err != nil {
        return nil, fmt.Errorf("dial failed: %w", err)
    }

    return conn, nil
}

// getEnv retrieves environment variable or returns default value
func getEnv(key, defaultValue string) string {
    value := os.Getenv(key)
    if value == "" {
        return defaultValue
    }
    return value
}
