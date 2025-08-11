// Podman is not implemented for Windows

//go:build windows

package software

import (
	"errors"

	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/software"
)

func GetPodmanContainers() ([]software.Container, error) {
	return []software.Container{}, errors.New("getting podman information is not supported in Windows")
}
