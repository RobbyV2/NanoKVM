package startup

import (
	"os"
	"strconv"
	"strings"
	"syscall"
)

// interruptDisposition reads what the kernel holds for SIGINT from
// /proc/self/status: "ignored", "default" or "caught", or "" when the file
// cannot be read. One read of a page, so it never grows into the allocation
// a large buffered read of a proc file costs on this kernel.
func interruptDisposition() string {
	file, err := os.Open("/proc/self/status")
	if err != nil {
		return ""
	}
	defer file.Close()
	buffer := make([]byte, 4096)
	n, err := file.Read(buffer)
	if err != nil {
		return ""
	}
	return dispositionFromStatus(string(buffer[:n]), syscall.SIGINT)
}

func dispositionFromStatus(status string, sig syscall.Signal) string {
	bit := uint64(1) << (uint(sig) - 1)
	masks := map[string]uint64{}
	for _, line := range strings.Split(status, "\n") {
		name, value, ok := strings.Cut(line, ":")
		if !ok || (name != "SigIgn" && name != "SigCgt") {
			continue
		}
		mask, err := strconv.ParseUint(strings.TrimSpace(value), 16, 64)
		if err != nil {
			return ""
		}
		masks[name] = mask
	}
	if len(masks) != 2 {
		return ""
	}
	switch {
	case masks["SigIgn"]&bit != 0:
		return "ignored"
	case masks["SigCgt"]&bit != 0:
		return "caught"
	}
	return "default"
}
