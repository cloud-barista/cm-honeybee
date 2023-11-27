package software

import (
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

func GetContainers() ([]types.Container, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	if err != nil {
		return nil, err
	}

	return containers, nil
}
