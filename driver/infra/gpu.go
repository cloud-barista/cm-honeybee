package infra

import "github.com/cloud-barista/cm-honeybee/gpu/nvidia"

func GetNVIDIAGpuInfo() (nvidia.AllGPUStats, error) {
	var stats nvidia.AllGPUStats

	nv, err := nvidia.NewNVReader()
	if err != nil {
		return stats, err
	}

	return nv.GPUStats()
}
