package sni

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"
)

func TestReadServerName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		record []byte
		want   string
	}{
		{
			name:   "extracts host name",
			record: tlsRecord(clientHello([]serverNameEntry{{nameType: 0, name: "example.com"}}, nil, false)),
			want:   "example.com",
		},
		{
			name:   "ignores unsupported name type",
			record: tlsRecord(clientHello([]serverNameEntry{{nameType: 1, name: "ignored"}, {nameType: 0, name: "example.com"}}, nil, false)),
			want:   "example.com",
		},
		{
			name:   "rejects duplicate host names",
			record: tlsRecord(clientHello([]serverNameEntry{{nameType: 0, name: "example.com"}, {nameType: 0, name: "example.net"}}, nil, false)),
			want:   "",
		},
		{
			name:   "rejects trailing client hello data",
			record: tlsRecord(clientHello([]serverNameEntry{{nameType: 0, name: "example.com"}}, nil, true)),
			want:   "",
		},
		{
			name:   "returns empty for non handshake record",
			record: []byte{23, 0x03, 0x03, 0x00, 0x00},
			want:   "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := readServerName(bytes.NewReader(test.record))
			if got != test.want {
				t.Fatalf("readServerName() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestPeekServerNameBuffersConsumedBytes(t *testing.T) {
	t.Parallel()

	record := tlsRecord(clientHello([]serverNameEntry{{nameType: 0, name: "example.com"}}, nil, false))
	payload := []byte("payload")
	original := append(append([]byte(nil), record...), payload...)
	source := bytes.NewReader(original)

	serverName, reader, err := peekServerName(source)
	if err != nil {
		t.Fatalf("peekServerName() error = %v", err)
	}
	if serverName != "example.com" {
		t.Fatalf("peekServerName() serverName = %q, want %q", serverName, "example.com")
	}

	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("io.ReadAll() error = %v", err)
	}
	if !bytes.Equal(got, record) {
		t.Fatalf("peekServerName() reader returned %x, want %x", got, record)
	}

	got, err = io.ReadAll(source)
	if err != nil {
		t.Fatalf("io.ReadAll(source) error = %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("source reader returned %x, want %x", got, payload)
	}
}

type serverNameEntry struct {
	nameType uint8
	name     string
}

func tlsRecord(clientHello []byte) []byte {
	handshake := append([]byte{1}, uint24(len(clientHello))...)
	handshake = append(handshake, clientHello...)

	record := []byte{22, 0x03, 0x03}
	record = append(record, uint16Bytes(len(handshake))...)
	record = append(record, handshake...)
	return record
}

func clientHello(serverNames []serverNameEntry, extraExtensions []byte, addTrailingByte bool) []byte {
	clientHello := []byte{0x03, 0x03}
	clientHello = append(clientHello, bytes.Repeat([]byte{0x42}, 32)...)
	clientHello = append(clientHello, 0)
	clientHello = append(clientHello, 0x00, 0x02, 0x13, 0x01)
	clientHello = append(clientHello, 0x01, 0x00)

	extensions := serverNameExtension(serverNames)
	extensions = append(extensions, extraExtensions...)
	clientHello = append(clientHello, uint16Bytes(len(extensions))...)
	clientHello = append(clientHello, extensions...)

	if addTrailingByte {
		clientHello = append(clientHello, 0xff)
	}

	return clientHello
}

func serverNameExtension(serverNames []serverNameEntry) []byte {
	var serverNameList []byte
	for _, serverName := range serverNames {
		serverNameList = append(serverNameList, serverName.nameType)
		serverNameList = append(serverNameList, uint16Bytes(len(serverName.name))...)
		serverNameList = append(serverNameList, serverName.name...)
	}

	extensionData := append(uint16Bytes(len(serverNameList)), serverNameList...)
	extension := uint16Bytes(0)
	extension = append(extension, uint16Bytes(len(extensionData))...)
	extension = append(extension, extensionData...)
	return extension
}

func uint16Bytes(value int) []byte {
	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf, uint16(value))
	return buf
}

func uint24(value int) []byte {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, uint32(value))
	return buf[1:]
}
