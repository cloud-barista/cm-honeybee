package route

import (
	"github.com/cloud-barista/cm-honeybee/agent/common"
	"github.com/labstack/echo/v4"
	echoSwagger "github.com/swaggo/echo-swagger"
	"strings"
)

func RegisterSwagger(e *echo.Echo) {
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/swagger/*", echoSwagger.WrapHandler)
}
