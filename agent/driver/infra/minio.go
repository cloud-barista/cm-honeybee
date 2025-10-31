package infra

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/infra"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// minioConnectionInfo holds MinIO connection information
type minioConnectionInfo struct {
	Endpoints []string
	AccessKey string
	SecretKey string
	Source    string // "docker", "config", "binary"
	Version   string
}

// GetMinIOInfo collects MinIO information from the system
func GetMinIOInfo() (infra.MinIO, error) {
	var minioInfo infra.MinIO
	var errors []string

	// Get MinIO version
	version, err := getMinIOVersion()
	if err != nil {
		errors = append(errors, fmt.Sprintf("Failed to get MinIO version: %v", err))
	} else {
		minioInfo.Version = version
	}

	// Check if MinIO is running and get process information
	processInfo, err := getMinIOProcessInfo()
	if err != nil {
		errors = append(errors, fmt.Sprintf("Failed to get MinIO process info: %v", err))
	} else {
		minioInfo.ProcessInfo = processInfo
	}

	// Try to get connection info from Docker first
	var connInfo *minioConnectionInfo
	if dockerConnInfo, err := getMinIODockerInfo(processInfo); err == nil && dockerConnInfo != nil {
		connInfo = dockerConnInfo
		minioInfo.RootUser = dockerConnInfo.AccessKey
		if dockerConnInfo.Version != "" {
			minioInfo.Version = dockerConnInfo.Version
		}
	} else if err != nil {
		errors = append(errors, fmt.Sprintf("Failed to check Docker MinIO: %v", err))
	}

	// If not found in Docker, try config files
	if connInfo == nil {
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
			expandedPath := os.ExpandEnv(path)
			if _, err := os.Stat(expandedPath); err == nil {
				configPath = expandedPath
				break
			}
		}

		if configPath != "" {
			minioInfo.ConfigPath = configPath

			config, err := parseMinIOConfig(configPath)
			if err != nil {
				errors = append(errors, fmt.Sprintf("Failed to parse MinIO config: %v", err))
			} else {
				minioInfo.RootUser = config["MINIO_ROOT_USER"]
				minioInfo.Volumes = config["MINIO_VOLUMES"]
				minioInfo.Address = config["MINIO_ADDRESS"]
				minioInfo.ConsoleAddress = config["MINIO_CONSOLE_ADDRESS"]
				minioInfo.Opts = config

				endpoint := minioInfo.Address
				if endpoint == "" {
					endpoint = "localhost:9000"
				}
				endpoint = strings.TrimPrefix(strings.TrimPrefix(endpoint, "http://"), "https://")

				accessKey := config["MINIO_ROOT_USER"]
				secretKey := config["MINIO_ROOT_PASSWORD"]

				if accessKey != "" && secretKey != "" {
					connInfo = &minioConnectionInfo{
						Endpoints: []string{endpoint},
						AccessKey: accessKey,
						SecretKey: secretKey,
						Source:    "config",
					}
				}
			}
		} else {
			errors = append(errors, "MinIO configuration file not found")
		}
	}

	// Connect to MinIO and get information
	if connInfo != nil {
		var serverInfo *infra.MinIOServerInfo
		var buckets []infra.MinioBucket
		var lastServerErr, lastBucketsErr error

		// Try all endpoints
		for _, endpoint := range connInfo.Endpoints {
			// Try to get server information
			if serverInfo == nil {
				if info, err := getMinIOServerInfo(endpoint, connInfo.AccessKey, connInfo.SecretKey); err == nil {
					serverInfo = info
					minioInfo.Address = endpoint
				} else {
					lastServerErr = err
				}
			}

			// Try to get bucket information (using same endpoint that worked for server info, or trying all)
			if len(buckets) == 0 {
				if b, err := getMinioBuckets(endpoint, connInfo.AccessKey, connInfo.SecretKey); err == nil {
					buckets = b
					if minioInfo.Address == "" {
						minioInfo.Address = endpoint
					}
				} else {
					lastBucketsErr = err
				}
			}

			// If both succeeded, no need to try other endpoints
			if serverInfo != nil && buckets != nil {
				break
			}
		}

		// Set results
		if serverInfo != nil {
			minioInfo.ServerInfo = serverInfo
		} else if lastServerErr != nil {
			errors = append(errors, fmt.Sprintf("Failed to get MinIO server info from all endpoints: %v", lastServerErr))
		}

		if buckets != nil {
			minioInfo.Buckets = buckets
		} else if lastBucketsErr != nil {
			errors = append(errors, fmt.Sprintf("Failed to get MinIO buckets from all endpoints: %v", lastBucketsErr))
		}
	} else {
		errors = append(errors, "MinIO credentials not found")
	}

	minioInfo.Errors = errors
	return minioInfo, nil
}

