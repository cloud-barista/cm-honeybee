package route

import (
	"github.com/cloud-barista/cm-honeybee/pkg/api/rest/controller"
	"github.com/labstack/echo/v4"
)

func RegisterMigrationGroup(e *echo.Echo) {
	e.POST("/migration_group", controller.MigrationGroupRegister)
	e.GET("/migration_group/:uuid", controller.MigrationGroupGet)
	e.GET("/migration_group", controller.MigrationGroupGetList)
	e.PUT("/migration_group/:uuid", controller.MigrationGroupUpdate)
	e.DELETE("/migration_group/:uuid", controller.MigrationGroupDelete)
	e.GET("/migration_group/check/:uuid", controller.MigrationGroupCheckConnection)
}
