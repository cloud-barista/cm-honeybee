package infra

import (
	"encoding/json"
	"github.com/cloud-barista/cm-honeybee/gpu/nvidia"
)

type Infra struct {
	Compute Compute            `json:"compute"`
	GPU     nvidia.AllGPUStats `json:"gpu"`
}

func GetInfraInfo() (string, error) {
	var infra Infra
	var err error

	infra.Compute, err = GetComputeInfo()
	if err != nil {
		return "", err
	}

	infra.GPU, err = GetNVIDIAGpuInfo()
	if err != nil {
		return "", err
	}

	data, err := json.MarshalIndent(&infra, "", " ")
	if err != nil {
		return "", err
	}

	return string(data), nil
}
