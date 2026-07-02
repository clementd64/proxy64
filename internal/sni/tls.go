package sni

import (
	"bytes"
	"errors"
	"io"

	"golang.org/x/crypto/cryptobyte"
)

func peekServerName(reader io.Reader) (string, io.Reader, error) {
	buffer := new(bytes.Buffer)
	serverName := readServerName(io.TeeReader(reader, buffer))
	return serverName, buffer, nil
}

func readServerName(reader io.Reader) string {
	fragment, err := readTLSPlaintext(reader)
	if err != nil || fragment == nil {
		return ""
	}

	clientHello, err := readHandshake(fragment)
	if err != nil || clientHello == nil {
		return ""
	}

	return readClientHello(clientHello)
}

// struct {
//   ContentType type; // uint8
//   ProtocolVersion legacy_record_version; // uint16
//   uint16 length;
//   opaque fragment[TLSPlaintext.length];
// } TLSPlaintext;

func readTLSPlaintext(reader io.Reader) (cryptobyte.String, error) {
	header, err := readN(reader, 5)
	if err != nil {
		return nil, err
	}

	var contentType uint8
	if !header.ReadUint8(&contentType) || contentType != 22 /* handshake */ {
		return nil, nil
	}

	// legacy record version
	if !header.Skip(2) {
		return nil, nil
	}

	var length uint16
	if !header.ReadUint16(&length) {
		return nil, nil
	}

	fragment, err := readN(reader, uint32(length))
	if err != nil {
		return nil, err
	}

	return fragment, nil
}

// struct {
//   HandshakeType msg_type; // uint8
//   uint24 length;
//   select (Handshake.msg_type) {
//     case client_hello:          ClientHello;
//     // ...
//   };
// } Handshake;

func readHandshake(fragment cryptobyte.String) (cryptobyte.String, error) {
	var msgType uint8
	if !fragment.ReadUint8(&msgType) || msgType != 1 /* client hello */ {
		return nil, nil
	}

	var length uint32
	if !fragment.ReadUint24(&length) {
		return nil, nil
	}

	if length > 16384-32 /* 2^14 - 32 */ {
		// TODO: handle multi-fragment handshake
		return nil, errors.New("multi-fragment handshake not supported")
	}

	var clientHello []byte
	if !fragment.ReadBytes(&clientHello, int(length)) {
		return nil, nil
	}

	return clientHello, nil
}

// struct {
//   ProtocolVersion legacy_version; // uint16
//   Random random; // opaque[32]
//   opaque legacy_session_id<0..32>;
//   CipherSuite cipher_suites<2..2^16-2>;
//   opaque legacy_compression_methods<1..2^8-1>;
//   Extension extensions<8..2^16-1>;
// } ClientHello;

func readClientHello(clientHello cryptobyte.String) (serverName string) {
	// legacy version
	if !clientHello.Skip(2) {
		return ""
	}

	// random
	if !clientHello.Skip(32) {
		return ""
	}

	// legacy session id
	var buf cryptobyte.String
	if !clientHello.ReadUint8LengthPrefixed(&buf) {
		return ""
	}

	// cipher suites
	if !clientHello.ReadUint16LengthPrefixed(&buf) {
		return ""
	}

	// legacy compression methods
	if !clientHello.ReadUint8LengthPrefixed(&buf) {
		return ""
	}

	var extensions cryptobyte.String
	if !clientHello.ReadUint16LengthPrefixed(&extensions) {
		return ""
	}

	// Last field, exit if remaining data (invalid client hello)
	for !clientHello.Empty() {
		return ""
	}

	for !extensions.Empty() {
		var extensionType uint16
		if !extensions.ReadUint16(&extensionType) {
			return ""
		}
		var extensionData cryptobyte.String
		if !extensions.ReadUint16LengthPrefixed(&extensionData) {
			return ""
		}

		if extensionType != 0 /* server name */ {
			continue
		}

		var serverNameList cryptobyte.String
		if !extensionData.ReadUint16LengthPrefixed(&serverNameList) {
			return ""
		}

		for !serverNameList.Empty() {
			var nameType uint8
			if !serverNameList.ReadUint8(&nameType) {
				return ""
			}

			var serverNameBytes []byte
			if !serverNameList.ReadUint16LengthPrefixed((*cryptobyte.String)(&serverNameBytes)) {
				return ""
			}

			// type other than 0 (host_name) is not supported
			if nameType != 0 /* host_name */ {
				continue
			}

			// only one server name can be specified
			if serverName != "" {
				return ""
			}
			serverName = string(serverNameBytes)
			// continue reading to ensure handshake is valid
		}
	}

	return
}

func readN(reader io.Reader, n uint32) (cryptobyte.String, error) {
	buf := make([]byte, n)
	_, err := io.ReadFull(reader, buf)
	return buf, err
}
