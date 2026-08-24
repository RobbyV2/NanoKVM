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
// point. Until the state flip in step 3 the bootloader still picks boot.sd, so
// a half-written boot.alt is never selectable; after it, bootcnt is already
// zero and the trial gets its single attempt. Reversing any pair of these is
// what bricks the device.
func installKernelPayload(kernel *kernelPayload, slot bootslot.Paths) error {
	switch slot.Slot() {
	case "":
		return errNoSlotPolicy
	case bootslot.SlotTrial:
		return errors.New("a kernel trial is still unconfirmed; reboot before installing another")
	}

	if err := stageKernel(slot.Alt(), kernel.itb); err != nil {
		return err
	}
	if err := slot.ResetBootCount(); err != nil {
		return fmt.Errorf("reset boot counter: %w", err)
	}
	if err := slot.SetState(bootslot.StateTrial); err != nil {
		return fmt.Errorf("arm trial slot: %w", err)
	}
	if err := slot.MarkPending(kernel.version); err != nil {
		return fmt.Errorf("record pending kernel: %w", err)
	}
	return nil
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
