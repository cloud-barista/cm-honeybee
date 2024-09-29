package main

import (
	"errors"
	"github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/cloud-barista/cm-honeybee/server/db"
	"github.com/cloud-barista/cm-honeybee/server/lib/config"
	"github.com/cloud-barista/cm-honeybee/server/lib/rsautil"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/controller"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/server"
	"github.com/jollaman999/utils/fileutil"
	"github.com/jollaman999/utils/logger"
	"github.com/jollaman999/utils/syscheck"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
)

func init() {
	err := syscheck.CheckRoot()
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

	controller.OkMessage.Message = "API server is not ready"

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer func() {
			wg.Done()
		}()
		server.Init()
	}()

	controller.OkMessage.Message = "Database is not ready"
	err = db.Open()
	if err != nil {
		logger.Panicln(logger.ERROR, true, err.Error())
	}

	privateKeyPath := common.RootPath + "/" + common.PrivateKeyFileName
	publicKeyPath := common.RootPath + "/" + common.PublicKeyFileName

	controller.OkMessage.Message = "RSA public key is not ready"
	if !fileutil.IsExist(privateKeyPath) && !fileutil.IsExist(publicKeyPath) {
		err := rsautil.GeneratePrivateKeyAndPublicKey(4096, privateKeyPath, publicKeyPath)
		if err != nil {
			logger.Panicln(logger.ERROR, true, err.Error())
		}
	} else if !fileutil.IsExist(publicKeyPath) {
		logger.Panicln(logger.ERROR, true, errors.New("public key not found ("+publicKeyPath+")"))
	}

	common.PubKey, err = rsautil.ReadPublicKey(publicKeyPath)
	if err != nil {
		logger.Panicln(logger.ERROR, true, "error occurred while reading public key")
	}

	controller.OkMessage.Message = "CM-Honeybee API server is ready"
	controller.IsReady = true

	wg.Wait()
}

func end() {
	db.Close()
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
