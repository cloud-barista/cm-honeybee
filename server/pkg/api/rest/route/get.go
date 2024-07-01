package route

import (
	"github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/controller"
	"github.com/labstack/echo/v4"
	"strings"
)

func RegisterGet(e *echo.Echo) {
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/source_group/:sgId/connection_info/:connId/infra", controller.GetInfraInfo)
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/source_group/:sgId/infra", controller.GetInfraInfoSourceGroup)
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/source_group/:sgId/connection_info/:connId/software", controller.GetSoftwareInfo)
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/source_group/:sgId/software", controller.GetSoftwareInfoSourceGroup)
}
