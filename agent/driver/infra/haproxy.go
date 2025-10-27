package infra

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/infra"
)

// GetHAProxyInfo collects HAProxy information from the system
func GetHAProxyInfo() (infra.HAProxy, error) {
	var haproxy infra.HAProxy
	var errors []string

	// Get HAProxy version
	version, err := getHAProxyVersion()
	if err != nil {
		errors = append(errors, fmt.Sprintf("Failed to get HAProxy version: %v", err))
	} else {
		haproxy.Version = version
	}

	// Check common HAProxy config file locations
	configPaths := []string{
		"/etc/haproxy/haproxy.cfg",
		"/usr/local/etc/haproxy/haproxy.cfg",
		"/opt/haproxy/haproxy.cfg",
	}

	var configPath string
	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			configPath = path
			break
		}
	}

	if configPath == "" {
		errors = append(errors, "HAProxy configuration file not found in common locations")
		haproxy.Errors = errors
		return haproxy, fmt.Errorf("HAProxy not found or not installed")
	}

	haproxy.ConfigPath = configPath

	// Parse configuration file
	config, err := parseHAProxyConfig(configPath)
	if err != nil {
		errors = append(errors, fmt.Sprintf("Failed to parse HAProxy config: %v", err))
	} else {
		haproxy.Global = config.Global
		haproxy.Defaults = config.Defaults
		haproxy.Frontends = config.Frontends
		haproxy.Backends = config.Backends
		haproxy.Listens = config.Listens
	}

	haproxy.Errors = errors
	return haproxy, nil
}

// getHAProxyVersion executes haproxy -v command to get version
func getHAProxyVersion() (string, error) {
	// Try common HAProxy binary locations
	binaries := []string{"haproxy", "/usr/sbin/haproxy", "/usr/local/sbin/haproxy"}

	for _, binary := range binaries {
		cmd := exec.Command(binary, "-v")
		output, err := cmd.Output()
		if err == nil {
			// Parse version from output (e.g., "HAProxy version 2.4.0-1ubuntu1 2021/04/02")
			lines := strings.Split(string(output), "\n")
			if len(lines) > 0 {
				return strings.TrimSpace(lines[0]), nil
			}
		}
	}

	return "", fmt.Errorf("haproxy binary not found")
}

// haproxyConfig represents parsed HAProxy configuration
type haproxyConfig struct {
	Global    map[string]string
	Defaults  map[string]string
	Frontends []infra.HAProxyFrontend
	Backends  []infra.HAProxyBackend
	Listens   []infra.HAProxyListen
}

