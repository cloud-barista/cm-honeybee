package software

import (
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/jollaman999/utils/logger"
)

func GetDockerContainers() ([]types.Container, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		logger.Print(logger.DEBUG, true, "DOCKER: "+err.Error())
		return []types.Container{}, err
	}

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	if err != nil {
		logger.Print(logger.ERROR, true, "DOCKER: "+err.Error())
		return []types.Container{}, err
	}

	return containers, nil
}
