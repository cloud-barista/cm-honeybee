// Podman implementation for Linux

//go:build linux && !android

package software

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/software"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/jollaman999/utils/cmd"
	"github.com/jollaman999/utils/logger"
	"os"
	"os/exec"
)

func startPodmanSocketService() error {
	_, err := cmd.RunCMD("systemctl status podman.socket")
	if exiterr, ok := err.(*exec.ExitError); ok {
		if exiterr.ExitCode() == 3 { // systemctl exit code 3 : service is not started
			_, err = cmd.RunCMD("systemctl start podman.socket")
			if err != nil {
				return err
			}
		}

		return err
	}

	return nil
}

type remoteSocket struct {
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
}

func checkPodman() error {
	_, err := cmd.RunCMD("podman --help")
	if err != nil {
		return err
	}

	return nil
}

func getPodmanSocketPath() (string, error) {
	output, err := cmd.RunCMD("podman system info -f '{{json .Host.RemoteSocket}}'")
	if err != nil {
		return "", err
	}

	var remoteSocket remoteSocket
	err = json.Unmarshal([]byte(output), &remoteSocket)
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
		errMsg := "podman not found"
		logger.Println(logger.DEBUG, true, "PODMAN: "+errMsg)
		return nil, errors.New(errMsg)
	}

	err = startPodmanSocketService()
	if err != nil {
		logger.Println(logger.ERROR, true, "PODMAN: "+err.Error())
		return nil, err
	}

	socketPath, err := getPodmanSocketPath()
	if err != nil {
		logger.Println(logger.ERROR, true, "PODMAN: "+err.Error())
		return nil, err
	}

	socket := "unix://" + socketPath
	err = os.Setenv(client.EnvOverrideHost, socket)
	defer func() {
		_ = os.Unsetenv(client.EnvOverrideHost)
	}()
	if err != nil {
		logger.Println(logger.ERROR, true, "PODMAN: "+err.Error())
		return nil, err
	}

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		logger.Println(logger.ERROR, true, "PODMAN: "+err.Error())
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cli.NegotiateAPIVersion(ctx)

	return cli, nil
}

func GetPodmanContainers() ([]software.Container, error) {
	var result []software.Container

	cli, err := newPodmanClient()
	if err != nil {
		return []software.Container{}, err
	}

	containers, err := cli.ContainerList(context.Background(), container.ListOptions{All: true})
	if err != nil {
		logger.Println(logger.ERROR, true, "PODMAN: "+err.Error())
		return []software.Container{}, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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
