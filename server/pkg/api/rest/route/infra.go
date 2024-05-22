package route

import (
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/controller"
	"github.com/labstack/echo/v4"
)

func RegisterInfra(e *echo.Echo) {
	e.GET("/infra/:uuid", controller.GetInfraInfo)
}
