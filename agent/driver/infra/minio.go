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

// GetMinIOInfo collects MinIO information from the system
func GetMinIOInfo() (infra.MinIO, error) {
	var minio infra.MinIO
	var errors []string

	// Get MinIO version
	version, err := getMinIOVersion()
	if err != nil {
		errors = append(errors, fmt.Sprintf("Failed to get MinIO version: %v", err))
	} else {
		minio.Version = version
	}

	// Check if MinIO is running and get process information
	processInfo, err := getMinIOProcessInfo()
	if err != nil {
		errors = append(errors, fmt.Sprintf("Failed to get MinIO process info: %v", err))
	} else {
		minio.ProcessInfo = processInfo
	}

	// Check common MinIO config and data directories
	configPaths := []string{
		"/etc/minio",
		"/etc/default/minio",
		os.Getenv("MINIO_CONFIG_ENV_FILE"),
		"$HOME/.minio",
	}

	var configPath string
	for _, path := range configPaths {
		if path == "" {
			continue
		}
		// Expand environment variables
		expandedPath := os.ExpandEnv(path)
		if _, err := os.Stat(expandedPath); err == nil {
			configPath = expandedPath
			break
		}
	}

	if configPath != "" {
		minio.ConfigPath = configPath

		// Parse configuration
		config, err := parseMinIOConfig(configPath)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Failed to parse MinIO config: %v", err))
		} else {
			minio.RootUser = config["MINIO_ROOT_USER"]
			minio.Volumes = config["MINIO_VOLUMES"]
			minio.Address = config["MINIO_ADDRESS"]
			minio.ConsoleAddress = config["MINIO_CONSOLE_ADDRESS"]
			minio.Opts = config
		}
	} else {
		errors = append(errors, "MinIO configuration file not found")
	}

	// Try to get storage information from common volume paths
	volumePaths := []string{
		"/data",
		"/mnt/minio",
		"/var/lib/minio",
	}

	if minio.Volumes != "" {
		// Parse volumes from config
		volumePaths = append(volumePaths, strings.Fields(minio.Volumes)...)
	}

	for _, volPath := range volumePaths {
		if stat, err := os.Stat(volPath); err == nil && stat.IsDir() {
			if diskInfo, err := getDiskUsage(volPath); err == nil {
				minio.StoragePaths = append(minio.StoragePaths, infra.MinIOStorage{
					Path:        volPath,
					TotalGB:     diskInfo.TotalGB,
					UsedGB:      diskInfo.UsedGB,
					FreeGB:      diskInfo.FreeGB,
					UsedPercent: diskInfo.UsedPercent,
				})
			}
		}
	}

	minio.Errors = errors
	return minio, nil
}

// getMinIOVersion executes minio --version command to get version
func getMinIOVersion() (string, error) {
	// Try common MinIO binary locations
	binaries := []string{"minio", "/usr/local/bin/minio", "/usr/bin/minio", "/opt/minio/minio"}

	for _, binary := range binaries {
		cmd := exec.Command(binary, "--version")
		output, err := cmd.Output()
		if err == nil {
			// Parse version from output
			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				if strings.Contains(line, "Version") || strings.Contains(line, "version") {
					return strings.TrimSpace(line), nil
				}
			}
			if len(lines) > 0 {
				return strings.TrimSpace(lines[0]), nil
			}
		}
	}

	return "", fmt.Errorf("minio binary not found")
}

// getMinIOProcessInfo checks if MinIO is running and gets process info
func getMinIOProcessInfo() (string, error) {
	cmd := exec.Command("ps", "aux")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "minio") && strings.Contains(line, "server") {
			return strings.TrimSpace(line), nil
		}
	}

	return "", fmt.Errorf("MinIO process not found")
}

// parseMinIOConfig parses MinIO configuration file
func parseMinIOConfig(configPath string) (map[string]string, error) {
	config := make(map[string]string)

	// Check if it's a directory or file
	stat, err := os.Stat(configPath)
	if err != nil {
		return nil, err
	}

	var filePath string
	if stat.IsDir() {
		// Look for config files in directory
		possibleFiles := []string{
			configPath + "/config.env",
			configPath + "/minio",
			configPath + "/.env",
		}
		for _, f := range possibleFiles {
			if _, err := os.Stat(f); err == nil {
				filePath = f
				break
			}
		}
		if filePath == "" {
			return nil, fmt.Errorf("no config file found in directory")
		}
	} else {
		filePath = configPath
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = file.Close()
	}()

	scanner := bufio.NewScanner(file)
	// Regex to match KEY=VALUE or export KEY=VALUE
	envRegex := regexp.MustCompile(`^(?:export\s+)?([A-Z_][A-Z0-9_]*)\s*=\s*["']?(.+?)["']?\s*$`)

	for scanner.Scan() {
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)

		// Skip comments and empty lines
		if trimmedLine == "" || strings.HasPrefix(trimmedLine, "#") {
			continue
		}

		if matches := envRegex.FindStringSubmatch(trimmedLine); matches != nil {
			key := matches[1]
			value := strings.Trim(matches[2], `"'`)
			config[key] = value
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return config, nil
}

// diskUsageInfo represents disk usage information
type diskUsageInfo struct {
	TotalGB     float64
	UsedGB      float64
	FreeGB      float64
	UsedPercent float64
}

// getDiskUsage gets disk usage for a specific path using df command
func getDiskUsage(path string) (*diskUsageInfo, error) {
	cmd := exec.Command("df", "-BG", path)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) < 2 {
		return nil, fmt.Errorf("unexpected df output")
	}

	// Parse df output (skip header)
	fields := strings.Fields(lines[1])
	if len(fields) < 5 {
		return nil, fmt.Errorf("unexpected df output format")
	}

	// Parse values (remove 'G' suffix)
	totalStr := strings.TrimSuffix(fields[1], "G")
	usedStr := strings.TrimSuffix(fields[2], "G")
	freeStr := strings.TrimSuffix(fields[3], "G")
	percentStr := strings.TrimSuffix(fields[4], "%")

	var total, used, free, percent float64
	_, _ = fmt.Sscanf(totalStr, "%f", &total)
	_, _ = fmt.Sscanf(usedStr, "%f", &used)
	_, _ = fmt.Sscanf(freeStr, "%f", &free)
	_, _ = fmt.Sscanf(percentStr, "%f", &percent)

	return &diskUsageInfo{
		TotalGB:     total,
		UsedGB:      used,
		FreeGB:      free,
		UsedPercent: percent,
	}, nil
}
