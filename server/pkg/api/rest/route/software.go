package route

import (
	"github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/controller"
	"github.com/labstack/echo/v4"
	"strings"
)

func RegisterSoftware(e *echo.Echo) {
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/software/:uuid", controller.GetSoftwareInfo)
}
