// Podman implementation for Linux

//go:build linux && !android

package software

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/jollaman999/utils/logger"
	"os"
	"os/exec"
)

func startPodmanSocketService() error {
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

type remoteSocket struct {
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
}

func checkPodman() error {
	cmd := exec.Command("sh", "-c", "podman --help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.New(string(output))
	}

	return nil
}

func getPodmanSocketPath() (string, error) {
	cmd := exec.Command("sh", "-c", "podman system info -f '{{json .Host.RemoteSocket}}'")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.New(string(output))
	}

	var remoteSocket remoteSocket
	err = json.Unmarshal(output, &remoteSocket)
	if err != nil {
		return "", err
	}

	if !remoteSocket.Exists {
		return "", errors.New("socket path '" + remoteSocket.Path + "' is not exist.")
	}

	return remoteSocket.Path, nil
}

func newPodmanClient() (*client.Client, error) {
	err := checkPodman()
	if err != nil {
		errMsg := "Podman not found."
		logger.Print(logger.DEBUG, true, "PODMAN: "+errMsg)
		return nil, err
	}

	err = startPodmanSocketService()
	if err != nil {
		logger.Print(logger.ERROR, true, "PODMAN: "+err.Error())
		return nil, err
	}

	socketPath, err := getPodmanSocketPath()
	if err != nil {
		logger.Print(logger.ERROR, true, "PODMAN: "+err.Error())
		return nil, err
	}

	socket := "unix://" + socketPath
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
