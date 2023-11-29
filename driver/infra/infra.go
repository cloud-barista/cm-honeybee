package infra

import "github.com/cloud-barista/cm-honeybee/model/infra"

func GetInfraInfo() (*infra.Infra, error) {
	var infra infra.Infra
	var err error

	infra.Compute, err = GetComputeInfo()
	if err != nil {
		return nil, err
	}

	infra.Network, err = GetNetworkInfo()
	if err != nil {
		return nil, err
	}

	infra.GPU, err = GetGPUInfo()
	if err != nil {
		return nil, err
	}

	return &infra, nil
}
