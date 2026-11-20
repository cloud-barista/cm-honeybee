package controller

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"

	inframodel "github.com/cloud-barista/cm-beetle/imdl/on-premise-model"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/kubernetes"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/software"
	"github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/cloud-barista/cm-honeybee/server/db"
	"github.com/cloud-barista/cm-honeybee/server/lib/spider"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
)

// managedCluster is a cluster the way cb-spider reports one it did not
// create: no NameId, two node groups, and a CreatedTime the driver left zero.
func managedCluster() *spider.ClusterInfo {
	return &spider.ClusterInfo{
		IId:         spider.IID{NameId: "", SystemId: "cls-0a1b2c3d"},
		Version:     "1.31",
		Status:      "Active",
		CreatedTime: "0001-01-01T00:00:00Z",
		NodeGroupList: []spider.NodeGroupInfo{
			{
				IId:             spider.IID{NameId: "", SystemId: "np-general"},
				ImageIID:        spider.IID{NameId: "", SystemId: "img-ubuntu-2204"},
				VMSpecName:      "s2.medium",
				RootDiskType:    "CLOUD_SSD",
				RootDiskSize:    "50",
				OnAutoScaling:   true,
				DesiredNodeSize: 2,
				MinNodeSize:     1,
				MaxNodeSize:     4,
				Status:          "Active",
			},
			{
				IId:             spider.IID{NameId: "batch", SystemId: "np-batch"},
				ImageIID:        spider.IID{NameId: "NA", SystemId: ""},
				VMSpecName:      "NA",
				DesiredNodeSize: 1,
				MinNodeSize:     1,
				MaxNodeSize:     1,
				Status:          "Creating",
			},
		},
	}
}

func saveCSPCluster(t *testing.T, connID string, cl *spider.ClusterInfo) {
	t.Helper()

	if err := upsertSavedK8s(connID, clusterInfoToK8s(cl)); err != nil {
		t.Fatal(err)
	}
}

func TestClusterInfoToK8s(t *testing.T) {
	got := clusterInfoToK8s(managedCluster())

	// No NameId: the SystemId is the only name the cluster has.
	if got.Cluster.Name != "cls-0a1b2c3d" || got.Cluster.Version != "1.31" {
		t.Errorf("cluster = %+v, want name cls-0a1b2c3d, version 1.31", got.Cluster)
	}
	if got.Cluster.PodCIDR != "" || got.Cluster.ServiceCIDR != "" {
		t.Errorf("cidrs = %q/%q, want empty", got.Cluster.PodCIDR, got.Cluster.ServiceCIDR)
	}
	if want := (kubernetes.NodeCount{Total: 3, Worker: 3}); got.NodeCount != want {
		t.Errorf("node count = %+v, want %+v", got.NodeCount, want)
	}
	if got.Nodes != nil {
		t.Errorf("nodes = %+v, want none", got.Nodes)
	}

	if got.CSP == nil {
		t.Fatal("csp section is missing")
	}
	if got.CSP.SystemID != "cls-0a1b2c3d" || got.CSP.Status != "Active" {
		t.Errorf("csp = %+v", got.CSP)
	}
	if got.CSP.CreatedTime != "" {
		t.Errorf("created time = %q, want the zero time dropped", got.CSP.CreatedTime)
	}

	want := []model.CSPKubernetesNodeGroup{
		{
			Name: "np-general", VMSpec: "s2.medium", Image: "img-ubuntu-2204",
			RootDiskType: "CLOUD_SSD", RootDiskSize: "50", OnAutoScaling: true,
			DesiredNodeSize: 2, MinNodeSize: 1, MaxNodeSize: 4, Status: "Active",
		},
		{
			Name: "batch", DesiredNodeSize: 1, MinNodeSize: 1, MaxNodeSize: 1, Status: "Creating",
		},
	}
	if !reflect.DeepEqual(got.CSP.NodeGroups, want) {
		t.Errorf("node groups =\n%+v\nwant\n%+v", got.CSP.NodeGroups, want)
	}
}

