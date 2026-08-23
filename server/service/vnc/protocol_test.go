package vnc

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"net"
	"testing"
)

func TestNegotiateSecurityOffersVeNCryptAndReadsPlainCredentials(t *testing.T) {
	client := new(bytes.Buffer)
	client.WriteByte(securityVeNCrypt)
	client.Write([]byte{0, 2})
	client.Write(binary.BigEndian.AppendUint32(nil, venCryptPlain))
	client.Write(binary.BigEndian.AppendUint32(nil, 5))
	client.Write(binary.BigEndian.AppendUint32(nil, 6))
	client.WriteString("admin")
	client.WriteString("secret")

	server := new(bytes.Buffer)
	creds, err := negotiateSecurity(bufio.NewReader(client), server, false)
	if err != nil {
		t.Fatalf("negotiate: %s", err)
	}
	if creds == nil || creds.username != "admin" || creds.password != "secret" {
		t.Fatalf("credentials = %+v, want admin/secret", creds)
	}

	written := server.Bytes()
	if written[0] != 1 || written[1] != securityVeNCrypt {
		t.Fatalf("security types = %v, want [1 19]", written[:2])
	}
	if written[2] != 0 || written[3] != 2 {
		t.Fatalf("vencrypt version = %v, want [0 2]", written[2:4])
	}
	if written[4] != 0 || written[5] != 1 {
		t.Fatalf("vencrypt ack/subtype count = %v, want [0 1]", written[4:6])
	}
	if binary.BigEndian.Uint32(written[6:10]) != venCryptPlain {
		t.Fatalf("subtype = %d, want %d", binary.BigEndian.Uint32(written[6:10]), venCryptPlain)
	}
}

func TestNegotiateSecurityOffersNoneWhenAuthenticationDisabled(t *testing.T) {
	client := bytes.NewBuffer([]byte{securityNone})
	server := new(bytes.Buffer)

	creds, err := negotiateSecurity(bufio.NewReader(client), server, true)
	if err != nil {
		t.Fatalf("negotiate: %s", err)
	}
	if creds != nil {
		t.Fatalf("credentials = %+v, want nil", creds)
	}
	if got := server.Bytes(); len(got) != 2 || got[0] != 1 || got[1] != securityNone {
		t.Fatalf("security types = %v, want [1 1]", got)
	}
}

func TestVerifyRejectsWrongPassword(t *testing.T) {
	restore := authenticate
	authenticate = func(username string, password string) (string, bool, error) {
		return "", username == "admin" && password == "right", nil
	}
	t.Cleanup(func() { authenticate = restore })

	server := &Server{}
	addr := &net.TCPAddr{IP: net.IPv4(127, 0, 0, 2), Port: 1234}

	if err := server.verify(addr, &credentials{username: "admin", password: "wrong"}); err == nil {
		t.Fatal("wrong password was accepted")
	}
	if err := server.verify(addr, &credentials{username: "admin", password: "right"}); err != nil {
		t.Fatalf("correct password was rejected: %s", err)
	}
}

func TestVerifyRequiresCredentialsUnlessAuthenticationIsDisabled(t *testing.T) {
	if err := (&Server{}).verify(&net.TCPAddr{}, nil); err == nil {
		t.Fatal("anonymous client was accepted while authentication is enabled")
	}
	if err := (&Server{AllowNone: true}).verify(&net.TCPAddr{}, nil); err != nil {
		t.Fatalf("anonymous client was rejected while authentication is disabled: %s", err)
	}
}

func TestWriteTightJPEGPassesEncoderOutputThrough(t *testing.T) {
	jpeg := bytes.Repeat([]byte{0xab}, 300)
	jpeg[0], jpeg[1] = 0xff, 0xd8

	out := new(bytes.Buffer)
	if err := writeTightJPEG(out, frame{data: jpeg, width: 1920, height: 1080}); err != nil {
		t.Fatalf("write: %s", err)
	}

	got := out.Bytes()
	want := []byte{0, 0, 0, 1, 0, 0, 0, 0, 0x07, 0x80, 0x04, 0x38, 0, 0, 0, 7, tightJPEG, 0xac, 0x02}
	if !bytes.Equal(got[:len(want)], want) {
		t.Fatalf("header = % x, want % x", got[:len(want)], want)
	}
	if !bytes.Equal(got[len(want):], jpeg) {
		t.Fatal("JPEG payload was not passed through byte for byte")
	}
}

func TestAppendCompactLength(t *testing.T) {
	cases := []struct {
		length int
		want   []byte
	}{
		{0x7f, []byte{0x7f}},
		{0x80, []byte{0x80, 0x01}},
		{0x3fff, []byte{0xff, 0x7f}},
		{0x4000, []byte{0x80, 0x80, 0x01}},
		{300, []byte{0xac, 0x02}},
	}

	for _, test := range cases {
		if got := appendCompactLength(nil, test.length); !bytes.Equal(got, test.want) {
			t.Fatalf("compact length %d = % x, want % x", test.length, got, test.want)
		}
	}
}

func TestKeyboardStateSynthesisesShift(t *testing.T) {
	keys := newKeyboardState()

	if got := keys.key('a', true); !bytes.Equal(got, []byte{0, 0, 0x04, 0, 0, 0, 0, 0}) {
		t.Fatalf("a down = % x", got)
	}
	if got := keys.key('a', false); !bytes.Equal(got, []byte{0, 0, 0, 0, 0, 0, 0, 0}) {
		t.Fatalf("a up = % x", got)
	}
	if got := keys.key('A', true); !bytes.Equal(got, []byte{modLeftShift, 0, 0x04, 0, 0, 0, 0, 0}) {
		t.Fatalf("A down = % x, want left shift set", got)
	}
	keys.key('A', false)

	keys.key(0xffe3, true)
	if got := keys.key('c', true); !bytes.Equal(got, []byte{modLeftControl, 0, 0x06, 0, 0, 0, 0, 0}) {
		t.Fatalf("ctrl+c = % x", got)
	}
	if got := keys.key(0xffe3, false); !bytes.Equal(got, []byte{0, 0, 0x06, 0, 0, 0, 0, 0}) {
		t.Fatalf("ctrl release = % x", got)
	}
	if keys.key(0x1008ff11, true) != nil {
		t.Fatal("an unmappable keysym produced a report")
	}
}

func TestPointerScalesAndEmitsWheelOnce(t *testing.T) {
	var pointer pointerState

	if got := pointer.pointer(rfbButtonLeft, 0, 0, 1920, 1080); !bytes.Equal(got, []byte{0x01, 0x01, 0x00, 0x01, 0x00, 0}) {
		t.Fatalf("top-left click = % x", got)
	}
	if got := pointer.pointer(0, 1919, 1079, 1920, 1080); !bytes.Equal(got, []byte{0x00, 0xff, 0x7f, 0xff, 0x7f, 0}) {
		t.Fatalf("bottom-right = % x, want the full 0x7fff range", got)
	}

	if got := pointer.pointer(rfbWheelUp, 0, 0, 1920, 1080); got[5] != 1 {
		t.Fatalf("wheel up = %d, want 1", got[5])
	}
	if got := pointer.pointer(rfbWheelUp, 0, 0, 1920, 1080); got[5] != 0 {
		t.Fatalf("held wheel bit repeated the tick: %d", got[5])
	}
	if got := pointer.pointer(rfbWheelDown, 0, 0, 1920, 1080); got[5] != 0xff {
		t.Fatalf("wheel down = %d, want 0xff", got[5])
	}
}
