package config

import (
	"errors"
	"fmt"
	"github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/jollaman999/utils/fileutil"
	"github.com/jollaman999/utils/logger"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type cmHoneybeeConfig struct {
	CMHoneybee struct {
		Listen struct {
			Port string `yaml:"port"`
		} `yaml:"listen"`
		Agent struct {
			Port string `yaml:"port"`
		} `yaml:"agent"`
	} `yaml:"cm-honeybee"`
}

var CMHoneybeeConfig cmHoneybeeConfig
var cmHoneybeeConfigFile = "cm-honeybee.yaml"

func checkCMHoneybeeConfigFile() error {
	if CMHoneybeeConfig.CMHoneybee.Listen.Port == "" {
		return errors.New("config error: cm-honeybee.listen.port is empty")
	}
	port, err := strconv.Atoi(CMHoneybeeConfig.CMHoneybee.Listen.Port)
	if err != nil || port < 1 || port > 65535 {
		return errors.New("config error: cm-honeybee.listen.port has invalid value")
	}

	if CMHoneybeeConfig.CMHoneybee.Agent.Port == "" {
		return errors.New("config error: cm-honeybee.agent.port is empty")
	}
	port, err = strconv.Atoi(CMHoneybeeConfig.CMHoneybee.Agent.Port)
	if err != nil || port < 1 || port > 65535 {
		return errors.New("config error: cm-honeybee.agent.port has invalid value")
	}

	return nil
}

func getCMHoneybeeDefaultConfig() cmHoneybeeConfig {
	var defaultConfig cmHoneybeeConfig

	defaultConfig.CMHoneybee.Listen.Port = "8081"
	defaultConfig.CMHoneybee.Agent.Port = "8082"

	return defaultConfig
}

func readCMHoneybeeConfigFile() error {
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
		logger.Println(logger.WARN, false, "can't find the config file ("+cmHoneybeeConfigFile+")"+fmt.Sprintln()+
			"Must be placed in '."+strings.ToLower(common.ModuleName)+"/conf' directory "+
			"under user's home directory or 'conf' directory where running the binary "+
			"or 'conf' directory where placed in the path of '"+common.ModuleROOT+"' environment variable")
		logger.Println(logger.WARN, false, "Using default configuration...")
		CMHoneybeeConfig = getCMHoneybeeDefaultConfig()
	} else {
		err = yaml.Unmarshal(data, &CMHoneybeeConfig)
		if err != nil {
			return err
		}

		err = checkCMHoneybeeConfigFile()
		if err != nil {
			return err
		}
	}

	err = yaml.Unmarshal(data, &CMHoneybeeConfig)
	if err != nil {
		return err
	}

	err = checkCMHoneybeeConfigFile()
	if err != nil {
		return err
	}

	return nil
}

func prepareCMHoneybeeConfig() error {
	err := readCMHoneybeeConfigFile()
	if err != nil {
		return err
	}

	return nil
}
