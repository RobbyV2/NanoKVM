package startup

import (
	"fmt"
	"os"
	"strings"
)

// The init script starts the server with stdout and stderr on /dev/null and
// the logger opens no file by default, so a line that has to be read after
// the process is gone goes to the kernel log as well. Info level stays off the
// serial console at the board's loglevel and reaches the ring buffer and
// netconsole, where dmesg finds it.
var kmsgPath = "/dev/kmsg"

const (
	kmsgPrefix = "<6>NanoKVM-Server: "
	// One write is one record and the kernel truncates a long one, so the
	// line is cut here where the cut is visible.
	kmsgLineMax = 900
)

// Kmsg writes one line to the kernel log at info level and says nothing if it
// cannot: a missing or unwritable /dev/kmsg costs the line, never the caller.
func Kmsg(format string, args ...any) {
	file, err := os.OpenFile(kmsgPath, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		return
	}
	defer file.Close()
	line := strings.ReplaceAll(fmt.Sprintf(format, args...), "\n", " ")
	if len(line) > kmsgLineMax {
		line = line[:kmsgLineMax]
	}
	_, _ = file.WriteString(kmsgPrefix + line + "\n")
}
