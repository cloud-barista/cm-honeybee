package route

import (
	"github.com/cloud-barista/cm-honeybee/common"
	"github.com/cloud-barista/cm-honeybee/pkg/api/rest/controller"
	"strings"

	"github.com/labstack/echo/v4"
)

func RegisterUtility(e *echo.Echo) {
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/health", controller.GetHealth)
}
