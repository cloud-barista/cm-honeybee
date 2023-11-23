package software

import (
	"github.com/shirou/gopsutil/v3/host"
)

type Software struct {
	DEB []DEB `json:"deb"`
	RPM []RPM `json:"rpm"`
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

	software := Software{
		DEB: deb,
		RPM: rpm,
	}

	return &software, nil
}
