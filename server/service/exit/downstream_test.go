package exit

import (
	"context"
	"errors"
	"testing"
)

func TestParseStatus(t *testing.T) {
	report, ok := parseStatus("forward=1\nrouting=1\ntun=0\nhev=1\nwstunnel=1\nnat=0\nnic=usb0\ngw=10.1.2.1\ndns_redirected=17\n")
	if !ok {
		t.Fatal("status not recognised")
	}
	if !report.Down.Forward || !report.Down.Routing || report.Down.Tun || !report.Down.Hev || !report.Down.Wstunnel || report.Down.NAT {
		t.Fatalf("vector = %+v", report.Down)
	}
	if report.Down.DNS {
		t.Fatal("dns is the forwarder's to report, not the script's")
	}
	if report.NIC != "usb0" || report.Gateway != "10.1.2.1" || report.DNSRedirected != 17 {
		t.Fatalf("extras = %+v", report)
	}

	if _, ok := parseStatus("S94exit: invalid slot 'x'\n"); ok {
		t.Fatal("garbage recognised as a status vector")
	}
}

type scriptedRunner struct {
	calls []string
	out   map[string]string
	err   map[string]error
}

func (r *scriptedRunner) Run(_ context.Context, name string, args ...string) (string, error) {
	key := name
	for _, a := range args {
		key += " " + a
	}
	r.calls = append(r.calls, key)
	return r.out[key], r.err[key]
}

func TestDownstreamStatusFoldsTheExitCode(t *testing.T) {
	root := useTempDirs(t)
	writeExec(t, InitSeedDir+"/"+S94Script, "#!/bin/sh\n")
	writeExec(t, InitSeedDir+"/"+S30Script, "#!/bin/sh\n")
	_ = root

	key := InitSeedDir + "/S94exit status 0"
	r := &scriptedRunner{
		out: map[string]string{key: "forward=1\nrouting=0\ntun=0\nhev=0\nwstunnel=1\nnat=0\nnic=\ngw=\ndns_redirected=0"},
		err: map[string]error{key: errors.New("exit status 1")},
	}
	d := downstream{run: r}
	report, err := d.Status(context.Background(), MustSlot("0"))
	if err != nil {
		t.Fatalf("a non-zero status with a vector is a report, got %v", err)
	}
	if !report.Down.Forward || report.Down.Hev {
		t.Fatalf("vector = %+v", report.Down)
	}

	r.out[key] = ""
	if _, err := d.Status(context.Background(), MustSlot("0")); !errors.Is(err, ErrStatusUnparsed) {
		t.Fatalf("empty output: %v", err)
	}

	if err := d.Start(context.Background(), MustSlot("0")); err != nil {
		t.Fatal(err)
	}
	if err := d.RestartRNDIS(context.Background()); err != nil {
		t.Fatal(err)
	}
	if r.calls[len(r.calls)-2] != InitSeedDir+"/S94exit start 0" || r.calls[len(r.calls)-1] != InitSeedDir+"/S30rndis restart" {
		t.Fatalf("calls = %v", r.calls)
	}
}
