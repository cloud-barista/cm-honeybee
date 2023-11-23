package config

import (
	"errors"
	"fmt"
	"github.com/cloud-barista/cm-honeybee/common"
	"github.com/jollaman999/utils/fileutil"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strings"
)

type cmHoneybeeConfig struct {
	CMHoneybeeAgent struct {
		Server struct {
			Address string `yaml:"address"`
			Timeout string `yaml:"timeout"`
		} `yaml:"server"`
		Listen struct {
			Port string `yaml:"port"`
		} `yaml:"listen"`
	} `yaml:"cm-honeybee"`
}

var CMHoneybeeConfig cmHoneybeeConfig
var cmHoneybeeConfigFile = "cm-honeybee.yaml"

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

	data, err := os.ReadFile(configDir + "/" + cmHoneybeeConfigFile)
	if err != nil {
		return errors.New("can't find the config file (" + cmHoneybeeConfigFile + ")" + fmt.Sprintln() +
			"Must be placed in 'conf' directory in user's home directory or 'conf' directory where running the binary " +
			"or 'conf' directory where placed in the path of '" + common.ModuleROOT + "' environment variable")
	}

	err = yaml.Unmarshal(data, &CMHoneybeeConfig)
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
