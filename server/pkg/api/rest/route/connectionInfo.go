package route

import (
	"github.com/cloud-barista/cm-honeybee/pkg/api/rest/controller"
	"github.com/labstack/echo/v4"
)

func RegisterConnectionInfo(e *echo.Echo) {
	e.POST("/connection_info", controller.ConnectionInfoRegister)
	e.GET("/connection_info/:uuid", controller.ConnectionInfoGet)
	e.GET("/connection_info", controller.ConnectionInfoGetList)
	e.PUT("/connection_info/:uuid", controller.ConnectionInfoUpdate)
	e.DELETE("/connection_info/:uuid", controller.ConnectionInfoDelete)
}
