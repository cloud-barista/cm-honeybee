package software

import (
	"github.com/docker/docker/api/types"
	"github.com/shirou/gopsutil/v3/host"
)

type Docker struct {
	Containers []types.Container
	//Images     []types.ImageMetadata
}

type Software struct {
	DEB    []DEB  `json:"deb"`
	RPM    []RPM  `json:"rpm"`
	Docker Docker `json:"docker"`
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

	containers, err := GetContainers()
	if err != nil {
		return nil, err
	}

	software := Software{
		DEB:    deb,
		RPM:    rpm,
		Docker: Docker{Containers: containers},
	}

	return &software, nil
}
