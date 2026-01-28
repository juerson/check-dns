package main

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// TestMakeFrame tests the frame construction function
func TestMakeFrame(t *testing.T) {
	tests := []struct {
		name    string
		cmd     byte
		flags   byte
		payload []byte
	}{
		{
			name:    "AUTH frame with token",
			cmd:     CMD_AUTH,
			flags:   FLAG_NONE,
			payload: []byte("test-token-123"),
		},
		{
			name:    "OPEN frame with target",
			cmd:     CMD_OPEN,
			flags:   FLAG_NONE,
			payload: []byte("example.com:443"),
		},
		{
			name:    "DATA frame with payload",
			cmd:     CMD_DATA,
			flags:   FLAG_NONE,
			payload: []byte("Hello, World!"),
		},
		{
			name:    "CLOSE frame without payload",
			cmd:     CMD_CLOSE,
			flags:   FLAG_NONE,
			payload: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame := makeFrame(tt.cmd, tt.flags, tt.payload)

			// Verify frame structure
			if frame[0] != MAGIC {
				t.Errorf("Expected MAGIC 0x%02X, got 0x%02X", MAGIC, frame[0])
			}

			if frame[1] != VERSION {
				t.Errorf("Expected VERSION %d, got %d", VERSION, frame[1])
			}

			if frame[2] != tt.cmd {
				t.Errorf("Expected CMD 0x%02X, got 0x%02X", tt.cmd, frame[2])
			}

			if frame[3] != tt.flags {
				t.Errorf("Expected FLAGS 0x%02X, got 0x%02X", tt.flags, frame[3])
			}

			payloadLen := binary.BigEndian.Uint32(frame[4:8])
			expectedLen := uint32(len(tt.payload))
			if payloadLen != expectedLen {
				t.Errorf("Expected payload length %d, got %d", expectedLen, payloadLen)
			}

			if tt.payload != nil {
				if !bytes.Equal(frame[8:], tt.payload) {
					t.Errorf("Payload mismatch: expected %v, got %v", tt.payload, frame[8:])
				}
			}
		})
	}
}

// TestParseFrame tests the frame parsing function
func TestParseFrame(t *testing.T) {
	tests := []struct {
		name           string
		frame          []byte
		expectedCmd    byte
		expectedFlags  byte
		expectedPayload []byte
		expectedOk     bool
	}{
		{
			name:           "Valid AUTH frame",
			frame:          makeFrame(CMD_AUTH, FLAG_NONE, []byte("token123")),
			expectedCmd:    CMD_AUTH,
			expectedFlags:  FLAG_NONE,
			expectedPayload: []byte("token123"),
			expectedOk:     true,
		},
		{
			name:           "Valid OPEN frame",
			frame:          makeFrame(CMD_OPEN, FLAG_NONE, []byte("host:port")),
			expectedCmd:    CMD_OPEN,
			expectedFlags:  FLAG_NONE,
			expectedPayload: []byte("host:port"),
			expectedOk:     true,
		},
		{
			name:           "Valid DATA frame",
			frame:          makeFrame(CMD_DATA, FLAG_NONE, []byte("data")),
			expectedCmd:    CMD_DATA,
			expectedFlags:  FLAG_NONE,
			expectedPayload: []byte("data"),
			expectedOk:     true,
		},
		{
			name:           "Valid CLOSE frame",
			frame:          makeFrame(CMD_CLOSE, FLAG_NONE, nil),
			expectedCmd:    CMD_CLOSE,
			expectedFlags:  FLAG_NONE,
			expectedPayload: nil,
			expectedOk:     true,
		},
		{
			name:       "Invalid frame - too short",
			frame:      []byte{0xA5, 0x02, 0x01},
			expectedOk: false,
		},
		{
			name:       "Invalid frame - wrong MAGIC",
			frame:      []byte{0xFF, 0x02, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00},
			expectedOk: false,
		},
		{
			name:       "Invalid frame - wrong VERSION",
			frame:      []byte{0xA5, 0x99, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00},
			expectedOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, flags, payload, ok := parseFrame(tt.frame)

			if ok != tt.expectedOk {
				t.Errorf("Expected ok=%v, got ok=%v", tt.expectedOk, ok)
				return
			}

			if !tt.expectedOk {
				return // Skip further checks for invalid frames
			}

			if cmd != tt.expectedCmd {
				t.Errorf("Expected CMD 0x%02X, got 0x%02X", tt.expectedCmd, cmd)
			}

			if flags != tt.expectedFlags {
				t.Errorf("Expected FLAGS 0x%02X, got 0x%02X", tt.expectedFlags, flags)
			}

			if !bytes.Equal(payload, tt.expectedPayload) {
				t.Errorf("Payload mismatch: expected %v, got %v", tt.expectedPayload, payload)
			}
		})
	}
}

