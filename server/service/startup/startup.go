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

	result := attempt(Step{Name: name, Budget: budget, Run: step})
	switch {
	case result.Abandoned:
		record(name, fmt.Sprintf("timed out after %s", budget))
	case result.Err != nil:
		record(name, result.Err.Error())
	}
}

// Step is one piece of bounded work with the name the log carries for it.
type Step struct {
	Name   string
	Budget time.Duration
	Run    func() error
}

// Result is what one Step came to. Abandoned means the budget ran out with the
// step still running; Elapsed is then the budget.
type Result struct {
	Name      string
	Elapsed   time.Duration
	Err       error
	Abandoned bool
}

func (r Result) String() string {
	switch {
	case r.Abandoned:
		return fmt.Sprintf("%s did not finish within %s", r.Name, r.Elapsed.Round(time.Millisecond))
	case r.Err != nil:
		return fmt.Sprintf("%s failed after %s: %s", r.Name, r.Elapsed.Round(time.Millisecond), r.Err)
	}
	return fmt.Sprintf("%s done in %s", r.Name, r.Elapsed.Round(time.Millisecond))
}

// Stop is Run for the way out: every step in order, each on its own budget,
// none of them able to keep the process alive. A step that overruns is left
// running and the next one starts, so the sum of the budgets is the most a
// signal can take to end the process, and the caller exits once the last
// result is in. The results say which step took the time, which is what the
// device log needs to attribute a slow stop; a step that was abandoned is
// finished by the kernel when the process exits, as a SIGKILL would have done.
func Stop(steps ...Step) []Result {
	results := make([]Result, 0, len(steps))
	for _, step := range steps {
		results = append(results, attempt(step))
	}
	return results
}

func attempt(step Step) Result {
	done := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- fmt.Errorf("panic: %v", r)
			}
		}()
		done <- step.Run()
	}()

	started := time.Now()
	timer := time.NewTimer(step.Budget)
	defer timer.Stop()
	select {
	case err := <-done:
		return Result{Name: step.Name, Elapsed: time.Since(started), Err: err}
	case <-timer.C:
		return Result{Name: step.Name, Elapsed: step.Budget, Abandoned: true}
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
