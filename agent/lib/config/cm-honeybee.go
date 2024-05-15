package config

import (
	"errors"
	"fmt"
	"github.com/cloud-barista/cm-honeybee/agent/common"
	"github.com/jollaman999/utils/fileutil"
	"gopkg.in/yaml.v3"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type cmHoneybeeAgentConfig struct {
	CMHoneybeeAgent struct {
		Server struct {
			Address string `yaml:"address"`
			Timeout string `yaml:"timeout"`
		} `yaml:"server"`
		Listen struct {
			Port string `yaml:"port"`
		} `yaml:"listen"`
	} `yaml:"cm-honeybee-agent"`
}

var CMHoneybeeAgentConfig cmHoneybeeAgentConfig
var cmHoneybeeAgentConfigFile = "cm-honeybee-agent.yaml"

func checkCMHoneybeeAgentConfigFile() error {
	if CMHoneybeeAgentConfig.CMHoneybeeAgent.Server.Address == "" {
		return errors.New("config error: cm-honeybee-agent.server.address is empty")
	}

	addrSplit := strings.Split(CMHoneybeeAgentConfig.CMHoneybeeAgent.Server.Address, ":")
	if len(addrSplit) < 2 {
		return errors.New("config error: invalid cm-honeybee-agent.server.address must be {IP or IPv6 or Domain}:{Port} form")
	}
	port, err := strconv.Atoi(addrSplit[len(addrSplit)-1])
	if err != nil || port < 1 || port > 65535 {
		return errors.New("config error: cm-honeybee-agent.server.address has invalid port value")
	}
	addr, _ := strings.CutSuffix(CMHoneybeeAgentConfig.CMHoneybeeAgent.Server.Address, ":"+strconv.Itoa(port))
	_, err = netip.ParseAddr(addr)
	if err != nil {
		_, err = net.LookupIP(addr)
		if err != nil {
			return errors.New("config error: cm-honeybee-agent.server.address has invalid address value " +
				"or can't find the domain (" + addr + ")")
		}
	}

	if CMHoneybeeAgentConfig.CMHoneybeeAgent.Server.Timeout == "" {
		return errors.New("config error: cm-honeybee-agent.server.timeout is empty")
	}

	timeout, err := strconv.Atoi(CMHoneybeeAgentConfig.CMHoneybeeAgent.Server.Timeout)
	if err != nil || timeout < 1 {
		return errors.New("config error: cm-honeybee-agent.server.timeout has invalid value")
	}

	if CMHoneybeeAgentConfig.CMHoneybeeAgent.Listen.Port == "" {
		return errors.New("config error: cm-honeybee-agent.listen.port is empty")
	}
	port, err = strconv.Atoi(CMHoneybeeAgentConfig.CMHoneybeeAgent.Listen.Port)
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
