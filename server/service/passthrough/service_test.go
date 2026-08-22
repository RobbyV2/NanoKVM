package passthrough

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
