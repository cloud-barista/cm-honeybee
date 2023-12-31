package infra

import (
	"github.com/cloud-barista/cm-honeybee/pkg/api/rest/model/infra"
)

func GetInfraInfo() (*infra.Infra, error) {
	var i infra.Infra
	var err error

	i.Compute, err = GetComputeInfo()
	if err != nil {
		return nil, err
	}

	i.Network, err = GetNetworkInfo()
	if err != nil {
		return nil, err
	}

	i.GPU, err = GetGPUInfo()
	if err != nil {
		return nil, err
	}

	return &i, nil
}
