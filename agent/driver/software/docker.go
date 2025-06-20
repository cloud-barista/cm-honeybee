package software

import (
	"context"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/jollaman999/utils/logger"
	"strings"
)

func isRealDocker(cli *client.Client) (bool, error) {
	info, err := cli.Info(context.Background())
	if err != nil {
		logger.Println(logger.ERROR, true, "DOCKER: Failed to get information of the docker.")
		return false, err
	}

	initBinary := info.InitBinary
	if strings.Contains(strings.ToLower(initBinary), "docker") {
		return true, nil
	}

	return false, nil
}

func GetDockerContainers() ([]container.Summary, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		logger.Println(logger.DEBUG, true, "DOCKER: "+err.Error())
		return []container.Summary{}, err
	}
	cli.NegotiateAPIVersion(ctx)

	yes, err := isRealDocker(cli)
	if err != nil {
		logger.Println(logger.ERROR, true, "DOCKER: "+err.Error())
		return []container.Summary{}, err
	}
	if !yes {
		logger.Println(logger.INFO, true, "DOCKER: Docker not found.")
		return []container.Summary{}, nil
	}

	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		logger.Println(logger.ERROR, true, "DOCKER: "+err.Error())
		return []container.Summary{}, err
	}

	return containers, nil
}
