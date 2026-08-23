package application

import (
	"NanoKVM-Server/proto"
	"NanoKVM-Server/service/startup"

	"github.com/gin-gonic/gin"
)

// A subsystem that failed to initialise must say so rather than look healthy:
// the listener now comes up regardless of what the hardware did, so this is the
// only place the difference is visible.
func (s *Service) GetStartup(c *gin.Context) {
	var rsp proto.Response

	report := startup.Report()
	steps := make([]proto.StartupStatus, 0, len(report))
	for _, status := range report {
		steps = append(steps, proto.StartupStatus{Name: status.Name, Error: status.Error})
	}

	rsp.OkRspWithData(c, &proto.GetStartupRsp{Steps: steps})
}
