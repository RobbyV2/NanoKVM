package passthrough

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
)

func requestFrom(ip string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/api/vm/passthrough/start", nil)
	local := &net.TCPAddr{IP: net.ParseIP(ip), Port: 80}
	return req.WithContext(context.WithValue(req.Context(), http.LocalAddrContextKey, local))
}

func TestManagementPathRejectsTheGadgetAndBridge(t *testing.T) {
	previous := managementInterfaces
	managementInterfaces = func() ([]localInterface, error) {
		return []localInterface{
			{name: "eth0", addrs: []net.Addr{&net.IPNet{IP: net.ParseIP("192.168.1.8"), Mask: net.CIDRMask(24, 32)}}},
			{name: "usb0", addrs: []net.Addr{&net.IPNet{IP: net.ParseIP("192.168.90.1"), Mask: net.CIDRMask(22, 32)}}},
			{name: "br0", addrs: []net.Addr{&net.IPNet{IP: net.ParseIP("10.0.0.8"), Mask: net.CIDRMask(24, 32)}}},
		}, nil
	}
	t.Cleanup(func() { managementInterfaces = previous })

	for _, ip := range []string{"192.168.90.1", "10.0.0.8"} {
		if err := validateManagementPath(requestFrom(ip)); !errors.Is(err, ErrGadgetManagement) {
			t.Fatalf("validate %s = %v, want %v", ip, err, ErrGadgetManagement)
		}
	}
	if err := validateManagementPath(requestFrom("192.168.1.8")); err != nil {
		t.Fatalf("validate Ethernet: %v", err)
	}
}

func TestStartHandlerChecksManagementBeforeTheManager(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &Service{
		guard: func(*http.Request) error { return ErrGadgetManagement },
	}
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/vm/passthrough/start",
		strings.NewReader(`{"Exporter":"10.0.0.5","BusID":"1-1"}`))
	context.Request.Header.Set("Content-Type", "application/json")

	service.StartPassthrough(context)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), ErrGadgetManagement.Error()) {
		t.Fatalf("response = %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestListHandlerNamesTheDevicesTheBackendWillRefuse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	keyboard := RemoteDevice{
		Device:     Device{BusID: "1-1", IDVendor: 0x046d, IDProduct: 0xc31c, Speed: SpeedLow, DeviceClass: 0x00},
		Interfaces: []Interface{{Class: 0x03}},
	}
	headset := RemoteDevice{
		Device:     Device{BusID: "1-2", IDVendor: 0x046d, IDProduct: 0x0a38, Speed: SpeedFull},
		Interfaces: []Interface{{Class: 0x01, SubClass: 0x02}},
	}
	service := &Service{list: func(context.Context, string) ([]RemoteDevice, error) {
		return []RemoteDevice{keyboard, headset}, nil
	}}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/vm/passthrough/devices",
		strings.NewReader(`{"Exporter":"10.0.0.5"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	service.ListPassthroughDevices(ctx)

	var body struct {
		Code int                      `json:"code"`
		Data proto.ListPassthroughRsp `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode %s: %v", recorder.Body.String(), err)
	}
	if body.Code != 0 || len(body.Data.Devices) != 2 {
		t.Fatalf("response = %s", recorder.Body.String())
	}
	if body.Data.Devices[0].BusID != "1-1" || body.Data.Devices[0].IDProduct != "c31c" || body.Data.Devices[0].Unsupported != "" {
		t.Fatalf("keyboard = %+v, want it offered", body.Data.Devices[0])
	}
	if !strings.Contains(body.Data.Devices[1].Unsupported, "isochronous") {
		t.Fatalf("headset = %+v, want the refusal", body.Data.Devices[1])
	}
}
