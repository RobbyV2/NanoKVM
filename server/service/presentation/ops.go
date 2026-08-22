package presentation

import "context"

// Ops is the privilege boundary. Every configfs, /proc and platform-driver
// access in this package goes through it, and paths are relative to the gadget
// root, validated for no leading slash, no ".." segment and a first segment in
// {functions, configs, strings, os_desc, UDC} or a known device attribute.
//
// A future split moves this interface across an IPC boundary unchanged: the
// server hands a helper an already-compiled, already-validated Plan of []Op and
// the helper executes it, never parsing a profile and never trusting user JSON.
// Four things stay on the server side of that line, and are why the split buys
// less than it appears to: the pty in server/service/vm/terminal.go, the shell
// splice behind /api/vm/script/run, the /dev/hidg* descriptors held open for the
// process lifetime, and GPIO. Revisit when those are gone.
type Ops interface {
	Mkdir(rel string) error
	WriteFile(rel string, data []byte) error
	ReadFile(rel string) ([]byte, error)
	Symlink(target, linkRel string) error
	Remove(rel string) error
	ListUDC() ([]string, error)
	BindUDC(name string) error
	UnbindUDC() error
	SetOTGRole(role string) error
	ResetPHY(ctx context.Context) error
}
