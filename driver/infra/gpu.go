package infra

import (
	"github.com/cloud-barista/cm-honeybee/gpu/drm"
	"github.com/cloud-barista/cm-honeybee/gpu/nvidia"
)

type GPU struct {
	NVIDIA []nvidia.NVIDIA `json:"nvidia"`
	DRM    drm.DRM         `json:"drm"`
}

func GetGPUInfo() (GPU, error) {
	nvStats, err := nvidia.QueryGPU()
	if err != nil {
		return GPU{}, err
	}

	d, err := drm.GetDRMInfo()
	if err != nil {
		return GPU{}, err
	}

	gpu := GPU{
		NVIDIA: nvStats,
		DRM:    d,
	}

	return gpu, nil
}