// parseHAProxyConfig parses HAProxy configuration file
func parseHAProxyConfig(configPath string) (*haproxyConfig, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = file.Close()
	}()

	config := &haproxyConfig{
		Global:    make(map[string]string),
		Defaults:  make(map[string]string),
		Frontends: []infra.HAProxyFrontend{},
		Backends:  []infra.HAProxyBackend{},
		Listens:   []infra.HAProxyListen{},
	}

	scanner := bufio.NewScanner(file)
	var currentSection string
	var currentFrontend *infra.HAProxyFrontend
	var currentBackend *infra.HAProxyBackend
	var currentListen *infra.HAProxyListen

	// Regex patterns
	sectionRegex := regexp.MustCompile(`^(global|defaults|frontend|backend|listen)\s+(.*)$`)
	bindRegex := regexp.MustCompile(`^\s*bind\s+(.+)$`)
	serverRegex := regexp.MustCompile(`^\s*server\s+(\S+)\s+(\S+)(?:\s+(.+))?$`)
	optionRegex := regexp.MustCompile(`^\s*(\S+)\s+(.*)$`)

	for scanner.Scan() {
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)

		// Skip comments and empty lines
		if trimmedLine == "" || strings.HasPrefix(trimmedLine, "#") {
			continue
		}

		// Check for section headers
		if matches := sectionRegex.FindStringSubmatch(trimmedLine); matches != nil {
			section := matches[1]
			name := strings.TrimSpace(matches[2])

			// Save previous section
			if currentFrontend != nil {
				config.Frontends = append(config.Frontends, *currentFrontend)
				currentFrontend = nil
			}
			if currentBackend != nil {
				config.Backends = append(config.Backends, *currentBackend)
				currentBackend = nil
			}
			if currentListen != nil {
				config.Listens = append(config.Listens, *currentListen)
				currentListen = nil
			}

			currentSection = section

			switch section {
			case "frontend":
				currentFrontend = &infra.HAProxyFrontend{
					Name:    name,
					Options: make(map[string]string),
				}
			case "backend":
				currentBackend = &infra.HAProxyBackend{
					Name:    name,
					Options: make(map[string]string),
					Servers: []infra.HAProxyServer{},
				}
			case "listen":
				currentListen = &infra.HAProxyListen{
					Name:    name,
					Options: make(map[string]string),
					Servers: []infra.HAProxyServer{},
				}
			}
			continue
		}

		// Parse section content
		switch currentSection {
		case "global":
			if matches := optionRegex.FindStringSubmatch(trimmedLine); matches != nil {
				config.Global[matches[1]] = strings.TrimSpace(matches[2])
			}

		case "defaults":
			if matches := optionRegex.FindStringSubmatch(trimmedLine); matches != nil {
				config.Defaults[matches[1]] = strings.TrimSpace(matches[2])
			}

		case "frontend":
			if currentFrontend != nil {
				if matches := bindRegex.FindStringSubmatch(line); matches != nil {
					currentFrontend.Bind = strings.TrimSpace(matches[1])
				} else if strings.HasPrefix(trimmedLine, "default_backend") {
					currentFrontend.DefaultBackend = strings.TrimSpace(strings.TrimPrefix(trimmedLine, "default_backend"))
				} else if matches := optionRegex.FindStringSubmatch(trimmedLine); matches != nil {
					currentFrontend.Options[matches[1]] = strings.TrimSpace(matches[2])
				}
			}

		case "backend":
			if currentBackend != nil {
				if matches := serverRegex.FindStringSubmatch(line); matches != nil {
					server := infra.HAProxyServer{
						Name:    matches[1],
						Address: matches[2],
					}
					if len(matches) > 3 {
						server.Options = strings.TrimSpace(matches[3])
					}
					currentBackend.Servers = append(currentBackend.Servers, server)
				} else if strings.HasPrefix(trimmedLine, "balance") {
					currentBackend.Balance = strings.TrimSpace(strings.TrimPrefix(trimmedLine, "balance"))
				} else if matches := optionRegex.FindStringSubmatch(trimmedLine); matches != nil {
					currentBackend.Options[matches[1]] = strings.TrimSpace(matches[2])
				}
			}

		case "listen":
			if currentListen != nil {
				if matches := bindRegex.FindStringSubmatch(line); matches != nil {
					currentListen.Bind = strings.TrimSpace(matches[1])
				} else if matches := serverRegex.FindStringSubmatch(line); matches != nil {
					server := infra.HAProxyServer{
						Name:    matches[1],
						Address: matches[2],
					}
					if len(matches) > 3 {
						server.Options = strings.TrimSpace(matches[3])
					}
					currentListen.Servers = append(currentListen.Servers, server)
				} else if strings.HasPrefix(trimmedLine, "balance") {
					currentListen.Balance = strings.TrimSpace(strings.TrimPrefix(trimmedLine, "balance"))
				} else if matches := optionRegex.FindStringSubmatch(trimmedLine); matches != nil {
					currentListen.Options[matches[1]] = strings.TrimSpace(matches[2])
				}
			}
		}
	}

	// Save last section
	if currentFrontend != nil {
		config.Frontends = append(config.Frontends, *currentFrontend)
	}
	if currentBackend != nil {
		config.Backends = append(config.Backends, *currentBackend)
	}
	if currentListen != nil {
		config.Listens = append(config.Listens, *currentListen)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return config, nil
}
