package infra

import (
	"errors"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/infra"
	"sync"
)

var infraInfoLock sync.Mutex

func GetInfraInfo() (*infra.Infra, error) {
	if !infraInfoLock.TryLock() {
		return nil, errors.New("infra info collection is in progress")
	}
	defer func() {
		infraInfoLock.Unlock()
	}()

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

	haproxyInfo, err := GetHAProxyInfo()
	if err == nil {
		i.HAProxy = haproxyInfo
	}

	minioInfo, err := GetMinIOInfo()
	if err == nil {
		i.MinIO = minioInfo
	}

	return &i, nil
}
