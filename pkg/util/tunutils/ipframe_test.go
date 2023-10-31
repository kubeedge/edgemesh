package cni

import (
	"net"
	"testing"
)

func TestParseIPFrame(t *testing.T) {
	// create IPV4 buffer
	data := []byte{
		0x45, 0x00, 0x00, 0x3c, 0x1c, 0x46, 0x40, 0x00,
		0x40, 0x06, 0xd2, 0x1e, 0xc0, 0xa8, 0x00, 0x0a,
		0xc0, 0xa8, 0x00, 0x14,
	}

	// Check ParseIPFrame
	buffer := NewRecycleByteBuffer(65536)
	n := len(data)
	buffer.Write(data[:n])
	frame, err := ParseIPFrame(buffer)
	if err != nil {
		t.Fatalf("Error parsing IPFrame: %v", err)
	}

	if frame.Version != 4 {
		t.Errorf("Expected Version to be 4, got %d", frame.Version)
	}

	if frame.HeaderLen != 5 {
		t.Errorf("Expected HeaderLen to be 5, got %d", frame.HeaderLen)
	}

	if frame.Tos != 0 {
		t.Errorf("Expected Tos to be 0, got %d", frame.Tos)
	}

	if frame.TotalLen != 60 {
		t.Errorf("Expected TotalLen to be 60, got %d", frame.TotalLen)
	}

	if frame.Identification != 7238 {
		t.Errorf("Expected Identification to be 7238, got %d", frame.Identification)
	}

	if frame.Flag != 2 {
		t.Errorf("Expected Flag to be 2, got %d", frame.Flag)
	}

	if frame.Offset != 0 {
		t.Errorf("Expected Offset to be 0, got %d", frame.Offset)
	}

	if frame.Ttl != 64 {
		t.Errorf("Expected Ttl to be 64, got %d", frame.Ttl)
	}

	if frame.Protocol != 6 {
		t.Errorf("Expected Protocol to be 6, got %d", frame.Protocol)
	}

	if frame.HeaderCheckSum != 53758 {
		t.Errorf("Expected HeaderCheckSum to be 53758, got %d", frame.HeaderCheckSum)
	}

	if !frame.Source.Equal(net.IPv4(192, 168, 0, 10)) {
		t.Errorf("Expected Source to be 192.168.0.10, got %s", frame.Source)
	}

	if !frame.Target.Equal(net.IPv4(192, 168, 0, 20)) {
		t.Errorf("Expected Target to be 192.168.0.20, got %s", frame.Target)
	}
}
