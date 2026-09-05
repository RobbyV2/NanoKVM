package vnc

import (
	"fmt"
	"net"
	"sync"
	"time"

	"NanoKVM-Server/authn"
	"NanoKVM-Server/service/auth"
	"NanoKVM-Server/service/controlmode"
	"NanoKVM-Server/utils"

	log "github.com/sirupsen/logrus"
)

const (
	maxConnections = 4
	authFailDelay  = 2 * time.Second
)

var authenticate = func(username string, password string) (string, bool, error) {
	user, ok, err := authn.DefaultStore.Authenticate(username, password)
	if err != nil || !ok {
		return "", ok, err
	}
	return user.Username, true, nil
}

// Server speaks RFB 3.8 over a plain TCP socket. Frames, HDMI demand reporting
// and the PicoClaw input gate are injected because their packages link against
// libkvm and cannot be built or tested off the device.
type Server struct {
	Addr        string
	AllowNone   bool
	Screen      func() (width uint16, height uint16, quality uint16, fps int)
	ReadJPEG    func(width uint16, height uint16, quality uint16) ([]byte, int)
	AllowInput  func(controlmode.Mode) bool
	ViewerCount func(count int, version uint64)

	pump      *framePump
	authMutex sync.Mutex

	mutex   sync.Mutex
	viewers int
	version uint64
}

func (s *Server) ListenAndServe() error {
	listener, err := utils.Listen(s.Addr)
	if err != nil {
		return err
	}
	log.Infof("vnc server listening on %s", s.Addr)

	return s.Serve(listener)
}

func (s *Server) Serve(listener net.Listener) error {
	s.pump = newFramePump(s.Screen, s.ReadJPEG)

	slots := make(chan struct{}, maxConnections)
	for {
		netConn, err := listener.Accept()
		if err != nil {
			return err
		}

		select {
		case slots <- struct{}{}:
		default:
			log.Warnf("vnc connection from %s rejected: too many clients", netConn.RemoteAddr())
			_ = netConn.Close()
			continue
		}

		go func() {
			defer func() { <-slots }()
			s.addViewer(1)
			defer s.addViewer(-1)
			newConn(s, netConn).serve()
		}()
	}
}

func (s *Server) addViewer(delta int) {
	if s.ViewerCount == nil {
		return
	}

	s.mutex.Lock()
	s.viewers += delta
	if s.viewers < 0 {
		s.viewers = 0
	}
	s.version++
	count, version := s.viewers, s.version
	s.mutex.Unlock()

	s.ViewerCount(count, version)
}

// verify serialises credential checks because the account store holds a write
// lock across a bcrypt comparison, which costs about a second on this SoC.
func (s *Server) verify(addr net.Addr, creds *credentials) error {
	if creds == nil {
		if !s.AllowNone {
			return fmt.Errorf("authentication required")
		}
		log.Warnf("vnc client %s connected without authentication", addr)
		return nil
	}

	ip := clientIP(addr)
	if locked, _, message := auth.CheckLoginAttempt(ip); locked {
		time.Sleep(authFailDelay)
		return fmt.Errorf("%s", message)
	}

	s.authMutex.Lock()
	user, ok, err := authenticate(creds.username, creds.password)
	s.authMutex.Unlock()

	if err != nil {
		log.Errorf("load account during vnc login: %s", err)
		return fmt.Errorf("authentication unavailable")
	}
	if !ok {
		time.Sleep(authFailDelay)
		auth.RecordLoginFailure(ip)
		return errAuthFailed
	}

	auth.ClearLoginAttempt(ip)
	log.Infof("vnc user logged in: %s", user)
	return nil
}

func clientIP(addr net.Addr) string {
	host, _, err := net.SplitHostPort(addr.String())
	if err != nil {
		return addr.String()
	}
	return host
}
