// Podman is not implemented for Windows

//go:build windows

package software

import (
	"errors"
	"github.com/docker/docker/api/types"
)

func GetPodmanContainers() ([]types.Container, error) {
	return []types.Container{}, errors.New("getting podman information is not supported in Windows")
}
