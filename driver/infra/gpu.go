package infra

import (
	"github.com/cloud-barista/cm-honeybee/gpu/drm"
	"github.com/cloud-barista/cm-honeybee/gpu/nvidia"
	"github.com/cloud-barista/cm-honeybee/pkg/api/rest/model/onprem/infra"
)

func GetGPUInfo() (infra.GPU, error) {
	nvStats, err := nvidia.QueryGPU()
	if err != nil {
		return infra.GPU{}, err
	}

	d, err := drm.GetDRMInfo()
	if err != nil {
		return infra.GPU{}, err
	}

	gpu := infra.GPU{
		NVIDIA: nvStats,
		DRM:    d,
	}

	return gpu, nil
}
