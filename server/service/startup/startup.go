// Package startup bounds and isolates the hardware-touching work the server
// does before it listens. The web UI is the device's only recovery channel, so
// a probe that blocks or a subsystem that panics must degrade that subsystem
// and nothing else; the listener comes up either way, and what failed is
// readable through the API.
package startup

import (
	"fmt"
	"sync"
	"time"
)

type Status struct {
	Name  string `json:"name"`
	Error string `json:"error,omitempty"`
}

var (
	mu     sync.Mutex
	failed = map[string]string{}
	order  []string
)

// Run gives step a budget of its own. A step that overruns it is abandoned to
// keep running detached rather than joined, because the calls behind these
// budgets are cgo and sysfs writes with no cancellation of their own.
func Run(name string, budget time.Duration, step func() error) {
	record(name, "")

	done := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- fmt.Errorf("panic: %v", r)
			}
		}()
		done <- step()
	}()

	select {
	case err := <-done:
		if err != nil {
			record(name, err.Error())
		}
	case <-time.After(budget):
		record(name, fmt.Sprintf("timed out after %s", budget))
	}
}

func Fail(name string, err error) {
	if err == nil {
		record(name, "")
		return
	}
	record(name, err.Error())
}

func Report() []Status {
	mu.Lock()
	defer mu.Unlock()

	report := make([]Status, 0, len(order))
	for _, name := range order {
		report = append(report, Status{Name: name, Error: failed[name]})
	}
	return report
}

func record(name string, message string) {
	mu.Lock()
	defer mu.Unlock()

	if _, seen := failed[name]; !seen {
		order = append(order, name)
	}
	failed[name] = message
}
