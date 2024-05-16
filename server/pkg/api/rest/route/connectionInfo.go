package route

import (
	"github.com/cloud-barista/cm-honeybee/pkg/api/rest/controller"
	"github.com/labstack/echo/v4"
)

func RegisterConnectionInfo(e *echo.Echo) {
	e.POST("/connection_info", controller.CreateConnectionInfo)
	e.GET("/connection_info/:uuid", controller.GetConnectionInfo)
	e.GET("/connection_info", controller.ListConnectionInfo)
	e.PUT("/connection_info/:uuid", controller.UpdateConnectionInfo)
	e.DELETE("/connection_info/:uuid", controller.DeleteConnectionInfo)
}
