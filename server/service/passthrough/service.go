package passthrough

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"

	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

type Service struct {
	manager *Manager
	guard   func(*http.Request) error
}

func NewService() *Service {
	return &Service{manager: GetManager(), guard: validateManagementPath}
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

	if _, err := s.manager.Start(c.Request.Context(), req.Exporter, req.BusID); err != nil {
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

	if err := s.manager.Stop(); err != nil {
		log.Errorf("passthrough: stop failed: %s", err)
		rsp.ErrRsp(c, -1, err.Error())
		return
	}

	rsp.OkRsp(c)
}