// parseMinIOPorts extracts ports from MinIO process command line
func parseMinIOPorts(processInfo string) (apiPort, consolePort int) {
	// Default ports
	apiPort = 9000
	consolePort = 9001

	if processInfo == "" {
		return
	}

	// Split command line into parts
	parts := strings.Fields(processInfo)

	// Look for --address flag for API port
	for i, part := range parts {
		if part == "--address" && i+1 < len(parts) {
			addr := parts[i+1]
			// Parse :port from address
			if strings.Contains(addr, ":") {
				portStr := strings.TrimPrefix(addr, ":")
				if p, err := strconv.Atoi(portStr); err == nil && p > 0 {
					apiPort = p
				}
			}
		}
		if part == "--console-address" && i+1 < len(parts) {
			addr := parts[i+1]
			// Parse :port from address
			if strings.Contains(addr, ":") {
				portStr := strings.TrimPrefix(addr, ":")
				if p, err := strconv.Atoi(portStr); err == nil && p > 0 {
					consolePort = p
				}
			}
		}
	}

	// Also check for inline flags like --address=:9000
	for _, part := range parts {
		if strings.HasPrefix(part, "--address=") {
			addr := strings.TrimPrefix(part, "--address=")
			if strings.Contains(addr, ":") {
				portStr := strings.TrimPrefix(addr, ":")
				if p, err := strconv.Atoi(portStr); err == nil && p > 0 {
					apiPort = p
				}
			}
		}
		if strings.HasPrefix(part, "--console-address=") {
			addr := strings.TrimPrefix(part, "--console-address=")
			if strings.Contains(addr, ":") {
				portStr := strings.TrimPrefix(addr, ":")
				if p, err := strconv.Atoi(portStr); err == nil && p > 0 {
					consolePort = p
				}
			}
		}
	}

	return
}

// getMinIODockerInfo checks if MinIO is running in Docker and extracts connection info
func getMinIODockerInfo(processInfo string) (*minioConnectionInfo, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cli.NegotiateAPIVersion(ctx)

	// List all containers
	containers, err := cli.ContainerList(ctx, container.ListOptions{All: false})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	// Look for MinIO container
	for _, c := range containers {
		// Check if container image or name contains "minio"
		isMinIO := strings.Contains(strings.ToLower(c.Image), "minio")
		if !isMinIO {
			for _, name := range c.Names {
				if strings.Contains(strings.ToLower(name), "minio") {
					isMinIO = true
					break
				}
			}
		}

		if !isMinIO {
			continue
		}

		// Inspect container to get environment variables
		containerInspect, err := cli.ContainerInspect(ctx, c.ID)
		if err != nil {
			continue
		}

		// Extract credentials from environment variables
		var accessKey, secretKey string
		for _, env := range containerInspect.Config.Env {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) != 2 {
				continue
			}
			key, value := parts[0], parts[1]

			switch key {
			case "MINIO_ROOT_USER", "MINIO_ACCESS_KEY":
				accessKey = value
			case "MINIO_ROOT_PASSWORD", "MINIO_SECRET_KEY":
				secretKey = value
			}
		}

		if accessKey == "" || secretKey == "" {
			continue
		}

		// Extract endpoint from network settings
		var endpoints []string

		// Check NetworkSettings for IP address
		if containerInspect.NetworkSettings != nil {
			// Try all networks (including default bridge network)
			for networkName, network := range containerInspect.NetworkSettings.Networks {
				if network.IPAddress != "" {
					endpoints = append(endpoints, fmt.Sprintf("%s:9000", network.IPAddress))
					// If this is the bridge network, also try it first
					if networkName == "bridge" && len(endpoints) > 1 {
						// Move bridge network to front
						endpoints[0], endpoints[len(endpoints)-1] = endpoints[len(endpoints)-1], endpoints[0]
					}
				}
			}
		}

		// Check all port mappings and bindings
		seenPorts := make(map[string]bool)

		// First, check HostConfig.PortBindings for accurate port mapping info
		if containerInspect.HostConfig != nil && containerInspect.HostConfig.PortBindings != nil {
			for containerPort, bindings := range containerInspect.HostConfig.PortBindings {
				portNum := containerPort.Int()
				if portNum >= 9000 && portNum <= 9010 {
					for _, binding := range bindings {
						hostIP := binding.HostIP
						if hostIP == "" || hostIP == "0.0.0.0" || hostIP == "::" {
							hostIP = "localhost"
						}
						hostPort := binding.HostPort
						if hostPort != "" {
							endpoint := fmt.Sprintf("%s:%s", hostIP, hostPort)
							if !seenPorts[endpoint] {
								// Prioritize port 9000 (API port) over others
								if portNum == 9000 {
									endpoints = append([]string{endpoint}, endpoints...)
								} else {
									endpoints = append(endpoints, endpoint)
								}
								seenPorts[endpoint] = true
							}
						}
					}
				}
			}
		}

		// Also check container.Ports as fallback
		for _, port := range c.Ports {
			// Only interested in MinIO-related ports (typically 9000-9001, but could be custom)
			if port.PrivatePort >= 9000 && port.PrivatePort <= 9010 {
				var endpoint string
				if port.PublicPort > 0 {
					// Port is exposed to host
					endpoint = fmt.Sprintf("localhost:%d", port.PublicPort)
				} else if port.IP != "" {
					// Port is bound to specific IP
					endpoint = fmt.Sprintf("%s:%d", port.IP, port.PrivatePort)
				}

				if endpoint != "" && !seenPorts[endpoint] {
					// Prioritize port 9000 (API port) over others
					if port.PrivatePort == 9000 || port.PublicPort == 9000 {
						endpoints = append([]string{endpoint}, endpoints...)
					} else {
						endpoints = append(endpoints, endpoint)
					}
					seenPorts[endpoint] = true
				}
			}
		}

		// Check for host network mode or if no endpoints found
		if containerInspect.HostConfig != nil && containerInspect.HostConfig.NetworkMode.IsHost() || len(endpoints) == 0 {
			// Parse ports from process info
			apiPort, _ := parseMinIOPorts(processInfo)
			if apiPort > 0 {
				endpoints = append(endpoints, fmt.Sprintf("localhost:%d", apiPort))
			}
			// Always add default port as fallback
			if apiPort != 9000 {
				endpoints = append(endpoints, "localhost:9000")
			}
		}

		// Extract version from image tag
		version := c.Image
		if strings.Contains(version, ":") {
			parts := strings.Split(version, ":")
			if len(parts) > 1 {
				version = "MinIO " + parts[len(parts)-1]
			}
		}

		return &minioConnectionInfo{
			Endpoints: endpoints,
			AccessKey: accessKey,
			SecretKey: secretKey,
			Source:    "docker",
			Version:   version,
		}, nil
	}

	return nil, nil // No MinIO container found
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

