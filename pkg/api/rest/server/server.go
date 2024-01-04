package server

import (
	"github.com/cloud-barista/cm-honeybee/lib/config"
	_ "github.com/cloud-barista/cm-honeybee/pkg/api/rest/docs" // Honeybee Documentation
	"github.com/cloud-barista/cm-honeybee/pkg/api/rest/middlewares"
	"github.com/cloud-barista/cm-honeybee/pkg/api/rest/route"
	"github.com/jollaman999/utils/logger"
	"github.com/labstack/echo/v4"
)

func Init() {
	e := echo.New()

	e.Use(middlewares.CustomLogger())

	route.RegisterInfra(e)
	route.RegisterSoftware(e)
	route.RegisterSwagger(e)
	route.RegisterUtility(e)

	err := e.Start(":" + config.CMHoneybeeConfig.CMHoneybee.Listen.Port)
	logger.Panicln(logger.ERROR, true, err)
}
