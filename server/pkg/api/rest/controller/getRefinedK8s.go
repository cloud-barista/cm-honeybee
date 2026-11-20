package controller

import (
	"encoding/json"
	"sort"

	softwaremodel "github.com/cloud-barista/cm-grasshopper/smdl"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/kubernetes"
	serverCommon "github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/cloud-barista/cm-honeybee/server/dao"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"github.com/jollaman999/utils/logger"
)

// tryGetHelmInfo returns the collected Helm releases of the connection, or nil
// when the connection has none or its last import did not succeed. Mirrors
// tryGetKubernetesInfo: a connection without Helm data is the normal case, not
// an error.
func tryGetHelmInfo(connID string) *kubernetes.Helm {
	savedHelmInfo, err := dao.SavedHelmInfoGet(connID)
	if err != nil {
		return nil
	}
	// A failed import leaves either nothing or the releases of an earlier one,
	// which would be planned from as if they were current.
	if savedHelmInfo.Status != model.ConnectionInfoStatusSuccess {
		logger.Println(logger.WARN, false, "Skipping helm information whose last import did not succeed."+
			" (ConnectionID = "+connID+", Status = "+savedHelmInfo.Status+")")
		return nil
	}

	var helmInfo kubernetes.Helm
	if err := json.Unmarshal([]byte(savedHelmInfo.HelmData), &helmInfo); err != nil {
		logger.Println(logger.WARN, false, "Error occurred while parsing helm information."+
			" (ConnectionID = "+connID+")")
		return nil
	}

	return &helmInfo
}

// hasK8sClusterData reports whether a collected record came from a control
// plane node. Worker nodes have no kubeconfig, so their record is saved empty;
// selecting on record existence alone picks that empty one. The record the CSP
// collection stores for a managed cluster names the cluster and its version,
// so it reads as cluster data as well.
func hasK8sClusterData(info *kubernetes.Kubernetes) bool {
	if info == nil {
		return false
	}

	return info.Cluster.Version != "" || info.Cluster.Name != "" || len(info.Nodes) > 0
}

// isCSPManagedK8s reports whether the connection is a CSP k8s connection, whose
// kubernetes record the CSP collection stores. Such a record names the cluster
// and its version but holds no workloads: the cluster is read through the
// CSP's API, not through a kubeconfig.
func isCSPManagedK8s(sourceGroup *model.SourceGroup, connectionInfo *model.ConnectionInfo) bool {
	return serverCommon.IsCSPType(sourceGroup.Type) &&
		connectionInfo.ResourceType == serverCommon.ResourceTypeK8s
}

// connIsCSPManagedK8s is isCSPManagedK8s for a connection ID. A connection or
// source group that cannot be read is not taken for one, which leaves the
// caller's handling of the connection as it was.
func connIsCSPManagedK8s(connID string) bool {
	connectionInfo, err := dao.ConnectionInfoGet(connID)
	if err != nil {
		return false
	}
	sourceGroup, err := dao.SourceGroupGet(connectionInfo.SourceGroupID)
	if err != nil {
		return false
	}

	return isCSPManagedK8s(sourceGroup, connectionInfo)
}

// groupK8sCluster picks the Kubernetes cluster of the source group and reports
// whether it is a CSP-managed one. The cluster is collected once per group,
// from the connection of a control plane node whose kubeconfig enumerates the
// whole cluster. Every connection is a candidate, not only the node ones, so a
// cluster stored under a connection that is not a node is still found.
//
// A cluster the agent collected is taken before a CSP-managed one, whatever
// the order of the connections: only it lists the nodes, which give the node
// roles and the workers only the API sees. A managed cluster, which lists
// none, is taken only when the group has no other.
func groupK8sCluster(sourceGroup *model.SourceGroup, conns []model.ConnectionInfo) (*kubernetes.Kubernetes, bool) {
	_, info, managed := groupK8sClusterSource(sourceGroup, conns)

	return info, managed
}

// groupK8sClusterSource is groupK8sCluster that also names the connection the
// cluster was taken from, or "" when the group has none.
func groupK8sClusterSource(sourceGroup *model.SourceGroup, conns []model.ConnectionInfo) (string, *kubernetes.Kubernetes, bool) {
	var managed *kubernetes.Kubernetes
	managedConnID := ""

	for i := range conns {
		candidate := tryGetKubernetesInfo(conns[i].ID)
		if !hasK8sClusterData(candidate) {
			continue
		}
		if !isCSPManagedK8s(sourceGroup, &conns[i]) {
			return conns[i].ID, candidate, false
		}
		if managed == nil {
			managed = candidate
			managedConnID = conns[i].ID
		}
	}

	if managed != nil {
		return managedConnID, managed, true
	}

	return "", nil, false
}

// workloadItems reads one kind out of the collected workload map. The agent
// stores each kind as a list of objects whose keys it chose to preserve.
func workloadItems(workloads map[string]interface{}, kind string) []map[string]interface{} {
	raw, ok := workloads[kind].([]interface{})
	if !ok {
		return nil
	}

	items := make([]map[string]interface{}, 0, len(raw))
	for _, entry := range raw {
		if item, ok := entry.(map[string]interface{}); ok {
			items = append(items, item)
		}
	}

	return items
}

func workloadStrings(item map[string]interface{}, key string) []string {
	raw, ok := item[key].([]interface{})
	if !ok {
		return nil
	}

	values := make([]string, 0, len(raw))
	for _, entry := range raw {
		if value, ok := entry.(string); ok {
			values = append(values, value)
		}
	}

	return values
}