// TestMakeParseRoundtrip tests making and parsing frames roundtrip
func TestMakeParseRoundtrip(t *testing.T) {
	testCases := []struct {
		cmd     byte
		flags   byte
		payload []byte
	}{
		{CMD_AUTH, FLAG_NONE, []byte("secret-token")},
		{CMD_OPEN, FLAG_NONE, []byte("example.com:443|GET / HTTP/1.1\r\n")},
		{CMD_DATA, FLAG_NONE, []byte("test data payload")},
		{CMD_CLOSE, FLAG_NONE, nil},
		{CMD_DATA, 0x01, []byte("data with flags")}, // Test with non-zero flags
	}

	for _, tc := range testCases {
		// Make frame
		frame := makeFrame(tc.cmd, tc.flags, tc.payload)

		// Parse frame
		cmd, flags, payload, ok := parseFrame(frame)

		if !ok {
			t.Errorf("Failed to parse frame: cmd=0x%02X", tc.cmd)
			continue
		}

		if cmd != tc.cmd {
			t.Errorf("CMD mismatch: expected 0x%02X, got 0x%02X", tc.cmd, cmd)
		}

		if flags != tc.flags {
			t.Errorf("FLAGS mismatch: expected 0x%02X, got 0x%02X", tc.flags, flags)
		}

		if !bytes.Equal(payload, tc.payload) {
			t.Errorf("Payload mismatch: expected %v, got %v", tc.payload, payload)
		}
	}
}

// TestFrameConstants verifies protocol constants
func TestFrameConstants(t *testing.T) {
	if MAGIC != 0xA5 {
		t.Errorf("MAGIC should be 0xA5, got 0x%02X", MAGIC)
	}

	if VERSION != 2 {
		t.Errorf("VERSION should be 2, got %d", VERSION)
	}

	if CMD_AUTH != 0x01 {
		t.Errorf("CMD_AUTH should be 0x01, got 0x%02X", CMD_AUTH)
	}

	if CMD_OPEN != 0x02 {
		t.Errorf("CMD_OPEN should be 0x02, got 0x%02X", CMD_OPEN)
	}

	if CMD_DATA != 0x03 {
		t.Errorf("CMD_DATA should be 0x03, got 0x%02X", CMD_DATA)
	}

	if CMD_CLOSE != 0x04 {
		t.Errorf("CMD_CLOSE should be 0x04, got 0x%02X", CMD_CLOSE)
	}

	if FLAG_NONE != 0x00 {
		t.Errorf("FLAG_NONE should be 0x00, got 0x%02X", FLAG_NONE)
	}
}

// TestLargePayload tests frames with large payloads
func TestLargePayload(t *testing.T) {
	// Create a large payload (1MB)
	largePayload := make([]byte, 1024*1024)
	for i := range largePayload {
		largePayload[i] = byte(i % 256)
	}

	frame := makeFrame(CMD_DATA, FLAG_NONE, largePayload)
	cmd, flags, payload, ok := parseFrame(frame)

	if !ok {
		t.Fatal("Failed to parse large frame")
	}

	if cmd != CMD_DATA {
		t.Errorf("Expected CMD_DATA, got 0x%02X", cmd)
	}

	if flags != FLAG_NONE {
		t.Errorf("Expected FLAG_NONE, got 0x%02X", flags)
	}

	if !bytes.Equal(payload, largePayload) {
		t.Error("Large payload mismatch")
	}
}

// TestEmptyPayload tests frames with empty payloads
func TestEmptyPayload(t *testing.T) {
	commands := []byte{CMD_AUTH, CMD_OPEN, CMD_DATA, CMD_CLOSE}

	for _, cmd := range commands {
		frame := makeFrame(cmd, FLAG_NONE, []byte{})
		parsedCmd, parsedFlags, parsedPayload, ok := parseFrame(frame)

		if !ok {
			t.Errorf("Failed to parse empty payload frame for CMD 0x%02X", cmd)
			continue
		}

		if parsedCmd != cmd {
			t.Errorf("CMD mismatch: expected 0x%02X, got 0x%02X", cmd, parsedCmd)
		}

		if parsedFlags != FLAG_NONE {
			t.Errorf("FLAGS mismatch: expected 0x%02X, got 0x%02X", FLAG_NONE, parsedFlags)
		}

		if len(parsedPayload) != 0 {
			t.Errorf("Expected empty payload, got %d bytes", len(parsedPayload))
		}
	}
}
