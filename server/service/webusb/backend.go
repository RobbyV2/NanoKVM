package webusb

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"

	"NanoKVM-Server/service/functionfs"
	"NanoKVM-Server/service/passthrough"
	"NanoKVM-Server/service/presentation"
	"NanoKVM-Server/service/sources"
)

type remoteStarter interface {
	StartRemoteHybrid(context.Context, string, passthrough.HybridRelay, presentation.FunctionFS) (*passthrough.Session, error)
	StopSession(*passthrough.Session) error
}

type Backend struct {
	store   *presentation.Store
	manager remoteStarter
	caps    func() presentation.CapabilityTable
}

func NewBackend() *Backend {
	return &Backend{store: presentation.NewStore(), manager: passthrough.GetManager(), caps: presentation.LoadCapabilities}
}

type Session struct {
	cancel context.CancelFunc
	device *Device
	ready  chan error
	done   chan error
	once   sync.Once

	mu          sync.Mutex
	passthrough *passthrough.Session
	manager     remoteStarter
}

func (b *Backend) Start(parent context.Context, stream sources.Stream, send func([]byte) error) (sources.BinarySession, error) {
	if stream.Kind != sources.KindUSBDevice || stream.USB == nil {
		return nil, errors.New("webusb: USB stream required")
	}
	ctx, cancel := context.WithCancel(parent)
	session := &Session{cancel: cancel, device: NewDevice(send), ready: make(chan error, 1), done: make(chan error, 1), manager: b.manager}
	go session.run(ctx, b, stream)
	return session, nil
}

func (s *Session) Receive(data []byte) error { return s.device.Receive(data) }
func (s *Session) Ready() <-chan error       { return s.ready }
func (s *Session) Done() <-chan error        { return s.done }

func (s *Session) Close() error {
	var result error
	s.once.Do(func() {
		s.cancel()
		s.mu.Lock()
		active := s.passthrough
		s.mu.Unlock()
		if active != nil {
			result = s.manager.StopSession(active)
		}
		result = errors.Join(result, s.device.Close())
	})
	return result
}

func (s *Session) run(ctx context.Context, backend *Backend, stream sources.Stream) {
	err := s.prepare(ctx, backend, stream)
	s.ready <- err
	close(s.ready)
	if err != nil {
		_ = s.device.Close()
		s.done <- err
		close(s.done)
		return
	}
	s.mu.Lock()
	active := s.passthrough
	s.mu.Unlock()
	select {
	case <-ctx.Done():
		_ = backend.manager.StopSession(active)
	case <-active.Done():
	}
	err = active.Err()
	s.done <- err
	close(s.done)
}

func (s *Session) prepare(ctx context.Context, backend *Backend, stream sources.Stream) error {
	offer := stream.USB
	profile, err := backend.store.LoadProfile(offer.Profile)
	if err != nil {
		return err
	}
	if profile.Name == "" || profile.Descriptors == nil || len(profile.Descriptors.Device) != 18 {
		return errors.New("webusb: captured profile is missing descriptors")
	}
	if err := profile.Validate(); err != nil {
		return fmt.Errorf("webusb: captured profile: %w", err)
	}
	configuration, index, err := configurationByValue(profile.Descriptors.Configurations, offer.Configuration)
	if err != nil {
		return err
	}
	liveDevice, err := s.device.Descriptor(1, 0, 0, 18)
	if err != nil || !bytes.Equal(liveDevice, profile.Descriptors.Device) {
		return errors.New("webusb: live device descriptor differs from the captured profile")
	}
	header, err := s.device.Descriptor(2, uint8(index), 0, 9)
	if err != nil || len(header) != 9 {
		return errors.New("webusb: cannot read live configuration header")
	}
	total := int(binary.LittleEndian.Uint16(header[2:4]))
	if total < 9 || total > functionfs.MaxDescriptorBytes {
		return errors.New("webusb: invalid live configuration length")
	}
	liveConfig, err := s.device.Descriptor(2, uint8(index), 0, total)
	if err != nil || !bytes.Equal(liveConfig, configuration) {
		return errors.New("webusb: live configuration differs from the captured profile")
	}
	raw, err := functionfs.Project(profile.Descriptors.Device, configuration, offer.Interfaces)
	if err != nil {
		return err
	}
	prepared, err := functionfs.PrepareRemote(raw, s.device, s.device, backend.caps())
	if err != nil {
		return err
	}
	active, err := backend.manager.StartRemoteHybrid(ctx, stream.Label, prepared.Relay, prepared.Image.Function)
	if err != nil {
		_ = prepared.Relay.Close()
		_ = functionfs.Cleanup()
		return err
	}
	s.mu.Lock()
	s.passthrough = active
	s.mu.Unlock()
	return nil
}

func configurationByValue(configurations [][]byte, value uint8) ([]byte, int, error) {
	for index, configuration := range configurations {
		if len(configuration) >= 9 && configuration[5] == value {
			return configuration, index, nil
		}
	}
	return nil, 0, fmt.Errorf("webusb: configuration %d is not captured", value)
}
