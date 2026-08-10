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
		Spider struct {
			Endpoint string `yaml:"endpoint"`
			Username string `yaml:"username"`
			Password string `yaml:"password"`
		} `yaml:"spider"`
		OpenBao struct {
			// Address of the OpenBao server, e.g. http://openbao:8200. Empty
			// disables OpenBao — secrets stay in the local (SQLite) store.
			Address string `yaml:"address"`
			// Token, or TokenFile to read it from (TokenFile wins if Token empty).
			Token     string `yaml:"token"`
			TokenFile string `yaml:"token_file"`
		} `yaml:"openbao"`
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

	if CMHoneybeeConfig.CMHoneybee.Spider.Endpoint == "" {
		return errors.New("config error: cm-honeybee.spider.endpoint is empty")
	}

	return nil
}

func getCMHoneybeeDefaultConfig() cmHoneybeeConfig {
	var defaultConfig cmHoneybeeConfig

	defaultConfig.CMHoneybee.Listen.Port = "8081"
	defaultConfig.CMHoneybee.Agent.Port = "8082"
	defaultConfig.CMHoneybee.Spider.Endpoint = "http://localhost:1024/spider"
	defaultConfig.CMHoneybee.Spider.Username = "default"
	defaultConfig.CMHoneybee.Spider.Password = "default"

	return defaultConfig
}

// resolveEnvRef expands a "${VAR}" reference to the value of the environment
// variable VAR (process environment). A non-reference literal is returned
// unchanged. When s IS a "${VAR}" reference but VAR is unset or empty, it
// returns "" with refUnset=true so the caller can warn instead of silently
// using a default. Mirrors cm-mayfly's common.ResolveEnvRef.
func resolveEnvRef(s string) (value string, refUnset bool) {
	trimmed := strings.TrimSpace(s)
	if strings.HasPrefix(trimmed, "${") && strings.HasSuffix(trimmed, "}") {
		name := strings.TrimSpace(trimmed[2 : len(trimmed)-1])
		if name != "" {
			if v := os.Getenv(name); v != "" {
				return v, false
			}
			return "", true
		}
	}

	return s, false
}

// expandConfigEnvRefs expands ${VAR} references in the loaded config values from
// the process environment. Applied after unmarshal so operators can inject
// endpoints and credentials from the environment (e.g. SPIDER_USERNAME) instead
// of committing them to cm-honeybee.yaml. Literals are left unchanged.
func expandConfigEnvRefs() {
	fields := []struct {
		name     string
		ptr      *string
		optional bool // optional fields resolve an unset ${VAR} to "" silently
	}{
		{"cm-honeybee.listen.port", &CMHoneybeeConfig.CMHoneybee.Listen.Port, false},
		{"cm-honeybee.agent.port", &CMHoneybeeConfig.CMHoneybee.Agent.Port, false},
		{"cm-honeybee.spider.endpoint", &CMHoneybeeConfig.CMHoneybee.Spider.Endpoint, false},
		{"cm-honeybee.spider.username", &CMHoneybeeConfig.CMHoneybee.Spider.Username, false},
		{"cm-honeybee.spider.password", &CMHoneybeeConfig.CMHoneybee.Spider.Password, false},
		{"cm-honeybee.openbao.address", &CMHoneybeeConfig.CMHoneybee.OpenBao.Address, true},
		{"cm-honeybee.openbao.token", &CMHoneybeeConfig.CMHoneybee.OpenBao.Token, true},
		{"cm-honeybee.openbao.token_file", &CMHoneybeeConfig.CMHoneybee.OpenBao.TokenFile, true},
	}

	for _, f := range fields {
		value, refUnset := resolveEnvRef(*f.ptr)
		if refUnset && !f.optional {
			logger.Println(logger.WARN, false, "config: "+f.name+
				" references an unset environment variable; using empty value")
		}
		*f.ptr = value
	}
}

// applyConfigEnvOverrides lets operators configure cm-honeybee entirely from the
// environment, without mounting a cm-honeybee.yaml. A non-empty env var wins
// over whatever the config file / defaults provided. This is what lets a stock
// image run with just `environment:` entries (e.g. from cm-mayfly) instead of a
// mounted config file. Runs after expandConfigEnvRefs so both ${VAR} references
// in the file and these direct overrides are honored.
func applyConfigEnvOverrides() {
	set := func(ptr *string, env string) {
		if v := strings.TrimSpace(os.Getenv(env)); v != "" {
			*ptr = v
		}
	}
	c := &CMHoneybeeConfig.CMHoneybee
	set(&c.Listen.Port, "HONEYBEE_LISTEN_PORT")
	set(&c.Agent.Port, "HONEYBEE_AGENT_PORT")
	set(&c.Spider.Endpoint, "HONEYBEE_SPIDER_ENDPOINT")
	set(&c.Spider.Username, "HONEYBEE_SPIDER_USERNAME")
	set(&c.Spider.Password, "HONEYBEE_SPIDER_PASSWORD")
	set(&c.OpenBao.Address, "HONEYBEE_VAULT_ADDR")
	set(&c.OpenBao.Token, "HONEYBEE_VAULT_TOKEN")
	set(&c.OpenBao.TokenFile, "HONEYBEE_VAULT_TOKEN_FILE")
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
	}

	// Expand ${VAR} references from the process environment before validation so
	// endpoints and credentials can be injected via env (e.g. SPIDER_USERNAME /
	// SPIDER_PASSWORD) instead of being committed to the config file. Plain
	// literals are left unchanged, keeping existing configs working. Mirrors
	// cm-mayfly's ${VAR} resolution.
	expandConfigEnvRefs()

	// Native env overrides (HONEYBEE_SPIDER_*, HONEYBEE_VAULT_*, ...) so the
	// stock image can be configured entirely from the environment — no mounted
	// cm-honeybee.yaml required. A set env var wins over file/default values.
	applyConfigEnvOverrides()

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
