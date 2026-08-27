package kubernetes

import (
	"errors"
	"sync"
	"time"

	"github.com/cloud-barista/cm-honeybee/agent/common"
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

	total := time.Now()
	defer func() {
		common.LogElapsed("kubernetes", "total", total, "")
	}()

	var i kubernetes.Kubernetes
	var err error

	if !isKubernetesReachable() {
		return &i, nil
	}

	start := time.Now()
	i.NodeCount, i.Nodes, err = GetNodeInfo()
	common.LogElapsed("kubernetes", "nodes", start, common.CountDetail(len(i.Nodes)))
	if err != nil {
		return nil, err
	}

	start = time.Now()
	i.Cluster, err = GetClusterInfo()
	common.LogElapsed("kubernetes", "cluster", start, "")
	if err != nil {
		return nil, err
	}

	start = time.Now()
	i.Workloads, err = GetWorkloadInfo()
	common.LogElapsed("kubernetes", "workloads", start, "")
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

	total := time.Now()
	defer func() {
		common.LogElapsed("helm", "total", total, "")
	}()

	var i kubernetes.Helm
	var err error

	if !isKubernetesReachable() {
		return &i, nil
	}

	start := time.Now()
	i.Repo, err = GetRepoInfo()
	common.LogElapsed("helm", "repo", start, common.CountDetail(len(i.Repo)))
	if err != nil {
		return nil, err
	}

	start = time.Now()
	i.Release, err = GetReleaseInfo()
	common.LogElapsed("helm", "release", start, common.CountDetail(len(i.Release)))
	if err != nil {
		return nil, err
	}

	return &i, nil
}
