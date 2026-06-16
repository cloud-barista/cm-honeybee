package kubernetes

import (
	"errors"
	"sync"

	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/kubernetes"

	"github.com/jollaman999/utils/logger"
)

var kubernetesInfoLock sync.Mutex
var helmInfoLock sync.Mutex

// isKubernetesReachable reports whether this host is a reachable Kubernetes
// control plane. Collection is best-effort: hosts that are not control plane
// nodes (plain VMs, worker nodes without admin.conf) or whose API server is
// unreachable are skipped rather than failing the whole collection. Cluster
// nodes that cannot collect locally are still covered by the control plane's
// node enumeration.
func isKubernetesReachable() bool {
	clientset, err := GetKubernetesClientSet()
	if err != nil {
		logger.Println(logger.INFO, true, "Kubernetes: no usable kubeconfig, skipping collection: "+err.Error())
		return false
	}

	if _, err := clientset.Discovery().ServerVersion(); err != nil {
		logger.Println(logger.WARN, true, "Kubernetes: API server unreachable, skipping collection: "+err.Error())
		return false
	}

	return true
}

func GetKubernetesInfo() (*kubernetes.Kubernetes, error) {
	if !kubernetesInfoLock.TryLock() {
		return nil, errors.New("kubernetes info collection is in progress")
	}
	defer func() {
		kubernetesInfoLock.Unlock()
	}()

	var i kubernetes.Kubernetes
	var err error

	if !isKubernetesReachable() {
		return &i, nil
	}

	i.NodeCount, i.Nodes, err = GetNodeInfo()
	if err != nil {
		return nil, err
	}

	i.Cluster, err = GetClusterInfo()
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

	if !isKubernetesReachable() {
		return &i, nil
	}

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
