// Getting DRM information for Linux & Unix like systems

//go:build !windows

package drm

import (
	"errors"
	"github.com/NeowayLabs/drm"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/infra"
	"github.com/jollaman999/utils/logger"
	"strconv"
)

func GetDRMInfo() ([]infra.DRM, error) {
	versions := drm.ListDevices()
	if len(versions) == 0 {
		errMsg := "drm: DRM is not available"
		logger.Println(logger.DEBUG, true, errMsg)

		return []infra.DRM{}, errors.New(errMsg)
	}

	var d []infra.DRM
	for _, v := range versions {
		d = append(d, infra.DRM{
			DriverName: v.Name,
			DriverVersion: strconv.Itoa(int(v.Major)) + "." +
				strconv.Itoa(int(v.Minor)) + "." + strconv.Itoa(int(v.Patch)),
			DriverDate:        v.Date,
			DriverDescription: v.Desc,
		})
	}

	return d, nil
}
