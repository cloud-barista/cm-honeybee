package software

import (
	"github.com/docker/docker/api/types"
	"github.com/jollaman999/utils/logger"
	"github.com/shirou/gopsutil/v3/host"
)

type Docker struct {
	Containers []types.Container
	//Images     []types.ImageMetadata
}

type Podman struct {
	Containers []types.Container
	//Images     []types.ImageMetadata
}

type Software struct {
	DEB    []DEB  `json:"deb"`
	RPM    []RPM  `json:"rpm"`
	Docker Docker `json:"docker"`
	Podman Podman `json:"podman"`
}

func GetSoftwareInfo() (*Software, error) {
	deb := make([]DEB, 0)
	rpm := make([]RPM, 0)
	var err error

	h, err := host.Info()
	if err != nil {
		return nil, err
	}

	if h.PlatformFamily == "debian" {
		deb, err = GetDEBs()
		if err != nil {
			return nil, err
		}
	}

	if h.PlatformFamily == "fedora" || h.PlatformFamily == "rhel" {
		rpm, err = GetRPMs()
		if err != nil {
			return nil, err
		}
	}

	dockerContainers, err := GetDockerContainers()
	if err != nil {
		logger.Println(logger.DEBUG, true, "DOCKER: "+err.Error())
	}

	podmanContainers, err := GetPodmanContainers()
	if err != nil {
		logger.Println(logger.DEBUG, true, "PODMAN: "+err.Error())
	}

	software := Software{
		DEB: deb,
		RPM: rpm,
		Docker: Docker{
			Containers: dockerContainers,
		},
		Podman: Podman{
			Containers: podmanContainers,
		},
	}

	return &software, nil
}
