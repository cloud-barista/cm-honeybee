package route

import (
	"strings"

	"github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/controller"
	"github.com/labstack/echo/v4"
)

func RegisterCSP(e *echo.Echo) {
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/csp", controller.ListCSP)
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/csp/:name", controller.GetCSP)
}
