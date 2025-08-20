package kubernetes

import (
	"errors"
	"sync"

	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/kubernetes"
)

var kubernetesInfoLock sync.Mutex
var helmInfoLock sync.Mutex

func GetKubernetesInfo() (*kubernetes.Kubernetes, error) {
	if !kubernetesInfoLock.TryLock() {
		return nil, errors.New("kubernetes info collection is in progress")
	}
	defer func() {
		kubernetesInfoLock.Unlock()
	}()

	var i kubernetes.Kubernetes
	var err error

	i.NodeCount, i.Nodes, err = GetNodeInfo()
	if err != nil {
		return nil, err
	}

	i.Workloads, err = GetWorkloadInfo()
	if err != nil {
		return nil, err
	}

	return &i, nil
}

func GetHelmInfo() (*kubernetes.Helm, error) {
	if !helmInfoLock.TryLock() {
		return nil, errors.New("helm info collection is in progress")
	}
	defer func() {
		helmInfoLock.Unlock()
	}()

	var i kubernetes.Helm
	var err error

	i.Repo, err = GetRepoInfo()
	if err != nil {
		return nil, err
	}

	i.Release, err = GetReleaseInfo()
	if err != nil {
		return nil, err
	}

	return &i, nil
}
