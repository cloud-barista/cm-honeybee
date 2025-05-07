package infra

import (
	"github.com/cloud-barista/cm-honeybee/agent/gpu/drm"
	"github.com/cloud-barista/cm-honeybee/agent/gpu/nvidia"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/infra"
	"strings"
)

func GetGPUInfo() (infra.GPU, error) {
	var gpu infra.GPU

	nvStats, err := nvidia.QueryGPU()
	if err != nil {
		gpu.Errors = append(gpu.Errors, strings.ReplaceAll(strings.Trim(err.Error(), "\n"), "\n", " "))
	}
	gpu.NVIDIA = nvStats

	d, err := drm.GetDRMInfo()
	if err != nil {
		gpu.Errors = append(gpu.Errors, err.Error())
	}
	gpu.DRM = d

	return gpu, nil
}