func TestClusterInfoToK8s_NameAndCreatedTime(t *testing.T) {
	cl := managedCluster()
	cl.IId.NameId = "shop-cluster"
	cl.CreatedTime = "2026-11-01T09:30:00Z"
	cl.NodeGroupList = nil

	got := clusterInfoToK8s(cl)
	if got.Cluster.Name != "shop-cluster" {
		t.Errorf("name = %q, want the NameId", got.Cluster.Name)
	}
	if got.CSP.SystemID != "cls-0a1b2c3d" {
		t.Errorf("system id = %q", got.CSP.SystemID)
	}
	if got.CSP.CreatedTime != "2026-11-01T09:30:00Z" {
		t.Errorf("created time = %q", got.CSP.CreatedTime)
	}

	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"node_groups":[]`) {
		t.Errorf("a cluster without node groups must store an empty list: %s", raw)
	}
}

// The stored record keeps the keys of the agent's shape, so the readers that
// parse it as kubernetes.Kubernetes find the cluster in it.
func TestClusterInfoToK8s_ReadsAsClusterData(t *testing.T) {
	raw, err := json.Marshal(clusterInfoToK8s(managedCluster()))
	if err != nil {
		t.Fatal(err)
	}

	var plain kubernetes.Kubernetes
	if err := json.Unmarshal(raw, &plain); err != nil {
		t.Fatal(err)
	}
	if !hasK8sClusterData(&plain) {
		t.Errorf("a stored managed cluster must read as cluster data: %s", raw)
	}
	if plain.Cluster.Name != "cls-0a1b2c3d" || plain.NodeCount.Worker != 3 {
		t.Errorf("parsed = %+v", plain)
	}
}

func refinedInfraBody(t *testing.T, body []byte) inframodel.OnpremiseInfraModel {
	t.Helper()

	var got inframodel.OnpremiseInfraModel
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatal(err)
	}
	return got
}

// A CSP k8s connection is a cluster and no machine: its refined infra holds
// the cluster and no node, where it used to hold one empty node.
func TestGetInfraInfoRefined_CSPK8s(t *testing.T) {
	openTestDB(t)
	createGroup(t, cspGroup, mixedCSPConns())
	saveInfra(t, "k8s", "success", nil, "")
	saveCSPCluster(t, "k8s", managedCluster())

	rec := connRequest(t, http.MethodGet, cspGroup.ID, "k8s", GetInfraInfoRefined)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"nodes": []`) {
		t.Errorf("body = %s, want an empty node list", rec.Body.String())
	}
	got := refinedInfraBody(t, rec.Body.Bytes())
	want := &inframodel.K8sClusterProperty{Name: "cls-0a1b2c3d", Version: "1.31"}
	if !reflect.DeepEqual(got.OnpremiseInfraModel.K8sCluster, want) {
		t.Errorf("k8sCluster = %+v, want %+v", got.OnpremiseInfraModel.K8sCluster, want)
	}
}

// The other CSP resources that are not VMs make no node either, whether or not
// their import left an infra record, while a CSP VM still makes its node.
func TestGetInfraInfoRefined_CSPNonVMHasNoNode(t *testing.T) {
	openTestDB(t)
	createGroup(t, cspGroup, mixedCSPConns())
	saveInfra(t, "bucket", "success", nil, "")
	saveInfra(t, "rdbms", "success", nil, "")
	saveInfra(t, "vm-ip", "success", hostInfra("node-ip"), "{}")

	for _, id := range []string{"bucket", "rdbms", "nlb"} {
		rec := connRequest(t, http.MethodGet, cspGroup.ID, id, GetInfraInfoRefined)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: status = %d, body = %s", id, rec.Code, rec.Body.String())
		}
		got := refinedInfraBody(t, rec.Body.Bytes())
		if n := len(got.OnpremiseInfraModel.Nodes); n != 0 {
			t.Errorf("%s: nodes = %d, want 0", id, n)
		}
		if got.OnpremiseInfraModel.K8sCluster != nil {
			t.Errorf("%s: k8sCluster = %+v, want none", id, got.OnpremiseInfraModel.K8sCluster)
		}
	}

	rec := connRequest(t, http.MethodGet, cspGroup.ID, "vm-ip", GetInfraInfoRefined)
	if rec.Code != http.StatusOK {
		t.Fatalf("vm-ip: status = %d, body = %s", rec.Code, rec.Body.String())
	}
	nodes := refinedInfraBody(t, rec.Body.Bytes()).OnpremiseInfraModel.Nodes
	if len(nodes) != 1 || nodes[0].Hostname != "node-ip" {
		t.Errorf("vm-ip: nodes = %+v, want node-ip", nodes)
	}
}

