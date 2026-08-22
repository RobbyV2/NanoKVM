package presentation

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

var ErrNotBound = errors.New("gadget did not bind")

// The transaction is unbind, mutate, bind, verify. It is add-only: no op ever
// rmdirs functions/* or removes the gadget root, because f_hid allocates the
// /dev/hidgN minor from an ida at mkdir time and hid/hid.go:29-32 hardcodes
// that mapping (H3, R1.1).
func (m *Manager) apply(ctx context.Context, profile Profile, plan Plan) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("apply %s: %w", profile.Name, err)
	}

	udcs, err := m.ops.ListUDC()
	if err != nil {
		return fmt.Errorf("apply %s: %w", profile.Name, err)
	}
	udc := udcs[0]

	before := readSnapshot(m.ops, profile.Functions)
	if err := m.ops.UnbindUDC(); err != nil {
		return fmt.Errorf("apply %s: unbind: %w", profile.Name, err)
	}
	if err := m.unlinkStale(before, plan); err != nil {
		return fmt.Errorf("apply %s: %w", profile.Name, err)
	}

	for i, op := range plan.Ops {
		if err := m.execute(op, udc); err != nil {
			return fmt.Errorf("apply %s: op %d %s %s: %w", profile.Name, i, op.Kind, op.Path, err)
		}
	}
	if err := m.verifyBind(udc); err != nil {
		return fmt.Errorf("apply %s: %w", profile.Name, err)
	}
	if err := mirrorSentinels(profile); err != nil {
		return fmt.Errorf("apply %s: %w", profile.Name, err)
	}
	if err := m.store.SetActive(profile.Name); err != nil {
		return fmt.Errorf("apply %s: %w", profile.Name, err)
	}
	return m.store.SetLastKnownGood(profile.Name)
}

func (m *Manager) execute(op Op, udc string) error {
	switch op.Kind {
	case OpMkdir:
		return m.ops.Mkdir(op.Path)
	case OpWrite:
		return m.ops.WriteFile(op.Path, op.Data)
	case OpSymlink:
		return m.ops.Symlink(op.Target, op.Path)
	case OpUnlink:
		return m.ops.Remove(op.Path)
	case OpBind:
		return m.ops.BindUDC(udc)
	case OpOTGRole:
		return m.ops.SetOTGRole(string(op.Data))
	default:
		return fmt.Errorf("unknown op kind %s", op.Kind)
	}
}

// Dropping a function is a symlink removal under configs/c.1 and nothing else.
// The hid links stay whatever the profile says, since unlinking them is the
// first half of the teardown that renumbers /dev/hidgN (R1.1).
func (m *Manager) unlinkStale(before Snapshot, plan Plan) error {
	linked := make(map[string]bool, len(plan.Ops))
	for _, op := range plan.Ops {
		if op.Kind == OpSymlink && strings.HasPrefix(op.Path, configPrefix+"/") {
			linked[strings.TrimPrefix(op.Path, configPrefix+"/")] = true
		}
	}

	for _, name := range before.Linked {
		if linked[name] || strings.HasPrefix(name, string(FunctionHID)+".") {
			continue
		}
		if err := m.ops.Remove(configPrefix + "/" + name); err != nil {
			return fmt.Errorf("unlink %s: %w", name, err)
		}
	}
	return nil
}

// H4: an empty UDC write is an unbind, so a gadget that silently never bound
// has to be caught by reading the attribute back rather than by the write
// returning nil.
func (m *Manager) verifyBind(udc string) error {
	data, err := m.ops.ReadFile(udcAttr)
	if err != nil {
		return fmt.Errorf("read %s: %w", udcAttr, err)
	}
	if bound := strings.TrimSpace(string(data)); bound != udc {
		return fmt.Errorf("%w: bound to %q, want %q", ErrNotBound, bound, udc)
	}
	return nil
}

func (m *Manager) bind() error {
	udcs, err := m.ops.ListUDC()
	if err != nil {
		return err
	}

	udc := udcs[0]
	if err := m.ops.BindUDC(udc); err != nil {
		return fmt.Errorf("bind %s: %w", udc, err)
	}
	if err := m.ops.SetOTGRole(OTGRoleDevice); err != nil {
		return err
	}
	return m.verifyBind(udc)
}

func (m *Manager) rebind(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("rebind: %w", err)
	}
	if err := m.ops.UnbindUDC(); err != nil {
		return fmt.Errorf("rebind: %w", err)
	}
	return m.bind()
}

func lunAttr(attr string) string {
	return functionsDir + "/" + diskFunctionName + "/" + lunDir + "/" + attr
}

// Ordering constraint 3: the LUN attributes carry no refcnt check, but ro and
// cdrom return -EBUSY while lun.0/file is open, so the file is released first.
// S03usbdev:136,142 rewrite removable and inquiry_string on every start, so this
// runtime state is deliberately not part of the profile (H7).
func (m *Manager) setLUN(ctx context.Context, lun LUN) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("set lun: %w", err)
	}

	if lun.File == "" || lun.CDROM {
		flag := []byte(boolBit(lun.File != "" && lun.CDROM))
		if err := m.ops.WriteFile(lunAttr("file"), []byte("\n")); err != nil {
			return fmt.Errorf("release backing file: %w", err)
		}
		for _, attr := range [...]string{"ro", "cdrom"} {
			if err := m.ops.WriteFile(lunAttr(attr), flag); err != nil {
				return fmt.Errorf("set %s: %w", lunAttr(attr), err)
			}
		}
	}
	if err := m.ops.WriteFile(lunAttr("inquiry_string"), []byte(lun.inquiry())); err != nil {
		return fmt.Errorf("set %s: %w", lunAttr("inquiry_string"), err)
	}

	if err := m.ops.WriteFile(lunAttr("file"), []byte(lun.backingFile())); err != nil {
		return fmt.Errorf("set %s: %w", lunAttr("file"), err)
	}
	return m.rebind(ctx)
}

func (m *Manager) readLUN() (LUN, error) {
	file, err := m.ops.ReadFile(lunAttr("file"))
	if err != nil {
		return LUN{}, fmt.Errorf("read %s: %w", lunAttr("file"), err)
	}
	cdrom, err := m.ops.ReadFile(lunAttr("cdrom"))
	if err != nil {
		return LUN{}, fmt.Errorf("read %s: %w", lunAttr("cdrom"), err)
	}

	lun := LUN{File: strings.TrimSpace(string(file)), CDROM: strings.TrimSpace(string(cdrom)) == "1"}
	if lun.File == DefaultDiskFile {
		lun.File = ""
	}
	return lun, nil
}
