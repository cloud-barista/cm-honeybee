package software

import (
	"context"
	"errors"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/software"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/jollaman999/utils/cmd"
	"github.com/jollaman999/utils/logger"
	"os"
	"strings"
)

func isRealDocker(cli *client.Client) (bool, error) {
	info, err := cli.Info(context.Background())
	if err != nil {
		logger.Println(logger.DEBUG, true, "DOCKER: Failed to get information of the docker.")
		return false, err
	}

	initBinary := info.InitBinary
	if strings.Contains(strings.ToLower(initBinary), "docker") {
		return true, nil
	}

	return false, nil
}

// getDockerHost returns the docker daemon endpoint. The socket location may not be the
// standard path (/var/run/docker.sock) depending on the distro, rootless mode, or a custom
// context, so instead of guessing paths we ask the docker CLI for the endpoint it actually
// uses, the same way podman resolves its socket via `podman system info`.
func getDockerHost() (string, error) {
	// Respect DOCKER_HOST if it is set explicitly (handled by client.FromEnv).
	if os.Getenv(client.EnvOverrideHost) != "" {
		return "", nil
	}

	output, err := cmd.RunCMD("docker context inspect --format '{{.Endpoints.docker.Host}}'")
	if err != nil {
		return "", err
	}

	host := strings.TrimSpace(output)
	if host == "" {
		return "", errors.New("failed to resolve docker endpoint")
	}

	return host, nil
}

func newDockerClient() (*client.Client, error) {
	opts := []client.Opt{client.FromEnv}

	// When DOCKER_HOST is unset, connect to the endpoint the docker CLI actually uses
	// (avoids assuming a standard socket path).
	host, err := getDockerHost()
	if err != nil {
		logger.Println(logger.DEBUG, true, "DOCKER: "+err.Error())
	} else if host != "" {
		opts = append(opts, client.WithHost(host))
	}

	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		logger.Println(logger.DEBUG, true, "DOCKER: "+err.Error())
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cli.NegotiateAPIVersion(ctx)

	return cli, nil
}

func GetDockerContainers() ([]software.Container, error) {
	var result []software.Container

	cli, err := newDockerClient()
	if err != nil {
		return []software.Container{}, err
	}
	defer func() {
		_ = cli.Close()
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	yes, err := isRealDocker(cli)
	if err != nil {
		logger.Println(logger.DEBUG, true, "DOCKER: "+err.Error())
		return []software.Container{}, err
	}
	if !yes {
		logger.Println(logger.INFO, true, "DOCKER: Docker not found.")
		return []software.Container{}, nil
	}

	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		logger.Println(logger.ERROR, true, "DOCKER: "+err.Error())
		return []software.Container{}, err
	}

	for _, c := range containers {
		containerInspect, err := cli.ContainerInspect(ctx, c.ID)
		if err != nil {
			logger.Println(logger.ERROR, true, "DOCKER: "+err.Error())
		}

		imageInspect, err := cli.ImageInspect(ctx, c.ImageID)
		if err != nil {
			logger.Println(logger.ERROR, true, "DOCKER: "+err.Error())
		}

		result = append(result, software.Container{
			ContainerSummary: c,
			ContainerInspect: containerInspect,
			ImageInspect:     imageInspect,
		})
	}

	return result, nil
}
