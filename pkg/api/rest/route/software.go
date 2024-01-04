package route

import (
	"github.com/cloud-barista/cm-honeybee/pkg/api/rest/controller"
	_ "github.com/cloud-barista/cm-honeybee/pkg/api/rest/docs" // Honeybee Documentation
	"github.com/labstack/echo/v4"
)

func RegisterSoftware(e *echo.Echo) {
	e.GET("/software", controller.GetSoftwareInfo)
}
