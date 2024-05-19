package route

import (
	"github.com/cloud-barista/cm-honeybee/pkg/api/rest/controller"
	"github.com/labstack/echo/v4"
)

func RegisterSoftware(e *echo.Echo) {
	e.GET("/software/:uuid", controller.GetSoftwareInfo)
}
