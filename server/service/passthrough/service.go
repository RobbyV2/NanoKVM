package passthrough

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"

	"NanoKVM-Server/middleware"
	"NanoKVM-Server/proto"
	"NanoKVM-Server/service/audit"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

type Service struct {
	manager *Manager
	guard   func(*http.Request) error
	list    func(context.Context, string) ([]RemoteDevice, error)
}

func NewService() *Service {
	return &Service{manager: GetManager(), guard: validateManagementPath, list: List}
}

var ErrGadgetManagement = errors.New("passthrough: use Ethernet or Wi-Fi before starting; the USB network will disconnect")

type localInterface struct {
	name  string
	addrs []net.Addr
}

var managementInterfaces = func() ([]localInterface, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	result := make([]localInterface, 0, len(interfaces))
	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			return nil, err
		}
		result = append(result, localInterface{name: iface.Name, addrs: addrs})
	}
	return result, nil
}

func validateManagementPath(req *http.Request) error {
	local, ok := req.Context().Value(http.LocalAddrContextKey).(net.Addr)
	if !ok {
		return fmt.Errorf("%w: local address unavailable", ErrGadgetManagement)
	}
	host, _, err := net.SplitHostPort(local.String())
	if err != nil {
		return fmt.Errorf("%w: %v", ErrGadgetManagement, err)
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	if ip == nil {
		return fmt.Errorf("%w: local address is not an IP", ErrGadgetManagement)
	}
	if ip.IsLoopback() {
		return nil
	}

	interfaces, err := managementInterfaces()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrGadgetManagement, err)
	}
	for _, iface := range interfaces {
		for _, addr := range iface.addrs {
			var network *net.IPNet
			switch value := addr.(type) {
			case *net.IPNet:
				network = value
			case *net.IPAddr:
				network = &net.IPNet{IP: value.IP, Mask: net.CIDRMask(128, 128)}
			}
			if network == nil || !network.Contains(ip) {
				continue
			}
			if iface.name == "usb0" || iface.name == "br0" {
				return ErrGadgetManagement
			}
			return nil
		}
	}
	return fmt.Errorf("%w: local interface not found", ErrGadgetManagement)
}

func (s *Service) GetPassthrough(c *gin.Context) {
	var rsp proto.Response

	status := s.manager.Status()
	rsp.OkRspWithData(c, &status)
}

// The operator should not have to know a busid by hand, and a device the
// backend will refuse is named here rather than after a port is taken.
func (s *Service) ListPassthroughDevices(c *gin.Context) {
	var req proto.ListPassthroughReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	devices, err := s.list(c.Request.Context(), req.Exporter)
	if err != nil {
		log.Errorf("passthrough: list %s failed: %s", req.Exporter, err)
		rsp.ErrRsp(c, -2, err.Error())
		return
	}
	rsp.OkRspWithData(c, &proto.ListPassthroughRsp{Devices: remoteDevices(devices)})
}

func remoteDevices(devices []RemoteDevice) []proto.PassthroughRemoteDevice {
	out := make([]proto.PassthroughRemoteDevice, 0, len(devices))
	for _, device := range devices {
		out = append(out, proto.PassthroughRemoteDevice{
			BusID:       device.BusID,
			IDVendor:    hex4(device.IDVendor),
			IDProduct:   hex4(device.IDProduct),
			Speed:       device.Speed.String(),
			Class:       device.DeviceClass,
			Unsupported: device.Refusal(),
		})
	}
	return out
}

func (s *Service) StartPassthrough(c *gin.Context) {
	var req proto.StartPassthroughReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}
	if err := s.guard(c.Request); err != nil {
		log.Warnf("passthrough: unsafe management path: %s", err)
		rsp.ErrRsp(c, -2, err.Error())
		return
	}

	principal, _ := middleware.CurrentPrincipal(c)
	_, err := s.manager.StartMode(c.Request.Context(), req.Exporter, req.BusID, req.Mode, req.AllowIsochronous)
	audit.Record(principal, "passthrough.start", strings.TrimSpace(req.BusID+" "+req.Mode), err)
	if err != nil {
		log.Errorf("passthrough: start failed: %s", err)
		rsp.ErrRsp(c, -2, err.Error())
		return
	}

	status := s.manager.Status()
	rsp.OkRspWithData(c, &status)
	log.Debugf("passthrough started: %s %s", req.Exporter, req.BusID)
}

func (s *Service) StopPassthrough(c *gin.Context) {
	var rsp proto.Response

	principal, _ := middleware.CurrentPrincipal(c)
	err := s.manager.Stop()
	audit.Record(principal, "passthrough.stop", "", err)
	if err != nil {
		log.Errorf("passthrough: stop failed: %s", err)
		rsp.ErrRsp(c, -1, err.Error())
		return
	}

	rsp.OkRsp(c)
}
