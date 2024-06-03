package route

import (
	"github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/controller"
	"github.com/labstack/echo/v4"
	"strings"
)

func RegisterConnectionInfo(e *echo.Echo) {
	e.POST("/"+strings.ToLower(common.ShortModuleName)+"/connection_info", controller.CreateConnectionInfo)
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/connection_info/:uuid", controller.GetConnectionInfo)
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/connection_info", controller.ListConnectionInfo)
	e.PUT("/"+strings.ToLower(common.ShortModuleName)+"/connection_info/:uuid", controller.UpdateConnectionInfo)
	e.DELETE("/"+strings.ToLower(common.ShortModuleName)+"/connection_info/:uuid", controller.DeleteConnectionInfo)
}
