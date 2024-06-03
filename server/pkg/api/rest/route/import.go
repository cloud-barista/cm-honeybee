package route

import (
	"github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/controller"
	"github.com/labstack/echo/v4"
	"strings"
)

func RegisterImport(e *echo.Echo) {
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/import/infra/:uuid", controller.ImportInfra)
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/import/software/:uuid", controller.ImportSoftware)
}
