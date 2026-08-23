package vnc

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const (
	protocolVersion = "RFB 003.008\n"

	securityNone     uint8 = 1
	securityVeNCrypt uint8 = 19

	venCryptPlain uint32 = 256

	maxCredentialLength = 1024
)

var errAuthFailed = errors.New("invalid username or password")

type credentials struct {
	username string
	password string
}

func negotiateVersion(r *bufio.Reader, w io.Writer) error {
	if _, err := io.WriteString(w, protocolVersion); err != nil {
		return err
	}

	var version [12]byte
	if _, err := io.ReadFull(r, version[:]); err != nil {
		return err
	}
	if string(version[:]) != protocolVersion {
		return fmt.Errorf("unsupported protocol version %q", version[:])
	}
	return nil
}

// negotiateSecurity runs the security handshake and returns the credentials the
// client supplied, if any. It does not verify them.
func negotiateSecurity(r *bufio.Reader, w io.Writer, allowNone bool) (*credentials, error) {
	offered := securityVeNCrypt
	if allowNone {
		offered = securityNone
	}
	if _, err := w.Write([]byte{1, offered}); err != nil {
		return nil, err
	}

	chosen, err := r.ReadByte()
	if err != nil {
		return nil, err
	}
	if chosen != offered {
		return nil, fmt.Errorf("client selected unsupported security type %d", chosen)
	}
	if offered == securityNone {
		return nil, nil
	}

	return negotiateVeNCrypt(r, w)
}

func negotiateVeNCrypt(r *bufio.Reader, w io.Writer) (*credentials, error) {
	if _, err := w.Write([]byte{0, 2}); err != nil {
		return nil, err
	}

	var version [2]byte
	if _, err := io.ReadFull(r, version[:]); err != nil {
		return nil, err
	}
	if version[0] != 0 || version[1] < 2 {
		_, _ = w.Write([]byte{255})
		return nil, fmt.Errorf("unsupported VeNCrypt version %d.%d", version[0], version[1])
	}
	if _, err := w.Write([]byte{0}); err != nil {
		return nil, err
	}

	subtypes := make([]byte, 1, 5)
	subtypes[0] = 1
	subtypes = binary.BigEndian.AppendUint32(subtypes, venCryptPlain)
	if _, err := w.Write(subtypes); err != nil {
		return nil, err
	}

	var choice [4]byte
	if _, err := io.ReadFull(r, choice[:]); err != nil {
		return nil, err
	}
	if binary.BigEndian.Uint32(choice[:]) != venCryptPlain {
		return nil, fmt.Errorf("client selected unsupported VeNCrypt subtype %d", binary.BigEndian.Uint32(choice[:]))
	}

	var lengths [8]byte
	if _, err := io.ReadFull(r, lengths[:]); err != nil {
		return nil, err
	}
	usernameLen := binary.BigEndian.Uint32(lengths[0:4])
	passwordLen := binary.BigEndian.Uint32(lengths[4:8])
	if usernameLen > maxCredentialLength || passwordLen > maxCredentialLength {
		return nil, fmt.Errorf("credential too long: username %d, password %d", usernameLen, passwordLen)
	}

	buffer := make([]byte, usernameLen+passwordLen)
	if _, err := io.ReadFull(r, buffer); err != nil {
		return nil, err
	}

	return &credentials{
		username: string(buffer[:usernameLen]),
		password: string(buffer[usernameLen:]),
	}, nil
}

func writeSecurityResult(w io.Writer, err error) error {
	if err == nil {
		_, writeErr := w.Write([]byte{0, 0, 0, 0})
		return writeErr
	}

	reason := err.Error()
	message := make([]byte, 0, 8+len(reason))
	message = binary.BigEndian.AppendUint32(message, 1)
	message = binary.BigEndian.AppendUint32(message, uint32(len(reason)))
	message = append(message, reason...)
	_, writeErr := w.Write(message)
	return writeErr
}

func readClientInit(r *bufio.Reader) error {
	_, err := r.ReadByte()
	return err
}

func writeServerInit(w io.Writer, width uint16, height uint16, name string) error {
	message := make([]byte, 0, 24+len(name))
	message = binary.BigEndian.AppendUint16(message, width)
	message = binary.BigEndian.AppendUint16(message, height)
	message = append(message, serverPixelFormat()...)
	message = binary.BigEndian.AppendUint32(message, uint32(len(name)))
	message = append(message, name...)
	_, err := w.Write(message)
	return err
}

func serverPixelFormat() []byte {
	return []byte{
		32, 24, 0, 1,
		0, 255, 0, 255, 0, 255,
		16, 8, 0,
		0, 0, 0,
	}
}
