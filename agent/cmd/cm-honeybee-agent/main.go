package main

import (
	"github.com/cloud-barista/cm-honeybee/agent/common"
	"github.com/cloud-barista/cm-honeybee/agent/lib/config"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/server"
	"github.com/jollaman999/utils/logger"
	"github.com/jollaman999/utils/syscheck"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func init() {
	err := syscheck.CheckRoot()
	if err != nil {
		log.Fatalln(err)
	}

	err = config.PrepareConfigs()
	if err != nil {
		log.Fatalln(err)
	}

	err = common.InitAgentUUID()
	if err != nil {
		log.Panicln(err)
	}

	err = logger.InitLogFile(common.RootPath+"/log", strings.ToLower(common.ModuleName))
	if err != nil {
		log.Panicln(err)
	}

	logger.Println(logger.INFO, false, "Agent UUID: "+common.AgentUUID)

	server.Init()
}

func end() {
	logger.CloseLogFile()
}

func main() {
	// Catch the exit signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		logger.Println(logger.INFO, false, "Exiting "+common.ModuleName+" module...")
		end()
		os.Exit(0)
	}()
}
