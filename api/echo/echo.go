package echo

import (
	"github.com/cloud-barista/cm-honeybee/lib/config"
	"github.com/jollaman999/utils/logger"
	"github.com/labstack/echo/v4"
)

var e *echo.Echo

func Init() {
	e = echo.New()

	InfraInfo()
	SoftwreInfo()

	err := e.Start(":" + config.CMHoneybeeConfig.CMHoneybeeAgent.Listen.Port)
	logger.Panicln(logger.ERROR, true, err)
}
