package controller

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/kubernetes"
)

// Collected workloads arrive grouped by kind, each item carrying only the keys
// the agent chose to preserve.
func k8sWorkloads() map[string]interface{} {
	return map[string]interface{}{
		"namespaces": []interface{}{
			map[string]interface{}{"Name": "default"},
			map[string]interface{}{"Name": "shop"},
			map[string]interface{}{"Name": "monitoring"},
		},
		"deployments": []interface{}{
			map[string]interface{}{"Namespace": "shop", "Name": "web"},
			map[string]interface{}{"Namespace": "shop", "Name": "api"},
			map[string]interface{}{"Namespace": "monitoring", "Name": "grafana"},
		},
		"statefulsets": []interface{}{
			map[string]interface{}{"Namespace": "shop", "Name": "mysql"},
		},
		"replicasets": []interface{}{
			map[string]interface{}{"Namespace": "shop", "Name": "web-7d9f"},
			map[string]interface{}{"Namespace": "shop", "Name": "api-5c8b"},
		},
		"pods": []interface{}{
			map[string]interface{}{"Namespace": "shop", "Name": "web-7d9f-xk2", "Status": "Running"},
			map[string]interface{}{"Namespace": "kube-system", "Name": "coredns-abc", "Status": "Running"},
		},
		"services": []interface{}{
			map[string]interface{}{"Namespace": "shop", "Name": "web-svc", "Type": "ClusterIP"},
		},
		"storageclasses": []interface{}{
			map[string]interface{}{"Name": "local-path", "Provisioner": "rancher.io/local-path"},
		},
		"persistentvolumeclaims": []interface{}{
			map[string]interface{}{
				"Namespace": "shop", "Name": "mysql-data", "StorageClass": "local-path",
				"AccessModes": []interface{}{"ReadWriteOnce"},
			},
		},
		"persistentvolumes": []interface{}{
			map[string]interface{}{
				"Name": "pvc-8bb703c5", "Capacity": "1Gi", "StorageClass": "local-path",
				"AccessModes":   []interface{}{"ReadWriteOnce"},
				"ReclaimPolicy": "Delete", "ClaimNamespace": "shop", "ClaimName": "mysql-data",
				"Status": "Bound",
			},
		},
	}
}

// Velero migrates a namespace at a time, so that is the unit to count in. Every
// collected kind is counted and no namespace is held back, system ones
// included: what is worth moving is the caller's call. A cluster-scoped kind
// such as StorageClass has no namespace to count under.
func TestWorkloadCountsByNamespace(t *testing.T) {
	got := workloadCountsByNamespace(k8sWorkloads())

	want := map[string]map[string]int{
		"shop": {"deployments": 2, "statefulsets": 1, "services": 1,
			"persistentvolumeclaims": 1, "replicasets": 2, "pods": 1},
		"monitoring":  {"deployments": 1},
		"kube-system": {"pods": 1},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("workloadCountsByNamespace() =\n %#v\nwant\n %#v", got, want)
	}
}

// A kind with no namespace cannot appear in the per-namespace counts, so it is
// counted here instead rather than being collected and dropped.
func TestWorkloadCountsClusterScoped(t *testing.T) {
	got := workloadCountsClusterScoped(k8sWorkloads())

	want := map[string]int{"namespaces": 3, "storageclasses": 1, "persistentvolumes": 1}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("workloadCountsClusterScoped() = %#v, want %#v", got, want)
	}
}

func TestWorkloadCountsByNamespace_Empty(t *testing.T) {
	if got := workloadCountsByNamespace(nil); len(got) != 0 {
		t.Errorf("empty input must give an empty result: %#v", got)
	}
}

// A worker node saves an empty record too, so selecting on record existence
// alone loads an empty cluster over the control plane's.
func TestHasK8sClusterData(t *testing.T) {
	cases := []struct {
		name string
		info *kubernetes.Kubernetes
		want bool
	}{
		{"nil", nil, false},
		{"empty record", &kubernetes.Kubernetes{}, false},
		{"version alone is enough", &kubernetes.Kubernetes{Cluster: kubernetes.Cluster{Version: "1.32.3"}}, true},
		{"nodes alone are enough", &kubernetes.Kubernetes{Nodes: []kubernetes.Node{{Type: kubernetes.NodeTypeControlPlane}}}, true},
	}
	for _, c := range cases {
		if got := hasK8sClusterData(c.info); got != c.want {
			t.Errorf("hasK8sClusterData(%s) = %v, want %v", c.name, got, c.want)
		}
	}
}

