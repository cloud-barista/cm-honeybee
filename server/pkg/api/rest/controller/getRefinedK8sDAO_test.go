package controller

import (
	"encoding/json"
	"testing"

	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/kubernetes"
	"github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/cloud-barista/cm-honeybee/server/db"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
)

// openTestDB points the module's own sqlite at a throwaway directory. The driver
// is pure Go, so this is the real storage path rather than a stand-in.
func openTestDB(t *testing.T) {
	t.Helper()

	common.RootPath = t.TempDir()
	if err := db.Open(); err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(db.Close)
}

func saveKubernetes(t *testing.T, connID string, info kubernetes.Kubernetes) {
	t.Helper()

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.DB.Create(&model.SavedKubernetesInfo{
		ConnectionID: connID, KubernetesData: string(data), Status: "success",
	}).Error; err != nil {
		t.Fatal(err)
	}
}

func saveHelm(t *testing.T, connID string, helm kubernetes.Helm) {
	t.Helper()

	data, err := json.Marshal(helm)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.DB.Create(&model.SavedHelmInfo{
		ConnectionID: connID, HelmData: string(data), Status: "success",
	}).Error; err != nil {
		t.Fatal(err)
	}
}

// A connection with no record is the normal case, not an error: a worker node
// never stores Helm data and must not fail the refinement.
func TestTryGetHelmInfo_MissingConnection(t *testing.T) {
	openTestDB(t)

	if got := tryGetHelmInfo("no-such-connection"); got != nil {
		t.Errorf("a connection without helm data must give nil, got %#v", got)
	}
}

func TestTryGetHelmInfo_RoundTrip(t *testing.T) {
	openTestDB(t)
	saveHelm(t, "conn-cp", kubernetes.Helm{Release: []kubernetes.Release{
		{Name: "prometheus", Namespace: "monitoring", ChartName: "kube-prometheus-stack", ChartVersion: "58.1.0"},
	}})

	got := tryGetHelmInfo("conn-cp")
	if got == nil {
		t.Fatal("stored helm data must come back")
	}
	if len(got.Release) != 1 || got.Release[0].ChartName != "kube-prometheus-stack" {
		t.Errorf("release = %#v", got.Release)
	}
}

// Stored data that no longer parses is logged and skipped, not returned half
// built and not propagated as a failure.
func TestTryGetHelmInfo_CorruptData(t *testing.T) {
	openTestDB(t)
	if err := db.DB.Create(&model.SavedHelmInfo{
		ConnectionID: "conn-broken", HelmData: "{not json", Status: "success",
	}).Error; err != nil {
		t.Fatal(err)
	}

	if got := tryGetHelmInfo("conn-broken"); got != nil {
		t.Errorf("unparsable data must give nil, got %#v", got)
	}
}

func TestTryGetKubernetesInfo_RoundTrip(t *testing.T) {
	openTestDB(t)
	saveKubernetes(t, "conn-cp", kubernetes.Kubernetes{
		Cluster:   kubernetes.Cluster{Version: "1.32.3", Name: "kubernetes"},
		Workloads: k8sWorkloads(),
	})

	got := tryGetKubernetesInfo("conn-cp")
	if got == nil {
		t.Fatal("stored kubernetes data must come back")
	}
	if !hasK8sClusterData(got) {
		t.Error("a stored control plane record must read as cluster data")
	}
	if len(buildK8sSoftware(got, tryGetHelmInfo("conn-cp"))) != 1 {
		t.Error("the refined entry must be built from what the DAO returned")
	}
}

// What a worker node stores: the agent finds no kubeconfig, returns an empty
// body, and the server saves it as a success.
func TestTryGetKubernetesInfo_WorkerRecordIsEmpty(t *testing.T) {
	openTestDB(t)
	saveKubernetes(t, "conn-worker", kubernetes.Kubernetes{})

	got := tryGetKubernetesInfo("conn-worker")
	if got == nil {
		t.Fatal("the record exists, so it must be returned")
	}
	if hasK8sClusterData(got) {
		t.Error("an empty worker record must not read as cluster data")
	}
	if buildK8sSoftware(got, nil) != nil {
		t.Error("a worker connection must contribute no kubernetes entry")
	}
}