// getMinIOServerInfo connects to MinIO Admin API and retrieves server information
func getMinIOServerInfo(endpoint, accessKey, secretKey string) (*infra.MinIOServerInfo, error) {
	ctx := context.Background()

	// Create admin client
	madmClient, err := madmin.NewWithOptions(endpoint, &madmin.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create admin client: %w", err)
	}

	// Set timeout
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Get server info
	info, err := madmClient.ServerInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get server info: %w", err)
	}

	serverInfo := &infra.MinIOServerInfo{
		Servers: len(info.Servers),
	}

	var totalBytes, usedBytes uint64
	totalDisks := 0
	onlineDisks := 0
	offlineDisks := 0

	// Aggregate disk information from all servers
	for _, server := range info.Servers {
		for _, disk := range server.Disks {
			totalDisks++
			if disk.State == "ok" {
				onlineDisks++
			} else {
				offlineDisks++
			}
			totalBytes += disk.TotalSpace
			usedBytes += disk.UsedSpace
		}
	}

	serverInfo.Disks = totalDisks
	serverInfo.OnlineDisks = onlineDisks
	serverInfo.OfflineDisks = offlineDisks

	// Convert bytes to GB
	totalGB := float64(totalBytes) / (1024 * 1024 * 1024)
	usedGB := float64(usedBytes) / (1024 * 1024 * 1024)
	freeGB := totalGB - usedGB
	usedPercent := 0.0
	if totalGB > 0 {
		usedPercent = (usedGB / totalGB) * 100
	}

	serverInfo.StorageInfo = infra.MinIOStorageInfo{
		TotalGB:     totalGB,
		UsedGB:      usedGB,
		FreeGB:      freeGB,
		UsedPercent: usedPercent,
	}

	return serverInfo, nil
}

// getMinioBuckets connects to MinIO and retrieves bucket information
func getMinioBuckets(endpoint, accessKey, secretKey string) ([]infra.MinioBucket, error) {
	ctx := context.Background()

	// Create MinIO client
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	// Set timeout
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// List buckets
	buckets, err := minioClient.ListBuckets(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list buckets: %w", err)
	}

	var result []infra.MinioBucket

	for _, bucket := range buckets {
		bucketInfo := infra.MinioBucket{
			Name: bucket.Name,
		}

		// Get bucket versioning status
		versioningCtx, versioningCancel := context.WithTimeout(ctx, 5*time.Second)
		versioning, err := minioClient.GetBucketVersioning(versioningCtx, bucket.Name)
		versioningCancel()
		if err == nil {
			bucketInfo.Versioned = versioning.Status == "Enabled"
		}

		// Get bucket replication status
		replicationCtx, replicationCancel := context.WithTimeout(ctx, 5*time.Second)
		_, err = minioClient.GetBucketReplication(replicationCtx, bucket.Name)
		replicationCancel()
		if err == nil {
			bucketInfo.Replication = true
		}

		// Calculate bucket size and object count
		var totalSize int64
		var objectCount int64

		objectsCtx, objectsCancel := context.WithTimeout(ctx, 30*time.Second)
		objectCh := minioClient.ListObjects(objectsCtx, bucket.Name, minio.ListObjectsOptions{
			Recursive: true,
		})

		for object := range objectCh {
			if object.Err == nil {
				totalSize += object.Size
				objectCount++
			}
		}
		objectsCancel()

		bucketInfo.SizeGB = float64(totalSize) / (1024 * 1024 * 1024)
		bucketInfo.Objects = objectCount

		result = append(result, bucketInfo)
	}

	return result, nil
}
