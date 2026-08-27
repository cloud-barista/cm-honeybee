package software

import (
	"errors"
	"sync"
	"time"

	"github.com/cloud-barista/cm-honeybee/agent/common"
	software2 "github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/software"
	"github.com/jollaman999/utils/logger"
	"github.com/shirou/gopsutil/v3/host"
)

var softwareInfoLock sync.Mutex

func GetSoftwareInfo(showDefaultPackages bool) (*software2.Software, error) {
	if !softwareInfoLock.TryLock() {
		return nil, errors.New("software info collection is in progress")
	}
	defer func() {
		softwareInfoLock.Unlock()
	}()

	total := time.Now()
	defer func() {
		common.LogElapsed("software", "total", total, "")
	}()

	deb := make([]software2.DEB, 0)
	rpm := make([]software2.RPM, 0)
	var err error

	h, err := host.Info()
	if err != nil {
		return nil, err
	}

	if h.PlatformFamily == "debian" {
		start := time.Now()
		deb, err = GetDEBs(showDefaultPackages)
		common.LogElapsed("software", "deb", start, common.CountDetail(len(deb)))
		if err != nil {
			return nil, err
		}
	}

	if h.PlatformFamily == "fedora" || h.PlatformFamily == "rhel" {
		start := time.Now()
		rpm, err = GetRPMs(showDefaultPackages)
		common.LogElapsed("software", "rpm", start, common.CountDetail(len(rpm)))
		if err != nil {
			return nil, err
		}
	}

	start := time.Now()
	snaps, err := GetSnaps(showDefaultPackages)
	common.LogElapsed("software", "snap", start, common.CountDetail(len(snaps)))
	if err != nil {
		logger.Println(logger.DEBUG, true, "SNAP: "+err.Error())
	}

	start = time.Now()
	flatpaks, err := GetFlatpaks(showDefaultPackages)
	common.LogElapsed("software", "flatpak", start, common.CountDetail(len(flatpaks)))
	if err != nil {
		logger.Println(logger.DEBUG, true, "FLATPAK: "+err.Error())
	}

	start = time.Now()
	legacySW, err := GetLegacySWs()
	common.LogElapsed("software", "legacy", start, common.CountDetail(len(legacySW)))
	if err != nil {
		logger.Println(logger.DEBUG, true, "LegacySW: "+err.Error())
	}

	start = time.Now()
	dockerContainers, err := GetDockerContainers()
	common.LogElapsed("software", "docker", start, common.CountDetail(len(dockerContainers)))
	if err != nil {
		logger.Println(logger.DEBUG, true, "DOCKER: "+err.Error())
	}

	start = time.Now()
	podmanContainers, err := GetPodmanContainers()
	common.LogElapsed("software", "podman", start, common.CountDetail(len(podmanContainers)))
	if err != nil {
		logger.Println(logger.DEBUG, true, "PODMAN: "+err.Error())
	}

	sw := software2.Software{
		DEB:     deb,
		RPM:     rpm,
		Snap:    snaps,
		Flatpak: flatpaks,
		Legacy:  legacySW,
		Docker:  dockerContainers,
		Podman:  podmanContainers,
	}

	return &sw, nil
}
