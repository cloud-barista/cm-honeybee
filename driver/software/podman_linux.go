// Podman implementation for Linux

//go:build linux && !android

package software

import (
	"context"
	"errors"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/jollaman999/utils/logger"
	"os"
	"os/exec"
)

func StartPodmanSocketService() error {
	cmd := exec.Command("sh", "-c", "systemctl status podman.socket")
	output, err := cmd.CombinedOutput()
	if exiterr, ok := err.(*exec.ExitError); ok {
		if exiterr.ExitCode() == 3 { // systemctl exit code 3 : service is not started
			cmd = exec.Command("sh", "-c", "systemctl start podman.socket")
			output, err = cmd.CombinedOutput()
			if err != nil {
				return errors.New(string(output))
			}
		} else {
			return errors.New(string(output))
		}
	}

	return nil
}

func newPodmanClient() (*client.Client, error) {
	err := StartPodmanSocketService()
	if err != nil {
		logger.Print(logger.DEBUG, true, "PODMAN: "+err.Error())
		return nil, err
	}

	socket := "unix:///var/run/podman/podman.sock"

	err = os.Setenv(client.EnvOverrideHost, socket)
	defer func() {
		_ = os.Unsetenv(client.EnvOverrideHost)
	}()
	if err != nil {
		logger.Print(logger.ERROR, true, "PODMAN: "+err.Error())
		return nil, err
	}

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		logger.Print(logger.ERROR, true, "PODMAN: "+err.Error())
		return nil, err
	}

	return cli, nil
}

func GetPodmanContainers() ([]types.Container, error) {
	cli, err := newPodmanClient()
	if err != nil {
		return []types.Container{}, err
	}

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	if err != nil {
		return []types.Container{}, err
	}

	return containers, nil
}
