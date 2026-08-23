package tunnel

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"NanoKVM-Server/proto"
	"NanoKVM-Server/utils"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// A tunnel's limit is its own file. /etc/kvm/GOMEMLIMIT is read by tailscaled
// and applied to the NanoKVM server's own runtime, so a value written there for
// one service lands on all three.
func memLimitPath(name proto.TunnelName) string {
	return filepath.Join(configDir, string(name)+".GOMEMLIMIT")
}

func memLimit(name proto.TunnelName) (int64, bool) {
	if spec, ok := specOf(name); !ok || spec.MemLimit == 0 {
		return 0, false
	}

	data, err := os.ReadFile(memLimitPath(name))
	if err != nil {
		return 0, false
	}

	limit, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil || limit <= 0 {
		return 0, false
	}
	return limit, true
}

func setMemLimit(name proto.TunnelName, limit int64) error {
	data := []byte(strconv.FormatInt(limit, 10) + "\n")
	return utils.WriteFileAtomic(memLimitPath(name), data, 0o644)
}

func delMemLimit(name proto.TunnelName) error {
	if err := os.Remove(memLimitPath(name)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (s *Service) GetMemoryLimit(c *gin.Context) {
	var rsp proto.Response

	spec, _ := specOf(s.name)
	limit, enabled := memLimit(s.name)
	if !enabled {
		limit = spec.MemLimit
	}

	rsp.OkRspWithData(c, &proto.GetTunnelMemoryRsp{
		Supported: spec.MemLimit > 0,
		Enabled:   enabled,
		Limit:     limit,
	})

	log.Debugf("get %s memory limit successfully", s.name)
}

func (s *Service) SetMemoryLimit(c *gin.Context) {
	var rsp proto.Response

	var req proto.SetTunnelMemoryReq
	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	spec, ok := specOf(s.name)
	if !ok || spec.MemLimit == 0 {
		rsp.ErrRsp(c, -1, "memory limit not supported")
		return
	}

	var err error
	if req.Enabled {
		err = setMemLimit(s.name, spec.MemLimit)
	} else {
		err = delMemLimit(s.name)
	}
	if err != nil {
		rsp.ErrRsp(c, -2, "set memory limit failed")
		log.Errorf("failed to set %s memory limit: %s", s.name, err)
		return
	}

	if isInstalled(s.name) {
		cfg, cfgErr := loadConfig(s.name)
		if cfgErr == nil {
			cfgErr = writeWrapper(s.name, cfg)
		}
		if cfgErr != nil {
			rsp.ErrRsp(c, -3, "set memory limit failed")
			log.Errorf("failed to rewrite %s wrapper: %s", s.name, cfgErr)
			return
		}
	}

	rsp.OkRsp(c)
	log.Debugf("set %s memory limit enabled: %t", s.name, req.Enabled)
}
