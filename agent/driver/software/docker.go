package software

import (
	"context"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/software"
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

func GetDockerContainers() ([]software.Docker, error) {
	var result []software.Docker

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		logger.Println(logger.DEBUG, true, "DOCKER: "+err.Error())
		return []software.Docker{}, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cli.NegotiateAPIVersion(ctx)

	yes, err := isRealDocker(cli)
	if err != nil {
		logger.Println(logger.ERROR, true, "DOCKER: "+err.Error())
		return []software.Docker{}, err
	}
	if !yes {
		logger.Println(logger.INFO, true, "DOCKER: Docker not found.")
		return []software.Docker{}, nil
	}

	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		logger.Println(logger.ERROR, true, "DOCKER: "+err.Error())
		return []software.Docker{}, err
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

		result = append(result, software.Docker{
			ContainerSummary: c,
			ContainerInspect: containerInspect,
			ImageInspect:     imageInspect,
		})
	}

	return result, nil
}
