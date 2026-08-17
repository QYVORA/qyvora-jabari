package adb

import (
	"testing"
)

func TestMessageEncodeDecode(t *testing.T) {
	tests := []struct {
		name string
		msg  *Message
	}{
		{
			name: "CNXN without data",
			msg: &Message{
				Command: A_CNXN,
				Arg0:    A_VERSION,
				Arg1:    MAX_PAYLOAD,
			},
		},
		{
			name: "CNXN with data",
			msg: &Message{
				Command: A_CNXN,
				Arg0:    A_VERSION,
				Arg1:    MAX_PAYLOAD,
				Data:    []byte("host::"),
			},
		},
		{
			name: "OPEN",
			msg: &Message{
				Command: A_OPEN,
				Arg0:    1,
				Arg1:    0,
				Data:    []byte("shell:ls\x00"),
			},
		},
		{
			name: "WRTE",
			msg: &Message{
				Command: A_WRTE,
				Arg0:    1,
				Arg1:    2,
				Data:    []byte("test data"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := tt.msg.Encode()
			decoded, err := DecodeMessage(encoded)
			if err != nil {
				t.Fatalf("DecodeMessage failed: %v", err)
			}

			if decoded.Command != tt.msg.Command {
				t.Errorf("Command mismatch: got 0x%08x, want 0x%08x", decoded.Command, tt.msg.Command)
			}
			if decoded.Arg0 != tt.msg.Arg0 {
				t.Errorf("Arg0 mismatch: got %d, want %d", decoded.Arg0, tt.msg.Arg0)
			}
			if decoded.Arg1 != tt.msg.Arg1 {
				t.Errorf("Arg1 mismatch: got %d, want %d", decoded.Arg1, tt.msg.Arg1)
			}
			if string(decoded.Data) != string(tt.msg.Data) {
				t.Errorf("Data mismatch: got %q, want %q", decoded.Data, tt.msg.Data)
			}
		})
	}
}

func TestMessageChecksum(t *testing.T) {
	msg := &Message{
		Command: A_WRTE,
		Data:    []byte("hello"),
	}

	expected := uint32('h' + 'e' + 'l' + 'l' + 'o')
	if got := msg.checksum(); got != expected {
		t.Errorf("checksum = %d, want %d", got, expected)
	}
}

func TestMessageString(t *testing.T) {
	tests := []struct {
		cmd  uint32
		want string
	}{
		{A_CNXN, "CNXN"},
		{A_AUTH, "AUTH"},
		{A_OPEN, "OPEN"},
		{A_OKAY, "OKAY"},
		{A_CLSE, "CLSE"},
		{A_WRTE, "WRTE"},
	}

	for _, tt := range tests {
		msg := &Message{Command: tt.cmd}
		if got := msg.String(); got != tt.want {
			t.Errorf("String() = %q, want %q", got, tt.want)
		}
	}
}

func TestDecodeMessageErrors(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "too short",
			data: []byte{1, 2, 3},
		},
		{
			name: "invalid magic",
			data: make([]byte, 24), // all zeros
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodeMessage(tt.data)
			if err == nil {
				t.Error("DecodeMessage should have failed")
			}
		})
	}
}
