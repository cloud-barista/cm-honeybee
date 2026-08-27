package server

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/cloud-barista/cm-honeybee/agent/common"
	"github.com/cloud-barista/cm-honeybee/agent/lib/config"
	_ "github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/docs" // Honeybee Documentation
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/middlewares"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/route"
	"github.com/jollaman999/utils/logger"
	"github.com/labstack/echo/v4"
)

const (
	infoColor   = "\033[1;34m%s\033[0m"
	noticeColor = "\033[1;36m%s\033[0m"
)

const (
	website = " https://github.com/cloud-barista/cm-honeybee"
)

// listenAddress is loopback only. cm-honeybee never reaches the agent over the
// network: it opens an SSH session to the source host and runs curl against
// localhost there. Binding every interface would expose an unauthenticated API
// running as root for no gain.
const listenAddress = "127.0.0.1"

// @title CM-Honeybee Agent REST API
// @version latest
// @description Collecting and Aggregating agent module

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @BasePath /honeybee-agent

func Init() {
	e := echo.New()

	e.Use(middlewares.CustomLogger())

	// Hide Echo Banner
	e.HideBanner = true

	route.RegisterInfra(e)
	route.RegisterData(e)
	route.RegisterSoftware(e)
	route.RegisterKubernetes(e)
	route.RegisterHelm(e)
	route.RegisterSwagger(e)
	route.RegisterUtility(e)

	// Bind before anything reports the port: with listen.port 0 the kernel picks
	// a free port, so the number only exists once the listener does.
	listener, err := net.Listen("tcp", listenAddress+":"+
		config.CMHoneybeeAgentConfig.CMHoneybeeAgent.Listen.Port)
	if err != nil {
		logger.Panicln(logger.ERROR, true, err)
	}
	port := listener.Addr().(*net.TCPAddr).Port

	// Publish the port before serving so a caller on this host can find the API
	// without being told which port it landed on.
	if err := common.WriteListenPort(port); err != nil {
		logger.Println(logger.ERROR, true, "Failed to write the listen port file: "+err.Error())
	}

	// Display API Docs Dashboard when server starts
	endpoint := listenAddress + ":" + strconv.Itoa(port)
	apiDocsDashboard := " http://" + endpoint + "/" + strings.ToLower(common.ShortModuleName) + "/api/index.html"

	fmt.Println("\n ")
	fmt.Println(" CM-Honeybee repository:")
	fmt.Printf(infoColor, website)
	fmt.Println("\n ")
	fmt.Println(" API Docs Dashboard:")
	fmt.Printf(noticeColor, apiDocsDashboard)
	fmt.Println("\n ")

	e.Listener = listener
	logger.Panicln(logger.ERROR, true, e.StartServer(e.Server))
}
