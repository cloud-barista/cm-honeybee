package route

import (
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/controller"
	"github.com/labstack/echo/v4"
)

func RegisterSourceGroup(e *echo.Echo) {
	e.POST("/source_group", controller.CreateSourceGroup)
	e.GET("/source_group/:uuid", controller.GetSourceGroup)
	e.GET("/source_group", controller.ListSourceGroup)
	e.PUT("/source_group/:uuid", controller.UpdateSourceGroup)
	e.DELETE("/source_group/:uuid", controller.DeleteSourceGroup)
	e.GET("/source_group/check/:uuid", controller.CheckConnectionSourceGroup)
}
