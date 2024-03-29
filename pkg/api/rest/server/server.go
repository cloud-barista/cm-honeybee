package server

import (
	"fmt"
	"strings"

	"github.com/cloud-barista/cm-honeybee/common"
	"github.com/cloud-barista/cm-honeybee/lib/config"
	_ "github.com/cloud-barista/cm-honeybee/pkg/api/rest/docs" // Honeybee Documentation
	"github.com/cloud-barista/cm-honeybee/pkg/api/rest/middlewares"
	"github.com/cloud-barista/cm-honeybee/pkg/api/rest/route"
	"github.com/jollaman999/utils/logger"
	"github.com/labstack/echo/v4"
)

const (
	infoColor    = "\033[1;34m%s\033[0m"
	noticeColor  = "\033[1;36m%s\033[0m"
	warningColor = "\033[1;33m%s\033[0m"
	errorColor   = "\033[1;31m%s\033[0m"
	debugColor   = "\033[0;36m%s\033[0m"
)

const (
	website = " https://github.com/cloud-barista/cm-honeybee"
)

func Init() {
	e := echo.New()

	e.Use(middlewares.CustomLogger())

	// Hide Echo Banner
	e.HideBanner = true

	route.RegisterInfra(e)
	route.RegisterSoftware(e)
	route.RegisterSwagger(e)
	route.RegisterUtility(e)

	// Display API Docs Dashboard when server starts
	endpoint := "localhost:" + config.CMHoneybeeConfig.CMHoneybee.Listen.Port
	apiDocsDashboard := " http://" + endpoint + "/" + strings.ToLower(common.ShortModuleName) + "/swagger/index.html"

	fmt.Println("\n ")
	fmt.Println(" CM-Honeybee repository:")
	fmt.Printf(infoColor, website)
	fmt.Println("\n ")
	fmt.Println(" API Docs Dashboard:")
	fmt.Printf(noticeColor, apiDocsDashboard)
	fmt.Println("\n ")

	err := e.Start(":" + config.CMHoneybeeConfig.CMHoneybee.Listen.Port)
	logger.Panicln(logger.ERROR, true, err)
}
