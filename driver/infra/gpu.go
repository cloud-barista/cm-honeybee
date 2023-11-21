package infra

import "github.com/cloud-barista/cm-honeybee/gpu/nvidia"

type GPU struct {
	NVIDIA []nvidia.NVIDIA `json:"nvidia"`
}

func GetNVIDIAGpuInfo() (GPU, error) {
	nv, err := nvidia.NewNVReader()
	if err != nil {
		return GPU{}, err
	}

	nvStats, err := nv.GPUStats()
	if err != nil {
		return GPU{}, err
	}

	gpu := GPU{
		NVIDIA: nvStats,
	}

	return gpu, nil
}
