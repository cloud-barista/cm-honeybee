package software

import (
	"errors"
	software2 "github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/software"
	"github.com/jollaman999/utils/logger"
	"github.com/shirou/gopsutil/v3/host"
	"sync"
)

var softwareInfoLock sync.Mutex

func GetSoftwareInfo(showDefaultPackages bool) (*software2.Software, error) {
	if !softwareInfoLock.TryLock() {
		return nil, errors.New("software info collection is in progress")
	}
	defer func() {
		softwareInfoLock.Unlock()
	}()

	deb := make([]software2.DEB, 0)
	rpm := make([]software2.RPM, 0)
	var err error

	h, err := host.Info()
	if err != nil {
		return nil, err
	}

	if h.PlatformFamily == "debian" {
		deb, err = GetDEBs(showDefaultPackages)
		if err != nil {
			return nil, err
		}
	}

	if h.PlatformFamily == "fedora" || h.PlatformFamily == "rhel" {
		rpm, err = GetRPMs(showDefaultPackages)
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

	sw := software2.Software{
		DEB: deb,
		RPM: rpm,
		Docker: software2.Docker{
			Containers: dockerContainers,
		},
		Podman: software2.Podman{
			Containers: podmanContainers,
		},
	}

	return &sw, nil
}