func workloadString(item map[string]interface{}, key string) string {
	value, _ := item[key].(string)

	return value
}

// workloadCountsByNamespace counts the collected objects per namespace. Velero
// migrates by namespace, so this is the unit the migration scope is chosen in.
//
// Every collected kind is counted, including the ones a migration is unlikely
// to want, and no namespace is held back: deciding what is worth moving is the
// caller's call, not honeybee's. A cluster-scoped kind carries no namespace and
// so cannot appear here at all.
func workloadCountsByNamespace(workloads map[string]interface{}) map[string]map[string]int {
	counts := map[string]map[string]int{}

	for kind := range workloads {
		for _, item := range workloadItems(workloads, kind) {
			namespace := workloadString(item, "Namespace")
			if namespace == "" {
				continue
			}
			if counts[namespace] == nil {
				counts[namespace] = map[string]int{}
			}
			counts[namespace][kind]++
		}
	}

	return counts
}

// workloadCountsClusterScoped counts the collected objects that belong to no
// namespace, which the per-namespace counts cannot reach.
func workloadCountsClusterScoped(workloads map[string]interface{}) map[string]int {
	counts := map[string]int{}

	for kind := range workloads {
		for _, item := range workloadItems(workloads, kind) {
			if workloadString(item, "Namespace") == "" {
				counts[kind]++
			}
		}
	}

	return counts
}

func k8sNamespaces(workloads map[string]interface{}) []string {
	names := make([]string, 0)
	for _, item := range workloadItems(workloads, "namespaces") {
		if name := workloadString(item, "Name"); name != "" {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	return names
}

func k8sStorageClasses(workloads map[string]interface{}) []softwaremodel.KubernetesStorageClass {
	classes := make([]softwaremodel.KubernetesStorageClass, 0)
	for _, item := range workloadItems(workloads, "storageclasses") {
		name := workloadString(item, "Name")
		if name == "" {
			continue
		}
		classes = append(classes, softwaremodel.KubernetesStorageClass{
			Name:        name,
			Provisioner: workloadString(item, "Provisioner"),
		})
	}

	return classes
}

// k8sPVs reports the cluster's PersistentVolumes. They are cluster-scoped, so
// they appear nowhere in the per-namespace counts and would otherwise be
// collected and then dropped.
func k8sPVs(workloads map[string]interface{}) []softwaremodel.KubernetesPersistentVolume {
	volumes := make([]softwaremodel.KubernetesPersistentVolume, 0)
	for _, item := range workloadItems(workloads, "persistentvolumes") {
		name := workloadString(item, "Name")
		if name == "" {
			continue
		}
		volumes = append(volumes, softwaremodel.KubernetesPersistentVolume{
			Name:           name,
			Capacity:       workloadString(item, "Capacity"),
			AccessModes:    workloadStrings(item, "AccessModes"),
			StorageClass:   workloadString(item, "StorageClass"),
			ReclaimPolicy:  workloadString(item, "ReclaimPolicy"),
			ClaimNamespace: workloadString(item, "ClaimNamespace"),
			ClaimName:      workloadString(item, "ClaimName"),
			Status:         workloadString(item, "Status"),
		})
	}

	return volumes
}

func k8sPVCs(workloads map[string]interface{}) []softwaremodel.KubernetesPersistentVolumeClaim {
	claims := make([]softwaremodel.KubernetesPersistentVolumeClaim, 0)
	for _, item := range workloadItems(workloads, "persistentvolumeclaims") {
		name := workloadString(item, "Name")
		if name == "" {
			continue
		}
		claims = append(claims, softwaremodel.KubernetesPersistentVolumeClaim{
			Namespace:    workloadString(item, "Namespace"),
			Name:         name,
			StorageClass: workloadString(item, "StorageClass"),
			AccessModes:  workloadStrings(item, "AccessModes"),
		})
	}

	return claims
}

func k8sHelmReleases(helm *kubernetes.Helm) []softwaremodel.KubernetesHelmRelease {
	if helm == nil {
		return nil
	}

	releases := make([]softwaremodel.KubernetesHelmRelease, 0, len(helm.Release))
	for _, release := range helm.Release {
		releases = append(releases, softwaremodel.KubernetesHelmRelease{
			Namespace:    release.Namespace,
			Name:         release.Name,
			Chart:        release.ChartName,
			ChartVersion: release.ChartVersion,
		})
	}

	return releases
}

// buildK8sSoftware turns a collected cluster into the refined software model
// entry that drives the Velero migration downstream.
//
// KubeConfig stays empty on purpose: the raw kubeconfig is not carried in the
// model. cm-grasshopper fetches it from honeybee by connection_id, where it is
// already stored encrypted.
func buildK8sSoftware(info *kubernetes.Kubernetes, helm *kubernetes.Helm) []softwaremodel.Kubernetes {
	if !hasK8sClusterData(info) {
		return nil
	}

	return []softwaremodel.Kubernetes{{
		Version:    info.Cluster.Version,
		KubeConfig: "",
		Resources: &softwaremodel.KubernetesResources{
			Namespaces:             k8sNamespaces(info.Workloads),
			Workloads:              workloadCountsByNamespace(info.Workloads),
			ClusterScopedWorkloads: workloadCountsClusterScoped(info.Workloads),
			StorageClasses:         k8sStorageClasses(info.Workloads),
			PersistentVolumes:      k8sPVs(info.Workloads),
			PersistentVolumeClaims: k8sPVCs(info.Workloads),
			HelmReleases:           k8sHelmReleases(helm),
		},
	}}
}
