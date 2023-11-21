package infra

import (
	"encoding/json"
)

type Infra struct {
	Compute Compute `json:"compute"`
	GPU     GPU     `json:"gpu"`
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
