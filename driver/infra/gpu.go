package infra

import "github.com/cloud-barista/cm-honeybee/gpu/nvidia"

type GPU struct {
	NVIDIA []nvidia.NVIDIA `json:"nvidia"`
}

func GetNVIDIAGpuInfo() (GPU, error) {
	nvStats, err := nvidia.QueryGPU()
	if err != nil {
		return GPU{}, err
	}

	gpu := GPU{
		NVIDIA: nvStats,
	}

	return gpu, nil
}