// The group refined infra finds the cluster under the CSP k8s connection,
// which is not a node, and leaves the VM nodes' role alone: a cluster that
// lists no nodes cannot tell which hosts are outside it.
func TestGetInfraInfoSourceGroupRefined_CSPK8s(t *testing.T) {
	openTestDB(t)
	createGroup(t, cspGroup, mixedCSPConns())
	saveInfra(t, "vm-ip", "success", hostInfra("node-ip"), "{}")
	saveInfra(t, "k8s", "success", nil, "")
	saveCSPCluster(t, "k8s", managedCluster())

	rec := groupRequest(t, cspGroup.ID, GetInfraInfoSourceGroupRefined)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	got := refinedInfraBody(t, rec.Body.Bytes())

	want := &inframodel.K8sClusterProperty{Name: "cls-0a1b2c3d", Version: "1.31"}
	if !reflect.DeepEqual(got.OnpremiseInfraModel.K8sCluster, want) {
		t.Errorf("k8sCluster = %+v, want %+v", got.OnpremiseInfraModel.K8sCluster, want)
	}
	nodes := got.OnpremiseInfraModel.Nodes
	if len(nodes) != 1 || nodes[0].Hostname != "node-ip" {
		t.Fatalf("nodes = %+v, want only node-ip", nodes)
	}
	if nodes[0].Role != "" {
		t.Errorf("role = %q, want it left empty", nodes[0].Role)
	}
}

// A failed collection of the cluster leaves it out of the group refined infra.
func TestGetInfraInfoSourceGroupRefined_CSPK8sFailed(t *testing.T) {
	openTestDB(t)
	createGroup(t, cspGroup, mixedCSPConns())
	saveCSPCluster(t, "k8s", managedCluster())
	if err := markCSPResourceFailed(&model.ConnectionInfo{ID: "k8s", ResourceType: common.ResourceTypeK8s}); err != nil {
		t.Fatal(err)
	}

	rec := groupRequest(t, cspGroup.ID, GetInfraInfoSourceGroupRefined)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if k := refinedInfraBody(t, rec.Body.Bytes()).OnpremiseInfraModel.K8sCluster; k != nil {
		t.Errorf("k8sCluster = %+v, want none", k)
	}
}

// An on-prem record that names the cluster and lists no nodes still marks the
// hosts standalone, as it did before the managed cluster was reflected.
func TestGetInfraInfoSourceGroupRefined_OnpremClusterWithoutNodes(t *testing.T) {
	openTestDB(t)
	createGroup(t, onpremGroup, []model.ConnectionInfo{testConn("onprem-cp", onpremGroup, "", "192.0.2.10")})
	saveInfra(t, "onprem-cp", "success", hostInfra("cp-1"), "")
	saveKubernetes(t, "onprem-cp", kubernetes.Kubernetes{Cluster: kubernetes.Cluster{Name: "kubernetes", Version: "1.30.2"}})

	rec := groupRequest(t, onpremGroup.ID, GetInfraInfoSourceGroupRefined)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	got := refinedInfraBody(t, rec.Body.Bytes()).OnpremiseInfraModel
	if got.K8sCluster == nil || got.K8sCluster.Version != "1.30.2" {
		t.Errorf("k8sCluster = %+v", got.K8sCluster)
	}
	if len(got.Nodes) != 1 || got.Nodes[0].Role != "standalone" {
		t.Errorf("nodes = %+v, want cp-1 standalone", got.Nodes)
	}

	nodes := mergeK8sNodes([]inframodel.NodeProperty{{Hostname: "a"}}, &kubernetes.Kubernetes{Cluster: kubernetes.Cluster{Version: "1.30.2"}})
	if len(nodes) != 1 || nodes[0].Role != "standalone" {
		t.Errorf("mergeK8sNodes = %+v, want a standalone", nodes)
	}
}

const kubeadmCPUUID = "11111111-2222-3333-4444-555555555555"

// kubeadmOnVM is what the agent collects from a CSP VM that is the control
// plane of a kubeadm cluster: the VM itself and a worker only the API sees.
func kubeadmOnVM() kubernetes.Kubernetes {
	return kubernetes.Kubernetes{
		Cluster: kubernetes.Cluster{Name: "kubernetes", Version: "1.30.2", PodCIDR: "198.51.100.0/24"},
		Nodes: []kubernetes.Node{
			{Type: kubernetes.NodeTypeControlPlane, Name: "cp-1",
				NodeInfo: map[string]interface{}{"systemUUID": kubeadmCPUUID}},
			{Type: kubernetes.NodeTypeWorker, Name: "worker-1",
				NodeInfo: map[string]interface{}{"systemUUID": "66666666-7777-8888-9999-000000000000"}},
		},
	}
}

