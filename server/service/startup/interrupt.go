package startup

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

var (
	interruptMu sync.Mutex
	interrupts  chan os.Signal
)

// Interrupts is the one channel the way out is read from: SIGINT, SIGTERM and
// SIGQUIT, through Go's handler. main creates it before anything reaches C,
// since ReassertInterrupt can only put back a handler that already exists.
func Interrupts() <-chan os.Signal {
	interruptMu.Lock()
	defer interruptMu.Unlock()
	if interrupts == nil {
		interrupts = make(chan os.Signal, 1)
		signal.Notify(interrupts, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	}
	return interrupts
}

// ReassertInterrupt puts Go's SIGINT handler back in front of the kernel.
//
// Nothing the server links installs a SIGINT handler of its own at run time:
// libkvm_mmf.so carries one in SAMPLE_PLAT_SYS_INIT (the vendor sample code's
// exit-on-signal handler, for INT and TERM both) that nothing calls, and
// libkvm.so's signal() calls are the MaixCDK constructor that catches crash
// signals. What does move SIGINT is system(3). kvm_vision.cpp in libkvm.so
// runs shells for every small write it makes (get_hdmi_mode, auto_try_res,
// get_manual_resolution, write_res_to_file, kvmv_hdmi_control), from the
// detection and watchdog threads kvmv_init starts and from the calls Go makes
// into it, and musl's system() sets SIGINT to SIG_IGN for the whole process
// while its child runs and then restores what it saved. A signal that lands
// during a shell is discarded, and two shells that overlap can leave SIGINT
// ignored for good, since the second saves the first's SIG_IGN and restores
// it last. SIGTERM is never touched, which is the shape measured on the
// device: an INT under a stream lost, the TERM twenty seconds later honoured.
//
// signal.Reset followed by signal.Notify does not reinstall the handler: the
// runtime keeps SIGINT marked as handled through a Reset, so the Notify that
// follows skips the sigaction. signal.Ignore clears that mark and the Notify
// after it installs the handler again. Between the two calls the signal is
// ignored for the microseconds the runtime takes, the same window a shell
// opens for its whole life, so this is called where a shell is likely to have
// just run, after the vision library's init and when a media worker starts a
// stream, and from KeepInterrupt on a timer for the cases with no such edge.
// The disposition found is reported when it was not a caught signal, which is
// the evidence the device log needs.
func ReassertInterrupt(where string) {
	interruptMu.Lock()
	defer interruptMu.Unlock()
	if interrupts == nil {
		return
	}
	found := interruptDisposition()
	signal.Ignore(syscall.SIGINT)
	signal.Notify(interrupts, syscall.SIGINT)
	if found == "" || found == "caught" {
		return
	}
	log.Warnf("interrupt: SIGINT was %s at %s, handler reinstalled", found, where)
	Kmsg("interrupt: SIGINT was %s at %s, handler reinstalled", found, where)
}

// KeepInterrupt reasserts the handler every interval, for the shells that run
// on no edge this side can see. It never returns.
func KeepInterrupt(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		ReassertInterrupt("periodic check")
	}
}
