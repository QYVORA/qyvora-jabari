// Package adb implements the native Android Debug Bridge protocol.
// This allows Jabari to communicate with Android devices without shelling
// out to the adb binary.
package adb

import (
	"encoding/binary"
	"fmt"
)

// ADB protocol constants
const (
	// Protocol version
	A_VERSION = 0x01000000

	// Message commands
	A_CNXN = 0x4e584e43 // CNXN
	A_AUTH = 0x48545541 // AUTH
	A_OPEN = 0x4e45504f // OPEN
	A_OKAY = 0x59414b4f // OKAY
	A_CLSE = 0x45534c43 // CLSE
	A_WRTE = 0x45545257 // WRTE

	// Auth types
	AUTH_TOKEN        = 1
	AUTH_SIGNATURE    = 2
	AUTH_RSAPUBLICKEY = 3

	// Maximum payload size
	MAX_PAYLOAD = 256 * 1024
)

// Message represents an ADB protocol message
type Message struct {
	Command uint32
	Arg0    uint32
	Arg1    uint32
	Data    []byte
}

// Encode serializes a message to wire format
func (m *Message) Encode() []byte {
	dataLen := uint32(len(m.Data))
	checksum := m.checksum()
	magic := m.Command ^ 0xffffffff

	buf := make([]byte, 24+dataLen)
	binary.LittleEndian.PutUint32(buf[0:4], m.Command)
	binary.LittleEndian.PutUint32(buf[4:8], m.Arg0)
	binary.LittleEndian.PutUint32(buf[8:12], m.Arg1)
	binary.LittleEndian.PutUint32(buf[12:16], dataLen)
	binary.LittleEndian.PutUint32(buf[16:20], checksum)
	binary.LittleEndian.PutUint32(buf[20:24], magic)
	copy(buf[24:], m.Data)

	return buf
}

// DecodeMessage parses a message from wire format
func DecodeMessage(buf []byte) (*Message, error) {
	if len(buf) < 24 {
		return nil, fmt.Errorf("message too short: %d bytes", len(buf))
	}

	m := &Message{
		Command: binary.LittleEndian.Uint32(buf[0:4]),
		Arg0:    binary.LittleEndian.Uint32(buf[4:8]),
		Arg1:    binary.LittleEndian.Uint32(buf[8:12]),
	}

	dataLen := binary.LittleEndian.Uint32(buf[12:16])
	checksum := binary.LittleEndian.Uint32(buf[16:20])
	magic := binary.LittleEndian.Uint32(buf[20:24])

	// Verify magic
	if magic != (m.Command ^ 0xffffffff) {
		return nil, fmt.Errorf("invalid magic: 0x%08x", magic)
	}

	// Read data
	if dataLen > 0 {
		if len(buf) < int(24+dataLen) {
			return nil, fmt.Errorf("incomplete data: expected %d, have %d", 24+dataLen, len(buf))
		}
		m.Data = buf[24 : 24+dataLen]

		// Verify checksum
		if m.checksum() != checksum {
			return nil, fmt.Errorf("checksum mismatch")
		}
	}

	return m, nil
}

// checksum computes the checksum for message data
func (m *Message) checksum() uint32 {
	var sum uint32
	for _, b := range m.Data {
		sum += uint32(b)
	}
	return sum
}

// String returns the command as a readable string
func (m *Message) String() string {
	cmd := make([]byte, 4)
	binary.LittleEndian.PutUint32(cmd, m.Command)
	return string(cmd)
}
