package route

import (
	"github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/controller"
	"github.com/labstack/echo/v4"
	"strings"
)

func RegisterGetRefined(e *echo.Echo) {
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/source_group/:sgId/connection_info/:connId/infra/refined", controller.GetInfraInfoRefined)
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/source_group/:sgId/infra/refined", controller.GetInfraInfoSourceGroupRefined)
}
