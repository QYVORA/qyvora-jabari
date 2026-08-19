package adb

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

// TCPConnection represents a TCP connection to an ADB daemon
type TCPConnection struct {
	addr        string
	conn        net.Conn
	mu          sync.Mutex
	localID     uint32
	remoteID    uint32
	nextLocalID uint32
}

// NewTCPConnection creates a new TCP ADB connection
func NewTCPConnection(addr string) *TCPConnection {
	return &TCPConnection{
		addr:        addr,
		nextLocalID: 1,
	}
}

// Connect establishes the TCP connection and performs the ADB handshake
func (c *TCPConnection) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		return nil // already connected
	}

	// Dial with timeout
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", c.addr)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	c.conn = conn

	// Send CNXN message
	systemIdentity := fmt.Sprintf("host::%d", A_VERSION)
	cnxn := &Message{
		Command: A_CNXN,
		Arg0:    A_VERSION,
		Arg1:    MAX_PAYLOAD,
		Data:    []byte(systemIdentity),
	}

	if err := c.writeMessage(cnxn); err != nil {
		_ = c.conn.Close()
		c.conn = nil
		return fmt.Errorf("send CNXN: %w", err)
	}

	// Read response (AUTH or CNXN)
	resp, err := c.readMessage(5 * time.Second)
	if err != nil {
		_ = c.conn.Close()
		c.conn = nil
		return fmt.Errorf("read response: %w", err)
	}

	switch resp.Command {
	case A_CNXN:
		// Connected successfully
		return nil
	case A_AUTH:
		// Device requires authentication
		// For now, we'll return an error indicating auth is needed
		// Full implementation would handle RSA key signing
		_ = c.conn.Close()
		c.conn = nil
		return fmt.Errorf("device requires authentication (not yet implemented)")
	default:
		_ = c.conn.Close()
		c.conn = nil
		return fmt.Errorf("unexpected response: %s", resp.String())
	}
}

// Close closes the TCP connection
func (c *TCPConnection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		return err
	}
	return nil
}

// OpenStream opens a stream to a service
func (c *TCPConnection) OpenStream(ctx context.Context, service string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return fmt.Errorf("not connected")
	}

	c.localID = c.nextLocalID
	c.nextLocalID++

	open := &Message{
		Command: A_OPEN,
		Arg0:    c.localID,
		Arg1:    0,
		Data:    append([]byte(service), 0), // null-terminated
	}

	if err := c.writeMessage(open); err != nil {
		return fmt.Errorf("send OPEN: %w", err)
	}

	// Wait for OKAY or CLSE
	resp, err := c.readMessage(5 * time.Second)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	switch resp.Command {
	case A_OKAY:
		c.remoteID = resp.Arg0
		return nil
	case A_CLSE:
		return fmt.Errorf("service rejected")
	default:
		return fmt.Errorf("unexpected response: %s", resp.String())
	}
}

// Write sends data on the open stream
func (c *TCPConnection) Write(data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return fmt.Errorf("not connected")
	}

	wrte := &Message{
		Command: A_WRTE,
		Arg0:    c.localID,
		Arg1:    c.remoteID,
		Data:    data,
	}

	if err := c.writeMessage(wrte); err != nil {
		return fmt.Errorf("send WRTE: %w", err)
	}

	// Wait for OKAY
	resp, err := c.readMessage(5 * time.Second)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.Command != A_OKAY {
		return fmt.Errorf("unexpected response: %s", resp.String())
	}

	return nil
}

// Read reads data from the open stream
func (c *TCPConnection) Read(timeout time.Duration) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return nil, fmt.Errorf("not connected")
	}

	resp, err := c.readMessage(timeout)
	if err != nil {
		return nil, err
	}

	switch resp.Command {
	case A_WRTE:
		// Send OKAY acknowledgment
		okay := &Message{
			Command: A_OKAY,
			Arg0:    c.localID,
			Arg1:    c.remoteID,
		}
		if err := c.writeMessage(okay); err != nil {
			return nil, fmt.Errorf("send OKAY: %w", err)
		}
		return resp.Data, nil
	case A_CLSE:
		return nil, io.EOF
	default:
		return nil, fmt.Errorf("unexpected message: %s", resp.String())
	}
}

// CloseStream closes the current stream
func (c *TCPConnection) CloseStream() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return fmt.Errorf("not connected")
	}

	clse := &Message{
		Command: A_CLSE,
		Arg0:    c.localID,
		Arg1:    c.remoteID,
	}

	return c.writeMessage(clse)
}

// Shell executes a shell command and returns the output
func (c *TCPConnection) Shell(ctx context.Context, command string) ([]byte, error) {
	service := "shell:" + command
	if err := c.OpenStream(ctx, service); err != nil {
		return nil, err
	}
	defer func() { _ = c.CloseStream() }()

	var result []byte
	for {
		data, err := c.Read(30 * time.Second)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		result = append(result, data...)
	}

	return result, nil
}

// writeMessage sends a message on the connection
func (c *TCPConnection) writeMessage(m *Message) error {
	data := m.Encode()
	_, err := c.conn.Write(data)
	return err
}

// readMessage reads a message from the connection
func (c *TCPConnection) readMessage(timeout time.Duration) (*Message, error) {
	if err := c.conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return nil, err
	}

	// Read header (24 bytes)
	header := make([]byte, 24)
	if _, err := io.ReadFull(c.conn, header); err != nil {
		return nil, err
	}

	// Parse header to get data length
	dataLen := binary.LittleEndian.Uint32(header[12:16])

	// Read data if present
	buf := make([]byte, 24+dataLen)
	copy(buf, header)
	if dataLen > 0 {
		if _, err := io.ReadFull(c.conn, buf[24:]); err != nil {
			return nil, err
		}
	}

	return DecodeMessage(buf)
}
