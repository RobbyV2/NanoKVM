package media

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"NanoKVM-Server/service/presentation"
	"NanoKVM-Server/service/sources"
)

// An output whose Run ignores cancellation, which is what a worker wedged
// inside a kernel call looks like from here.
type stuckOutput struct {
	closed  chan struct{}
	release chan struct{}
}

func (o *stuckOutput) Run(context.Context, <-chan Packet, Fallback, func(sources.Demand), func(bool)) error {
	<-o.release
	return nil
}

func (o *stuckOutput) Close() error {
	select {
	case <-o.closed:
	default:
		close(o.closed)
	}
	return nil
}

type stuckFactory struct{ output *stuckOutput }

func (f *stuckFactory) Open(SlotSpec, string) (Output, error) {
	f.output = &stuckOutput{closed: make(chan struct{}), release: make(chan struct{})}
	return f.output, nil
}

func (f *stuckFactory) OpenInput(SlotSpec, string) (Input, error) {
	return nil, errors.New("no capture in this factory")
}

// An output whose Close never returns, the shape Close takes when a worker is
// stuck inside the C layer holding the output's own mutex.
type deafOutput struct{ release chan struct{} }

func (o *deafOutput) Run(ctx context.Context, _ <-chan Packet, _ Fallback, _ func(sources.Demand), _ func(bool)) error {
	<-ctx.Done()
	return nil
}

func (o *deafOutput) Close() error {
	<-o.release
	return nil
}

type deafFactory struct{ output *deafOutput }

func (f *deafFactory) Open(SlotSpec, string) (Output, error) {
	f.output = &deafOutput{release: make(chan struct{})}
	return f.output, nil
}

func (f *deafFactory) OpenInput(SlotSpec, string) (Input, error) {
	return nil, errors.New("no capture in this factory")
}

func cameraManager(t *testing.T, factory OutputFactory, open map[string]int) *Manager {
	t.Helper()
	resolver := fakeResolver{nodes: map[string]string{"uvc.cam0": "/dev/video0"}}
	manager := newTestManager(&fakeRegistry{}, resolver, factory)
	manager.openNodes = func() (map[string]int, error) {
		result := make(map[string]int, len(open))
		for node, count := range open {
			result[node] = count
		}
		return result, nil
	}
	profile := presentation.Profile{Functions: []presentation.Function{cameraFunction("cam0")}}
	if err := manager.Reconcile(context.Background(), profile, presentation.Plan{}); err != nil {
		t.Fatal(err)
	}
	return manager
}

// configfs does not refuse an unlink whose video node is still open: it blocks
// in the kernel, where no Go context reaches it. So Suspend has to be able to
// say it failed, and the whole point is that saying so is not optional.
func TestSuspendReportsANodeThatIsStillOpen(t *testing.T) {
	manager := cameraManager(t, &fakeFactory{}, map[string]int{"/dev/video0": 1})
	err := manager.Suspend()
	if !errors.Is(err, ErrNodeBusy) {
		t.Fatalf("Suspend() = %v, want ErrNodeBusy", err)
	}
	if !strings.Contains(err.Error(), "/dev/video0") {
		t.Fatalf("Suspend() = %v, want the node named", err)
	}
}

func TestSuspendSucceedsWhenEveryNodeIsClosed(t *testing.T) {
	manager := cameraManager(t, &fakeFactory{}, nil)
	if err := manager.Suspend(); err != nil {
		t.Fatalf("Suspend() = %v, want nil", err)
	}
}

func TestSuspendReportsAWorkerThatWillNotStop(t *testing.T) {
	factory := &stuckFactory{}
	manager := cameraManager(t, factory, nil)
	defer close(factory.output.release)

	start := time.Now()
	err := manager.Suspend()
	if elapsed := time.Since(start); elapsed > 3*stopTimeout {
		t.Fatalf("Suspend() took %s, want a bounded wait", elapsed)
	}
	if !errors.Is(err, ErrNodeBusy) || !strings.Contains(err.Error(), "uvc.cam0") {
		t.Fatalf("Suspend() = %v, want a refusal naming uvc.cam0", err)
	}
}

// Close itself can block, because it takes the same mutex the wedged worker
// holds. A teardown that waits on it is the hang all over again.
func TestSuspendDoesNotWaitOnAnOutputThatWillNotClose(t *testing.T) {
	factory := &deafFactory{}
	manager := cameraManager(t, factory, nil)
	defer close(factory.output.release)

	done := make(chan error, 1)
	go func() { done <- manager.Suspend() }()
	select {
	case <-done:
	case <-time.After(3 * stopTimeout):
		t.Fatal("Suspend blocked on an output that would not close")
	}
}