// A csp group that holds both a kubeadm cluster the agent collected from a VM
// and a managed cluster gives the kubeadm one, whichever connection comes
// first: only it lists the nodes the refined infra merges.
func TestGetInfraInfoSourceGroupRefined_AgentClusterBeforeManaged(t *testing.T) {
	orders := map[string][]model.ConnectionInfo{
		"managed first": {
			testConn("k8s", cspGroup, common.ResourceTypeK8s, ""),
			testConn("vm-cp", cspGroup, common.ResourceTypeVM, "198.51.100.20"),
		},
		"vm first": {
			testConn("vm-cp", cspGroup, common.ResourceTypeVM, "198.51.100.20"),
			testConn("k8s", cspGroup, common.ResourceTypeK8s, ""),
		},
	}

	for name, conns := range orders {
		t.Run(name, func(t *testing.T) {
			openTestDB(t)
			createGroup(t, cspGroup, conns)
			host := hostInfra("cp-1")
			host.Compute.OS.Node.Machineid = kubeadmCPUUID
			saveInfra(t, "vm-cp", "success", host, "{}")
			saveKubernetes(t, "vm-cp", kubeadmOnVM())
			saveInfra(t, "k8s", "success", nil, "")
			saveCSPCluster(t, "k8s", managedCluster())

			k8sInfo, managed := groupK8sCluster(cspGroup, conns)
			if k8sInfo == nil || managed || k8sInfo.Cluster.Version != "1.30.2" {
				t.Fatalf("groupK8sCluster = %+v, managed %v, want the kubeadm cluster", k8sInfo, managed)
			}

			rec := groupRequest(t, cspGroup.ID, GetInfraInfoSourceGroupRefined)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
			}
			got := refinedInfraBody(t, rec.Body.Bytes()).OnpremiseInfraModel

			want := &inframodel.K8sClusterProperty{Name: "kubernetes", Version: "1.30.2", PodCIDR: "198.51.100.0/24"}
			if !reflect.DeepEqual(got.K8sCluster, want) {
				t.Errorf("k8sCluster = %+v, want %+v", got.K8sCluster, want)
			}
			var nodes []string
			for _, n := range got.Nodes {
				nodes = append(nodes, n.Hostname+"/"+n.Role)
			}
			if w := []string{"cp-1/control-plane", "worker-1/worker"}; !reflect.DeepEqual(nodes, w) {
				t.Errorf("nodes = %v, want %v", nodes, w)
			}
		})
	}
}

// The single refined infra of a CSP VM connection that holds a kubeadm record
// still merges its nodes.
func TestGetInfraInfoRefined_CSPVMKubeadm(t *testing.T) {
	openTestDB(t)
	createGroup(t, cspGroup, []model.ConnectionInfo{testConn("vm-cp", cspGroup, common.ResourceTypeVM, "198.51.100.20")})
	host := hostInfra("cp-1")
	host.Compute.OS.Node.Machineid = kubeadmCPUUID
	saveInfra(t, "vm-cp", "success", host, "{}")
	saveKubernetes(t, "vm-cp", kubeadmOnVM())

	rec := connRequest(t, http.MethodGet, cspGroup.ID, "vm-cp", GetInfraInfoRefined)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var nodes []string
	for _, n := range refinedInfraBody(t, rec.Body.Bytes()).OnpremiseInfraModel.Nodes {
		nodes = append(nodes, n.Hostname+"/"+n.Role)
	}
	if w := []string{"cp-1/control-plane", "worker-1/worker"}; !reflect.DeepEqual(nodes, w) {
		t.Errorf("nodes = %v, want %v", nodes, w)
	}
}

// A managed cluster has no workloads in its record, so it makes no kubernetes
// entry of the refined software. An on-prem record that names a version and
// holds no workloads still makes one, as before.
func TestDoGetRefinedSoftwareInfo_CSPK8sHasNoEntry(t *testing.T) {
	openTestDB(t)
	createGroup(t, cspGroup, mixedCSPConns())
	saveCSPCluster(t, "k8s", managedCluster())

	got, err := doGetRefinedSoftwareInfo(&software.Software{}, "k8s")
	if err != nil {
		t.Fatal(err)
	}
	if got.Kubernetes != nil {
		t.Errorf("kubernetes = %+v, want none for a CSP-managed cluster", got.Kubernetes)
	}

	createGroup(t, onpremGroup, []model.ConnectionInfo{testConn("onprem-cp", onpremGroup, "", "192.0.2.10")})
	saveKubernetes(t, "onprem-cp", kubernetes.Kubernetes{Cluster: kubernetes.Cluster{Version: "1.30.2"}})

	got, err = doGetRefinedSoftwareInfo(&software.Software{}, "onprem-cp")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Kubernetes) != 1 || got.Kubernetes[0].Version != "1.30.2" {
		t.Errorf("kubernetes = %+v, want the on-prem entry", got.Kubernetes)
	}
}

