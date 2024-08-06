package route

import (
	"strings"

	"github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/controller"
	"github.com/labstack/echo/v4"
)

func RegisterBenchmark(e *echo.Echo) {
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/bench/:connId", controller.GetBenchmarkInfo)
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/run/bench/:connId", controller.RunBenchmarkInfo)
}
