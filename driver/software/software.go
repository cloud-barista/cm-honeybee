package software

import (
	"github.com/cloud-barista/cm-honeybee/model/software"
	"github.com/jollaman999/utils/logger"
	"github.com/shirou/gopsutil/v3/host"
)

func GetSoftwareInfo() (*software.Software, error) {
	deb := make([]software.DEB, 0)
	rpm := make([]software.RPM, 0)
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

	sw := software.Software{
		DEB: deb,
		RPM: rpm,
		Docker: software.Docker{
			Containers: dockerContainers,
		},
		Podman: software.Podman{
			Containers: podmanContainers,
		},
	}

	return &sw, nil
}
