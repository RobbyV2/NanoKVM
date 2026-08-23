package presentation

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
)

var ErrNotBound = errors.New("gadget did not bind")

type recoveryPlan struct {
	profile Profile
	plan    Plan
}

// The transaction is unbind, mutate, bind, verify. It is add-only: no op ever
// rmdirs functions/* or removes the gadget root, because f_hid allocates the
// /dev/hidgN minor from an ida at mkdir time and hid/hid.go:29-32 hardcodes
// that mapping (H3, R1.1).
func (m *Manager) apply(ctx context.Context, profile Profile, plan Plan) error {
	_, _, err := m.applyPlan(ctx, profile, plan, true)
	return err
}

func (m *Manager) applyPlan(ctx context.Context, profile Profile, plan Plan, persist bool) (recoveryPlan, string, error) {
	if err := ctx.Err(); err != nil {
		return recoveryPlan{}, "", fmt.Errorf("apply %s: %w", profile.Name, err)
	}
	recovery, err := m.prepareRecovery()
	if err != nil {
		return recoveryPlan{}, "", fmt.Errorf("apply %s: prepare rollback: %w", profile.Name, err)
	}

	udcs, err := m.ops.ListUDC()
	if err != nil {
		return recoveryPlan{}, "", fmt.Errorf("apply %s: %w", profile.Name, err)
	}
	udc := udcs[0]

	probes := append(append([]Function(nil), profile.Functions...), recovery.profile.Functions...)
	before := readSnapshot(m.ops, probes)
	plan = m.dropRedundantWrites(before, plan)
	if err := m.unbindIfBound(); err != nil {
		applyErr := fmt.Errorf("apply %s: unbind: %w", profile.Name, err)
		if bindErr := m.ensureBound(udc); bindErr != nil {
			return recoveryPlan{}, udc, errors.Join(applyErr, fmt.Errorf("restore binding: %w", bindErr))
		}
		return recoveryPlan{}, udc, applyErr
	}
	if err := m.unlinkStale(before, plan); err != nil {
		return recoveryPlan{}, udc, m.rollbackFailure(profile, recovery, udc, err)
	}

	for i, op := range plan.Ops {
		if err := m.execute(op, udc); err != nil {
			return recoveryPlan{}, udc, m.rollbackFailure(profile, recovery, udc,
				fmt.Errorf("op %d %s %s: %w", i, op.Kind, op.Path, err))
		}
	}
	if err := m.verify(udc, profile); err != nil {
		return recoveryPlan{}, udc, m.rollbackFailure(profile, recovery, udc, err)
	}
	if !persist {
		return recovery, udc, nil
	}
	if err := m.store.SaveProfile(profile); err != nil {
		return recoveryPlan{}, udc, m.rollbackFailure(profile, recovery, udc, err)
	}
	if err := mirrorSentinels(profile); err != nil {
		return recoveryPlan{}, udc, m.rollbackFailure(profile, recovery, udc, err)
	}
	if err := m.store.SetActive(profile.Name); err != nil {
		return recoveryPlan{}, udc, m.rollbackFailure(profile, recovery, udc, err)
	}
	if err := m.store.SetLastKnownGood(profile.Name); err != nil {
		return recoveryPlan{}, udc, m.rollbackFailure(profile, recovery, udc, err)
	}
	return recovery, udc, nil
}

func (m *Manager) restoreFunctionFS(ctx context.Context, state *Transient) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := m.ops.UnbindUDC(); err != nil {
		return err
	}
	if err := m.ops.Remove(configPrefix + "/ffs.hybrid"); err != nil {
		return errors.Join(err, m.ensureBound(state.udc))
	}
	return m.restore(state.Profile, state.recovery, state.udc)
}

func (m *Manager) prepareRecovery() (recoveryPlan, error) {
	name, err := m.store.LastKnownGood()
	if err != nil {
		return recoveryPlan{}, err
	}
	if name == "" {
		name, err = m.store.Active()
		if err != nil {
			return recoveryPlan{}, err
		}
	}

	profile := standardProfile()
	if name != "" {
		profile, err = m.store.LoadProfile(name)
		if err != nil {
			return recoveryPlan{}, err
		}
		if profile.Name != name {
			return recoveryPlan{}, fmt.Errorf("profile %q contains name %q", name, profile.Name)
		}
	}
	plan, err := Compile(profile, m.caps)
	if err != nil {
		return recoveryPlan{}, err
	}
	return recoveryPlan{profile: profile, plan: plan}, nil
}

