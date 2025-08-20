package main

import (
	_ "embed"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/cloud-barista/cm-honeybee/agent/common"
	"github.com/cloud-barista/cm-honeybee/agent/lib/config"
	"github.com/cloud-barista/cm-honeybee/agent/lib/privileged"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/controller"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/server"
	"github.com/jollaman999/utils/fileutil"
	"github.com/jollaman999/utils/logger"
	"github.com/jollaman999/utils/syscheck"
)

var version = "v0.3.3"

func init() {
	argsWithoutProg := os.Args[1:]
	if len(argsWithoutProg) > 0 {
		if argsWithoutProg[0] == "version" {
			fmt.Println(version)
			return
		}
	}

	err := syscheck.CheckRoot()
	if err != nil {
		log.Fatalln(err)
	}

	err = privileged.CheckPrivileged()
	if err != nil {
		log.Fatalln(err)
	}

	common.RootPath = os.Getenv(common.ModuleROOT)
	if len(common.RootPath) == 0 {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatalln(err)
		}

		common.RootPath = homeDir + "/." + strings.ToLower(common.ModuleName)
	}

	err = fileutil.CreateDirIfNotExist(common.RootPath)
	if err != nil {
		log.Fatalln(err)
	}

	err = logger.InitLogFile(common.RootPath+"/log", strings.ToLower(common.ModuleName))
	if err != nil {
		log.Panicln(err)
	}

	err = config.PrepareConfigs()
	if err != nil {
		logger.Println(logger.ERROR, false, err.Error())
	}

	err = common.InitAgentUUID()
	if err != nil {
		logger.Println(logger.ERROR, false, err.Error())
	}

	logger.Println(logger.INFO, false, "Agent UUID: "+common.AgentUUID)

	controller.OkMessage.Message = "API server is not ready"

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer func() {
			wg.Done()
		}()
		server.Init()
	}()

	controller.OkMessage.Message = "CM-Honeybee Agent is ready"
	controller.IsReady = true

	wg.Wait()
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
