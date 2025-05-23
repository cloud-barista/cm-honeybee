package route

import (
	"github.com/cloud-barista/cm-honeybee/agent/common"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/controller"
	_ "github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/docs" // Honeybee Documentation
	"github.com/labstack/echo/v4"
	"strings"
)

func RegisterHelm(e *echo.Echo) {
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/helm", controller.GetHelmInfo)
}