func (m *Manager) rollbackFailure(failed Profile, recovery recoveryPlan, udc string, cause error) error {
	applyErr := fmt.Errorf("apply %s: %w", failed.Name, cause)
	err := m.restore(failed, recovery, udc)
	if err == nil {
		return fmt.Errorf("%w; rolled back to %s", applyErr, recovery.profile.Name)
	}

	rollbackErr := fmt.Errorf("rollback to %s: %w", recovery.profile.Name, err)
	// The last rung. The target failed and so did the rollback, so the controller
	// carries whatever the half-finished transaction left on it. hid-only is built
	// from code rather than from the store and links nothing but the three HID
	// functions that already exist, so it is the smallest gadget this package can
	// put back and the operator keeps a keyboard instead of a dead port. It cannot
	// wedge what it did not find wedged: restore never calls back into here, so
	// there is no second attempt, and a failure still lands on the bind restore
	// already defers.
	fallback, compileErr := hidOnlyRecovery(m.caps)
	if recovery.profile.Name == ProfileHIDOnly || compileErr != nil {
		return errors.Join(applyErr, rollbackErr, compileErr)
	}
	// The rollback that just failed may have linked its own functions before it
	// did, so the stale set this rung has to unlink is both profiles.
	stale := failed
	stale.Functions = append(append([]Function(nil), failed.Functions...), recovery.profile.Functions...)
	if err := m.restore(stale, fallback, udc); err != nil {
		return errors.Join(applyErr, rollbackErr, fmt.Errorf("fall back to %s: %w", ProfileHIDOnly, err))
	}
	return errors.Join(applyErr, rollbackErr, fmt.Errorf("fell back to %s", ProfileHIDOnly))
}

func hidOnlyRecovery(caps CapabilityTable) (recoveryPlan, error) {
	profile := hidOnlyProfile()
	plan, err := Compile(profile, caps)
	if err != nil {
		return recoveryPlan{}, fmt.Errorf("compile %s: %w", profile.Name, err)
	}
	return recoveryPlan{profile: profile, plan: plan}, nil
}

func (m *Manager) restore(failed Profile, recovery recoveryPlan, udc string) (err error) {
	defer func() {
		if err == nil {
			return
		}
		if bindErr := m.ensureBound(udc); bindErr != nil {
			err = errors.Join(err, fmt.Errorf("restore binding: %w", bindErr))
		}
	}()

	probes := append(append([]Function(nil), failed.Functions...), recovery.profile.Functions...)
	before := readSnapshot(m.ops, probes)
	recovery.plan = m.dropRedundantWrites(before, recovery.plan)
	if err := m.unbindIfBound(); err != nil {
		return fmt.Errorf("unbind: %w", err)
	}
	for _, function := range failed.Functions {
		if function.Kind == FunctionFFS {
			if err := m.ops.Remove(configPrefix + "/ffs.hybrid"); err != nil {
				return err
			}
			break
		}
	}
	if err := m.unlinkStale(before, recovery.plan); err != nil {
		return err
	}
	for i, op := range recovery.plan.Ops {
		if err := m.execute(op, udc); err != nil {
			return fmt.Errorf("op %d %s %s: %w", i, op.Kind, op.Path, err)
		}
	}
	if err := m.verify(udc, recovery.profile); err != nil {
		return err
	}
	if err := m.store.SaveProfile(recovery.profile); err != nil {
		return err
	}
	if err := mirrorSentinels(recovery.profile); err != nil {
		return err
	}
	if err := m.store.SetActive(recovery.profile.Name); err != nil {
		return err
	}
	return m.store.SetLastKnownGood(recovery.profile.Name)
}

// An empty UDC write is unregister_gadget, which is ENODEV unless the gadget is
// bound right now, so it is an unbind and never a no-op. Every rollback rung
// reaches this with the transaction's own unbind already done, and a device
// only escapes it because S03usbdev binds before the server starts.
func (m *Manager) unbindIfBound() error {
	data, err := m.ops.ReadFile(udcAttr)
	if err == nil && strings.TrimSpace(string(data)) == "" {
		return nil
	}
	return m.ops.UnbindUDC()
}

