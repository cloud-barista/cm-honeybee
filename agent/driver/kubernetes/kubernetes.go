package infra

import (
	"errors"
	"sync"

	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/kubernetes"
)

var kubernetesInfoLock sync.Mutex

func GetKubernetesInfo() (*kubernetes.Kubernetes, error) {
	if !kubernetesInfoLock.TryLock() {
		return nil, errors.New("kubernetes info collection is in progress")
	}
	defer func() {
		kubernetesInfoLock.Unlock()
	}()

	var i kubernetes.Kubernetes
	var err error

	i.Nodes, err = GetNodeInfo()
	if err != nil {
		return nil, err
	}

	i.Workloads, err = GetWorkloadInfo()
	if err != nil {
		return nil, err
	}

	return &i, nil
}
