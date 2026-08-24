package application

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"NanoKVM-Server/proto"
	"NanoKVM-Server/service/bootslot"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

const kernelSubtree = "kernel"

var fdtMagic = []byte{0xd0, 0x0d, 0xfe, 0xed}

var errNoSlotPolicy = errors.New("this bootloader has no A/B slot policy, so a kernel cannot be installed safely")

func (s *Service) GetKernel(c *gin.Context) {
	var rsp proto.Response

	slot := bootslot.Default()
	rolledBack, _ := slot.RolledBack()
	rsp.OkRspWithData(c, &proto.GetKernelRsp{
		Slot:       slot.Slot(),
		Installed:  slot.InstalledVersion(),
		RolledBack: rolledBack,
	})
}

func (s *Service) DismissRollback(c *gin.Context) {
	var rsp proto.Response

	if err := bootslot.Default().ClearRollback(); err != nil {
		rsp.ErrRsp(c, -1, "failed to dismiss rollback")
		return
	}
	rsp.OkRsp(c)
}

// installKernelPayload runs the one ordering that survives losing power at any
// point. Until the state flip the bootloader still picks boot.sd, so a
// half-written boot.alt is never selectable; after it, bootcnt is already zero
// and the trial gets its single attempt. Reversing any pair of these is what
// bricks the device. The leading disarm matters because a rolled back device,
// and one whose reboot never happened, still carries ab_state=trial.
func installKernelPayload(kernel *kernelPayload, slot bootslot.Paths) error {
	switch slot.Slot() {
	case "":
		return errNoSlotPolicy
	case bootslot.SlotTrial:
		return errors.New("a kernel trial is still unconfirmed; reboot before installing another")
	}

	for _, step := range kernelInstallSteps(kernel, slot) {
		if err := step.run(); err != nil {
			return fmt.Errorf("%s: %w", step.name, err)
		}
	}
	return nil
}

type kernelStep struct {
	name string
	run  func() error
}

func kernelInstallSteps(kernel *kernelPayload, slot bootslot.Paths) []kernelStep {
	return []kernelStep{
		{"disarm trial slot", func() error { return slot.SetState(bootslot.StateCommitted) }},
		{"write trial kernel", func() error { return stageKernel(slot.Alt(), kernel.itb) }},
		{"reset boot counter", slot.ResetBootCount},
		{"arm trial slot", func() error { return slot.SetState(bootslot.StateTrial) }},
		{"record pending kernel", func() error { return slot.MarkPending(kernel.version) }},
	}
}

// stageKernel writes straight over the trial slot. /boot holds under 2 MiB
// free against a 7 MiB kernel, so a temporary file beside it cannot exist; the
// package staged in CacheDir on partition 2 is the copy a torn write falls
// back to, and the read-back is what proves the write landed.
func stageKernel(dst, src string) error {
	want, err := digest(src)
	if err != nil {
		return fmt.Errorf("read staged kernel: %w", err)
	}
	if err := writeKernel(dst, src); err != nil {
		return fmt.Errorf("write trial kernel: %w", err)
	}
	return verifyStaged(dst, want)
}

func verifyStaged(dst string, want []byte) error {
	got, err := digest(dst)
	if err != nil {
		return fmt.Errorf("re-read trial kernel: %w", err)
	}
	if !bytes.Equal(want, got) {
		return errors.New("trial kernel does not match the package after writing")
	}
	return nil
}

func writeKernel(dst, src string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	if _, err = io.Copy(out, in); err == nil {
		err = out.Sync()
	}
	if closeErr := out.Close(); err == nil {
		err = closeErr
	}
	return err
}

func digest(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return nil, err
	}
	return hasher.Sum(nil), nil
}

func rebootForKernel() {
	time.Sleep(1 * time.Second)

	_ = exec.Command("sync").Run()
	if err := exec.Command("reboot").Run(); err != nil {
		log.Errorf("failed to reboot into the trial kernel: %v", err)
	}
}
