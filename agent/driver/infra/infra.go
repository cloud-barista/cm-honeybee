package infra

import (
	"errors"
	"sync"
	"time"

	"github.com/cloud-barista/cm-honeybee/agent/common"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/infra"
)

var infraInfoLock sync.Mutex

func GetInfraInfo() (*infra.Infra, error) {
	if !infraInfoLock.TryLock() {
		return nil, errors.New("infra info collection is in progress")
	}
	defer func() {
		infraInfoLock.Unlock()
	}()

	total := time.Now()
	defer func() {
		common.LogElapsed("infra", "total", total, "")
	}()

	var i infra.Infra
	var err error

	start := time.Now()
	i.Compute, err = GetComputeInfo()
	common.LogElapsed("infra", "compute", start, "")
	if err != nil {
		return nil, err
	}

	start = time.Now()
	i.Network, err = GetNetworkInfo()
	common.LogElapsed("infra", "network", start, "")
	if err != nil {
		return nil, err
	}

	start = time.Now()
	i.GPU, err = GetGPUInfo()
	common.LogElapsed("infra", "gpu", start, "")
	if err != nil {
		return nil, err
	}

	start = time.Now()
	haproxyInfo, err := GetHAProxyInfo()
	common.LogElapsed("infra", "haproxy", start, "")
	if err == nil {
		i.HAProxy = haproxyInfo
	}

	start = time.Now()
	minioInfo, err := GetMinIOInfo()
	common.LogElapsed("infra", "minio", start, "")
	if err == nil {
		i.MinIO = minioInfo
	}

	return &i, nil
}
