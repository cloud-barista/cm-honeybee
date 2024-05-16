package route

import (
	"github.com/cloud-barista/cm-honeybee/pkg/api/rest/controller"
	"github.com/labstack/echo/v4"
)

func RegisterMigrationGroup(e *echo.Echo) {
	e.POST("/migration_group", controller.CreateMigrationGroup)
	e.GET("/migration_group/:uuid", controller.GetMigrationGroup)
	e.GET("/migration_group", controller.ListMigrationGroup)
	e.PUT("/migration_group/:uuid", controller.UpdateMigrationGroup)
	e.DELETE("/migration_group/:uuid", controller.DeleteMigrationGroup)
	e.GET("/migration_group/check/:uuid", controller.CheckConnectionMigrationGroup)
}
