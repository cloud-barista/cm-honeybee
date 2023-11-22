package infra

type Infra struct {
	Compute Compute `json:"compute"`
	Network Network `json:"network"`
	GPU     GPU     `json:"gpu"`
}

func GetInfraInfo() (*Infra, error) {
	var infra Infra
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
