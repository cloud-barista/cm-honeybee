package config

import (
	"errors"
	"fmt"
	"github.com/cloud-barista/cm-honeybee/agent/common"
	"github.com/jollaman999/utils/fileutil"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type cmHoneybeeAgentConfig struct {
	CMHoneybeeAgent struct {
		Listen struct {
			Port string `yaml:"port"`
		} `yaml:"listen"`
	} `yaml:"cm-honeybee-agent"`
}

var CMHoneybeeAgentConfig cmHoneybeeAgentConfig
var cmHoneybeeAgentConfigFile = "cm-honeybee-agent.yaml"

func checkCMHoneybeeAgentConfigFile() error {
	if CMHoneybeeAgentConfig.CMHoneybeeAgent.Listen.Port == "" {
		return errors.New("config error: cm-honeybee-agent.listen.port is empty")
	}
	port, err := strconv.Atoi(CMHoneybeeAgentConfig.CMHoneybeeAgent.Listen.Port)
	if err != nil || port < 1 || port > 65535 {
		return errors.New("config error: cm-honeybee-agent.listen.port has invalid value")
	}

	return nil
}

func readCMHoneybeeAgentConfigFile() error {
	common.RootPath = os.Getenv(common.ModuleROOT)
	if len(common.RootPath) == 0 {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}

		common.RootPath = homeDir + "/." + strings.ToLower(common.ModuleName)
	}

	err := fileutil.CreateDirIfNotExist(common.RootPath)
	if err != nil {
		return err
	}

	ex, err := os.Executable()
	if err != nil {
		return err
	}

	exPath := filepath.Dir(ex)
	configDir := exPath + "/conf"
	if !fileutil.IsExist(configDir) {
		configDir = common.RootPath + "/conf"
	}

	data, err := os.ReadFile(configDir + "/" + cmHoneybeeAgentConfigFile)
	if err != nil {
		return errors.New("can't find the config file (" + cmHoneybeeAgentConfigFile + ")" + fmt.Sprintln() +
			"Must be placed in '." + strings.ToLower(common.ModuleName) + "/conf' directory " +
			"under user's home directory or 'conf' directory where running the binary " +
			"or 'conf' directory where placed in the path of '" + common.ModuleROOT + "' environment variable")
	}

	err = yaml.Unmarshal(data, &CMHoneybeeAgentConfig)
	if err != nil {
		return err
	}

	err = checkCMHoneybeeAgentConfigFile()
	if err != nil {
		return err
	}

	return nil
}

func prepareCMHoneybeeAgentConfig() error {
	err := readCMHoneybeeAgentConfigFile()
	if err != nil {
		return err
	}

	return nil
}