// kube_config stays empty: the raw kubeconfig is not carried in the model,
// cm-grasshopper fetches it from honeybee by connection_id.
func TestBuildK8sSoftware(t *testing.T) {
	k8sInfo := &kubernetes.Kubernetes{
		Cluster:   kubernetes.Cluster{Version: "1.32.3"},
		Workloads: k8sWorkloads(),
	}
	helm := &kubernetes.Helm{Release: []kubernetes.Release{
		{Name: "prometheus", Namespace: "monitoring", ChartName: "kube-prometheus-stack", ChartVersion: "58.1.0"},
	}}

	got := buildK8sSoftware(k8sInfo, helm)
	if len(got) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(got))
	}

	k := got[0]
	if k.Version != "1.32.3" {
		t.Errorf("version = %q, want %q", k.Version, "1.32.3")
	}
	if k.KubeConfig != "" {
		t.Errorf("kube_config must stay empty: %q", k.KubeConfig)
	}
	if k.Resources == nil {
		t.Fatal("resources must be set")
	}

	if want := []string{"default", "monitoring", "shop"}; !reflect.DeepEqual(k.Resources.Namespaces, want) {
		t.Errorf("namespaces = %#v, want %#v", k.Resources.Namespaces, want)
	}

	sc := k.Resources.StorageClasses
	if len(sc) != 1 || sc[0].Name != "local-path" || sc[0].Provisioner != "rancher.io/local-path" {
		t.Errorf("storageClasses = %#v", sc)
	}

	pvc := k.Resources.PersistentVolumeClaims
	if len(pvc) != 1 || pvc[0].Namespace != "shop" || pvc[0].StorageClass != "local-path" ||
		len(pvc[0].AccessModes) != 1 || pvc[0].AccessModes[0] != "ReadWriteOnce" {
		t.Errorf("persistentVolumeClaims = %#v", pvc)
	}

	pv := k.Resources.PersistentVolumes
	if len(pv) != 1 || pv[0].Name != "pvc-8bb703c5" || pv[0].Capacity != "1Gi" ||
		len(pv[0].AccessModes) != 1 || pv[0].AccessModes[0] != "ReadWriteOnce" ||
		pv[0].ClaimNamespace != "shop" || pv[0].ClaimName != "mysql-data" || pv[0].ReclaimPolicy != "Delete" {
		t.Errorf("persistentVolumes = %#v", pv)
	}

	rel := k.Resources.HelmReleases
	if len(rel) != 1 || rel[0].Chart != "kube-prometheus-stack" || rel[0].ChartVersion != "58.1.0" {
		t.Errorf("helmReleases = %#v", rel)
	}
}

// Without cluster data no entry is produced at all, so a worker node's
// software/refined does not carry an empty kubernetes item.
func TestBuildK8sSoftware_NoCluster(t *testing.T) {
	if got := buildK8sSoftware(nil, nil); got != nil {
		t.Errorf("nil input must give nil: %#v", got)
	}
	if got := buildK8sSoftware(&kubernetes.Kubernetes{}, nil); got != nil {
		t.Errorf("an empty record must give nil: %#v", got)
	}
}

// The JSON is the contract with cm-grasshopper, so pin the wire shape rather
// than only the Go fields.
func TestBuildK8sSoftware_JSONShape(t *testing.T) {
	k8sInfo := &kubernetes.Kubernetes{
		Cluster:   kubernetes.Cluster{Version: "1.32.3"},
		Workloads: k8sWorkloads(),
	}
	helm := &kubernetes.Helm{Release: []kubernetes.Release{
		{Name: "prometheus", Namespace: "monitoring", ChartName: "kube-prometheus-stack", ChartVersion: "58.1.0"},
	}}

	encoded, err := json.Marshal(buildK8sSoftware(k8sInfo, helm)[0])
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	want := `{"version":"1.32.3","kube_config":"","resources":{` +
		`"namespaces":["default","monitoring","shop"],` +
		`"workloads":{"kube-system":{"pods":1},"monitoring":{"deployments":1},` +
		`"shop":{"deployments":2,"persistentvolumeclaims":1,"pods":1,"replicasets":2,"services":1,"statefulsets":1}},` +
		`"clusterScopedWorkloads":{"namespaces":3,"persistentvolumes":1,"storageclasses":1},` +
		`"storageClasses":[{"name":"local-path","provisioner":"rancher.io/local-path"}],` +
		`"persistentVolumes":[{"name":"pvc-8bb703c5","capacity":"1Gi","accessModes":["ReadWriteOnce"],` +
		`"storageClass":"local-path",` +
		`"reclaimPolicy":"Delete","claimNamespace":"shop","claimName":"mysql-data","status":"Bound"}],` +
		`"persistentVolumeClaims":[{"namespace":"shop","name":"mysql-data","storageClass":"local-path",` +
		`"accessModes":["ReadWriteOnce"]}],` +
		`"helmReleases":[{"namespace":"monitoring","name":"prometheus","chart":"kube-prometheus-stack","chartVersion":"58.1.0"}]}}`

	if string(encoded) != want {
		t.Errorf("JSON =\n%s\nwant\n%s", encoded, want)
	}
}
