package main

import (
	"errors"
	"github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/cloud-barista/cm-honeybee/server/db"
	"github.com/cloud-barista/cm-honeybee/server/lib/config"
	"github.com/cloud-barista/cm-honeybee/server/lib/openbao"
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
		logger.Panicln(logger.ERROR, false, err.Error())
	}

	// Wire the optional OpenBao secrets backend (no-op when not configured).
	openbao.Init()

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

	// Load the RSA keys BEFORE OpenBao: cm-honeybee encrypts/decrypts the OpenBao
	// unseal material it stores in the DB with these keys (see openbao.WaitReady),
	// and uses them for CSP credential encryption.
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

	// The private key is required to decrypt secrets stored encrypted at rest
	// (the OpenBao unseal material and CSP credentials). It is optional only for
	// SSH-only deployments without OpenBao, so a missing/unreadable key is logged
	// rather than fatal — features that need it report it on use.
	if fileutil.IsExist(privateKeyPath) {
		common.PrivKey, err = rsautil.ReadPrivateKey(privateKeyPath)
		if err != nil {
			logger.Println(logger.WARN, true, "error occurred while reading private key; encrypted-secret decryption will be unavailable: "+err.Error())
		}
	} else {
		logger.Println(logger.WARN, true, "private key not found ("+privateKeyPath+"); encrypted-secret decryption will be unavailable")
	}

	// Block until OpenBao is initialized, unsealed, has a token, and its KV engine
	// is ready. Container start order is not guaranteed on a host reboot (restart
	// policies ignore compose depends_on), so cm-honeybee drives OpenBao readiness
	// itself and stays not-ready until then rather than failing secret operations.
	controller.OkMessage.Message = "Waiting for OpenBao (init/unseal)"
	openbao.WaitReady()

	// Migrate any SSH secrets left in the DB (from pre-OpenBao rows) into OpenBao.
	// No-op when OpenBao is disabled or nothing remains.
	controller.MigratePlaintextSecretsToOpenBao()

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