func TestTryGetHelmInfo_NotSuccess(t *testing.T) {
	openTestDB(t)
	for connID, status := range map[string]string{"conn-failed": "failed", "conn-importing": "importing"} {
		if err := db.DB.Create(&model.SavedHelmInfo{
			ConnectionID: connID, HelmData: `{"release":[{"name":"old"}]}`, Status: status,
		}).Error; err != nil {
			t.Fatal(err)
		}
		if got := tryGetHelmInfo(connID); got != nil {
			t.Errorf("%s: got %+v, want nil for status %q", connID, got, status)
		}
	}
}

// GET kubernetes answers with the record as stored, csp section included; an
// agent-collected record has no csp section and keeps its shape.
func TestGetKubernetesInfo_CSPSection(t *testing.T) {
	openTestDB(t)
	createGroup(t, cspGroup, mixedCSPConns())
	saveCSPCluster(t, "k8s", managedCluster())

	rec := connRequest(t, http.MethodGet, cspGroup.ID, "k8s", GetKubernetesInfo)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var got model.CSPKubernetesRecord
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Cluster.Name != "cls-0a1b2c3d" || got.CSP == nil || len(got.CSP.NodeGroups) != 2 ||
		got.CSP.NodeGroups[0].VMSpec != "s2.medium" {
		t.Errorf("body = %s", rec.Body.String())
	}

	createGroup(t, onpremGroup, []model.ConnectionInfo{testConn("onprem-cp", onpremGroup, "", "192.0.2.10")})
	saveKubernetes(t, "onprem-cp", kubernetes.Kubernetes{Cluster: kubernetes.Cluster{Name: "kubernetes", Version: "1.30.2"}})

	rec = connRequest(t, http.MethodGet, onpremGroup.ID, "onprem-cp", GetKubernetesInfo)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &keys); err != nil {
		t.Fatal(err)
	}
	var names []string
	for k := range keys {
		names = append(names, k)
	}
	if _, ok := keys["csp"]; ok || len(keys) != 4 {
		t.Errorf("keys = %v, want cluster, node_count, nodes, workloads", names)
	}
}

// The group GET carries the csp section of a managed cluster as the single
// GET does, and an agent-collected record without one.
func TestGetKubernetesInfoSourceGroup_CSPSection(t *testing.T) {
	openTestDB(t)
	conns := []model.ConnectionInfo{
		testConn("vm-cp", cspGroup, common.ResourceTypeVM, "198.51.100.20"),
		testConn("k8s", cspGroup, common.ResourceTypeK8s, ""),
	}
	createGroup(t, cspGroup, conns)
	saveKubernetes(t, "vm-cp", kubeadmOnVM())
	saveCSPCluster(t, "k8s", managedCluster())

	rec := groupRequest(t, cspGroup.ID, GetKubernetesInfoSourceGroup)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var got struct {
		Servers []map[string]json.RawMessage `json:"servers"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Servers) != 2 {
		t.Fatalf("servers = %s", rec.Body.String())
	}

	// The list follows the order the connections are read in, which is not
	// the order they were created in, so each record is told by its cluster.
	byCluster := map[string]map[string]json.RawMessage{}
	for _, server := range got.Servers {
		var cluster kubernetes.Cluster
		if err := json.Unmarshal(server["cluster"], &cluster); err != nil {
			t.Fatal(err)
		}
		byCluster[cluster.Name] = server
	}

	agent := byCluster["kubernetes"]
	if agent == nil {
		t.Fatalf("no agent record: %s", rec.Body.String())
	}
	if _, ok := agent["csp"]; ok || len(agent) != 4 {
		t.Errorf("agent record has keys beyond the four agent ones: %s", rec.Body.String())
	}

	managed := byCluster["cls-0a1b2c3d"]
	if managed == nil {
		t.Fatalf("no managed record: %s", rec.Body.String())
	}
	var csp model.CSPKubernetesCluster
	if err := json.Unmarshal(managed["csp"], &csp); err != nil {
		t.Fatalf("managed record has no csp section: %s", rec.Body.String())
	}
	if csp.SystemID != "cls-0a1b2c3d" || len(csp.NodeGroups) != 2 {
		t.Errorf("csp = %+v", csp)
	}
}
