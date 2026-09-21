package infra

import (
	"errors"
	"sync"
	"time"

	"github.com/cloud-barista/cm-honeybee/agent/common"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/infra"
)

var infraInfoLock sync.Mutex

const (
	gibibyte     = 1024 * 1024 * 1024
	halfGibibyte = gibibyte / 2
)

// bytesToGiB rounds a byte reading to the whole gibibytes the disk fields
// carry. Truncating instead reports anything under a gibibyte as 0, and a
// disk sized in round decimal gigabytes always falls short of the binary
// figure it is labelled with (a "100 GB" volume is 93 GiB).
func bytesToGiB(b uint64) uint {
	return uint((b + halfGibibyte) / gibibyte)
}

// bytesToGiBCapacity is bytesToGiB for a total-capacity field. A 0 there is
// not a small disk, it is a missing one: it reads downstream as a node with
// no root disk at all, so a volume that exists reports at least 1.
func bytesToGiBCapacity(b uint64) uint {
	if gib := bytesToGiB(b); gib > 0 {
		return gib
	}

	if b > 0 {
		return 1
	}

	return 0
}

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
