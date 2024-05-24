package route

import (
	"github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/controller"
	"strings"

	"github.com/labstack/echo/v4"
)

func RegisterUtility(e *echo.Echo) {
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/readyz", controller.CheckReady)
}