// f_hid, f_ncm, f_rndis, f_uvc and f_uac2 all take opts->refcnt when the
// function is linked into a config and return -EBUSY from every option store
// while they hold it, and R1.1 forbids unlinking hid.* to release it.
// S03usbdev builds and links hid.GS0-GS2 at boot from the same values the
// built-in profiles carry, and unlinkStale keeps a link the incoming plan also
// carries, so the first apply after boot reissues writes the attribute already
// holds. The stores are guarded but the shows are not, so a write whose
// attribute reads back as the bytes the plan wants is dropped. One that differs
// is left in the plan for the kernel to answer.
func (m *Manager) dropRedundantWrites(before Snapshot, plan Plan) Plan {
	linked := make(map[string]bool, len(before.Linked))
	for _, name := range before.Linked {
		linked[name] = true
	}

	ops := make([]Op, 0, len(plan.Ops))
	for _, op := range plan.Ops {
		if op.Kind == OpWrite && linked[writtenFunction(op.Path)] {
			if current, err := m.ops.ReadFile(op.Path); err == nil && bytes.Equal(current, op.Data) {
				continue
			}
		}
		ops = append(ops, op)
	}
	plan.Ops = ops
	return plan
}

func writtenFunction(path string) string {
	rest, ok := strings.CutPrefix(path, functionsDir+"/")
	if !ok {
		return ""
	}
	name, _, ok := strings.Cut(rest, "/")
	if !ok {
		return ""
	}
	return name
}

func (m *Manager) ensureBound(udc string) error {
	data, readErr := m.ops.ReadFile(udcAttr)
	bound := strings.TrimSpace(string(data))
	if readErr == nil && bound == udc {
		return nil
	}
	if readErr == nil && bound != "" {
		if err := m.ops.UnbindUDC(); err != nil {
			return err
		}
	}
	if err := m.ops.BindUDC(udc); err != nil {
		if readErr != nil {
			return errors.Join(readErr, err)
		}
		return err
	}
	if err := m.ops.SetOTGRole(OTGRoleDevice); err != nil {
		return err
	}
	return m.verifyBind(udc)
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
	case OpRmdir:
		return m.ops.RemoveDir(op.Path)
	case OpBind:
		return m.ops.BindUDC(udc)
	case OpOTGRole:
		return m.ops.SetOTGRole(string(op.Data))
	default:
		return fmt.Errorf("unknown op kind %s", op.Kind)
	}
}

// Hybrid unlinks GS2 but retains its function directory, which keeps its minor
// reserved for rollback. Persistent profile changes keep the original rule.
func (m *Manager) unlinkStale(before Snapshot, plan Plan) error {
	for _, name := range plan.Outcome(before).Removes {
		if err := m.ops.Remove(configPrefix + "/" + name); err != nil {
			return fmt.Errorf("unlink %s: %w", name, err)
		}
	}
	return nil
}

// H4 proves the controller took the gadget and says nothing about /dev/hidgN,
// which f_hid creates when its function binds and destroys when it unbinds. A
// bind that leaves no HID nodes behind is indistinguishable from a healthy one
// through the UDC attribute alone, so the transaction opens them before it
// commits and hands the failure to the rollback ladder rather than to an
// operator with no keyboard. They are closed again because the rest of the
// transaction still runs quiesced; withHIDQuiesced reopens them for good on the
// way out and releases every key there.
func (m *Manager) verify(udc string, profile Profile) error {
	if err := m.verifyBind(udc); err != nil {
		return err
	}

	h := m.quiescer()
	if h == nil || !linksEveryHID(profile) {
		return nil
	}
	if err := h.OpenNoLockWithRetry(hidQuiesceTimeout, hidQuiesceRetryDelay); err != nil {
		return fmt.Errorf("hid devices did not come back: %w", err)
	}
	h.CloseNoLock()
	return nil
}

// The devices open as a set, so the probe means something only for a profile
// that links all three. The hybrid profile drops GS2 on purpose, and demanding
// a node its own plan removed would turn every transient start into a rollback.
func linksEveryHID(profile Profile) bool {
	linked := 0
	for _, function := range profile.Functions {
		if function.Kind == FunctionHID {
			linked++
		}
	}
	return linked == len(hidInstances)
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
// runtime state is deliberately not part of the profile (H7). The exception is
// an inquiry_string the active profile set for itself, which a mount would
// otherwise throw away every time the disk changed.
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
	inquiry := lun.inquiry()
	if chosen := m.store.ActiveInquiry(); chosen != "" {
		inquiry = chosen
	}
	if err := m.ops.WriteFile(lunAttr("inquiry_string"), []byte(inquiry)); err != nil {
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
