package route

import (
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/controller"
	"github.com/labstack/echo/v4"
)

func RegisterImport(e *echo.Echo) {
	e.GET("/import/infra/:uuid", controller.ImportInfra)
	e.GET("/import/software/:uuid", controller.ImportSoftware)
}
