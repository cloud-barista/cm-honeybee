package route

import (
	"github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/labstack/echo/v4"
	echoSwagger "github.com/swaggo/echo-swagger"
	"net/http"
	"strings"
)

func RegisterSwagger(e *echo.Echo) {
	swaggerRedirect := func(c echo.Context) error {
		return c.Redirect(http.StatusMovedPermanently, "/"+strings.ToLower(common.ShortModuleName)+"/api/index.html")
	}
	e.GET("", swaggerRedirect)
	e.GET("/", swaggerRedirect)
	e.GET("/"+strings.ToLower(common.ShortModuleName), swaggerRedirect)
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/", swaggerRedirect)
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/api", swaggerRedirect)
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/api/", swaggerRedirect)
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/api/*", echoSwagger.WrapHandler)
}
