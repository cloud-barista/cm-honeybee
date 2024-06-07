package route

import (
	"github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/controller"
	"github.com/labstack/echo/v4"
	"strings"
)

func RegisterBenchmark(e *echo.Echo) {
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/bench/:connId", controller.GetBenchmarkInfo)
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/run/bench/:connId", controller.RunBenchmarkInfo)
}
