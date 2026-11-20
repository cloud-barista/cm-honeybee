package controller

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"

	inframodel "github.com/cloud-barista/cm-beetle/imdl/on-premise-model"
	"github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/cloud-barista/cm-honeybee/server/lib/spider"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
)

// bucket/refined answers exactly as data/refined, for the group and for each
// connection, a bucket that was not collected included.
func TestBucketRefined_SameAsDataRefined(t *testing.T) {
	openTestDB(t)
	createGroup(t, osakaGroup, []model.ConnectionInfo{
		testConn("near", osakaGroup, common.ResourceTypeObjectStorage, ""),
		testConn("legacy", osakaGroup, common.ResourceTypeObjectStorage, ""),
		testConn("far", osakaGroup, common.ResourceTypeObjectStorage, ""),
		testConn("never", osakaGroup, common.ResourceTypeObjectStorage, ""),
	})
	for id, region := range map[string]string{"near": "ap-northeast-3", "legacy": "", "far": "ap-northeast-2"} {
		d := spider.S3BucketDetail{ItemsTried: 1, ItemsRead: 1}
		d.Name = id
		if err := storeBucketDetail(id, "ap-northeast-3", region, &d); err != nil {
			t.Fatal(err)
		}
	}

	data := groupRequest(t, osakaGroup.ID, GetDataInfoSourceGroupRefined)
	bucket := groupRequest(t, osakaGroup.ID, GetBucketInfoSourceGroupRefined)
	if data.Code != http.StatusOK || bucket.Code != data.Code || bucket.Body.String() != data.Body.String() {
		t.Errorf("group: bucket/refined = %d %s, data/refined = %d %s",
			bucket.Code, bucket.Body.String(), data.Code, data.Body.String())
	}
	if !strings.Contains(bucket.Body.String(), `"near"`) {
		t.Errorf("group bucket/refined = %s, want the near bucket", bucket.Body.String())
	}

	for _, id := range []string{"near", "far", "never"} {
		data := connRequest(t, http.MethodGet, osakaGroup.ID, id, GetDataInfoRefined)
		bucket := connRequest(t, http.MethodGet, osakaGroup.ID, id, GetBucketInfoRefined)
		if bucket.Code != data.Code || bucket.Body.String() != data.Body.String() {
			t.Errorf("%s: bucket/refined = %d %s, data/refined = %d %s",
				id, bucket.Code, bucket.Body.String(), data.Code, data.Body.String())
		}
	}
}

func refinedK8sBody(t *testing.T, body []byte) model.RefinedK8sClusterModel {
	t.Helper()

	var got model.RefinedK8sClusterModel
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("%v: %s", err, body)
	}
	return got
}

// k8s/refined of a group with a managed cluster gives the k8sCluster
// infra/refined gives, with the connection it came from and its node groups.
func TestK8sRefined_ManagedCluster(t *testing.T) {
	openTestDB(t)
	createGroup(t, cspGroup, mixedCSPConns())
	saveInfra(t, "vm-ip", "success", hostInfra("node-ip"), "{}")
	saveInfra(t, "k8s", "success", nil, "")
	saveCSPCluster(t, "k8s", managedCluster())

	infraRec := groupRequest(t, cspGroup.ID, GetInfraInfoSourceGroupRefined)
	wantCluster := refinedInfraBody(t, infraRec.Body.Bytes()).OnpremiseInfraModel.K8sCluster
	if wantCluster == nil {
		t.Fatalf("infra/refined has no k8sCluster: %s", infraRec.Body.String())
	}

	rec := groupRequest(t, cspGroup.ID, GetK8sInfoSourceGroupRefined)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	got := refinedK8sBody(t, rec.Body.Bytes())
	if !reflect.DeepEqual(got.K8sCluster, wantCluster) {
		t.Errorf("k8sCluster = %+v, want %+v", got.K8sCluster, wantCluster)
	}
	if got.ConnectionID != "k8s" || !got.Managed {
		t.Errorf("connection_id = %q, managed = %v, want k8s and true", got.ConnectionID, got.Managed)
	}
	if got.CSP == nil || got.CSP.SystemID != "cls-0a1b2c3d" || len(got.CSP.NodeGroups) != 2 ||
		got.CSP.NodeGroups[0].DesiredNodeSize != 2 {
		t.Errorf("csp = %+v, want the cluster's two node groups", got.CSP)
	}

	single := connRequest(t, http.MethodGet, cspGroup.ID, "k8s", GetK8sInfoRefined)
	if single.Code != http.StatusOK {
		t.Fatalf("single: status = %d, body = %s", single.Code, single.Body.String())
	}
	singleInfra := connRequest(t, http.MethodGet, cspGroup.ID, "k8s", GetInfraInfoRefined)
	if g, w := refinedK8sBody(t, single.Body.Bytes()), refinedInfraBody(t, singleInfra.Body.Bytes()); !reflect.DeepEqual(g.K8sCluster, w.OnpremiseInfraModel.K8sCluster) ||
		!reflect.DeepEqual(g, got) {
		t.Errorf("single = %+v, want %+v with infra/refined's k8sCluster", g, got)
	}

	for _, id := range []string{"vm-ip", "bucket"} {
		rec := connRequest(t, http.MethodGet, cspGroup.ID, id, GetK8sInfoRefined)
		if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "no kubernetes cluster") {
			t.Errorf("%s: status = %d, body = %s, want 400", id, rec.Code, rec.Body.String())
		}
	}
}

// A cluster the agent collected is the group's before a managed one, as in
// infra/refined, and carries no csp section.
func TestK8sRefined_AgentClusterBeforeManaged(t *testing.T) {
	openTestDB(t)
	conns := []model.ConnectionInfo{
		testConn("k8s", cspGroup, common.ResourceTypeK8s, ""),
		testConn("vm-cp", cspGroup, common.ResourceTypeVM, "198.51.100.20"),
	}
	createGroup(t, cspGroup, conns)
	host := hostInfra("cp-1")
	host.Compute.OS.Node.Machineid = kubeadmCPUUID
	saveInfra(t, "vm-cp", "success", host, "{}")
	saveKubernetes(t, "vm-cp", kubeadmOnVM())
	saveInfra(t, "k8s", "success", nil, "")
	saveCSPCluster(t, "k8s", managedCluster())

	infraRec := groupRequest(t, cspGroup.ID, GetInfraInfoSourceGroupRefined)
	wantCluster := refinedInfraBody(t, infraRec.Body.Bytes()).OnpremiseInfraModel.K8sCluster

	got := refinedK8sBody(t, groupRequest(t, cspGroup.ID, GetK8sInfoSourceGroupRefined).Body.Bytes())
	want := model.RefinedK8sClusterModel{ConnectionID: "vm-cp", Managed: false, K8sCluster: wantCluster}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("k8s/refined = %+v, want %+v", got, want)
	}
	if wantCluster == nil || wantCluster.Version != "1.30.2" {
		t.Errorf("infra/refined k8sCluster = %+v, want the kubeadm cluster", wantCluster)
	}
}

// A group without a cluster answers 200 with k8sCluster null.
func TestK8sRefined_NoCluster(t *testing.T) {
	openTestDB(t)
	createGroup(t, cspGroup, mixedCSPConns())

	rec := groupRequest(t, cspGroup.ID, GetK8sInfoSourceGroupRefined)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"k8sCluster": null`) ||
		!strings.Contains(rec.Body.String(), `"managed": false`) {
		t.Errorf("status = %d, body = %s, want 200 with k8sCluster null", rec.Code, rec.Body.String())
	}
}

func refinedNLBBody(t *testing.T, body []byte) []inframodel.NlbProperty {
	t.Helper()

	var got model.RefinedNLBModel
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("%v: %s", err, body)
	}
	return got.NLBs
}

// nlb/refined gives the nlbs of infra/refined: the HAProxy NLB on the node and
// the collected CSP NLB, for the group and for each connection.
func TestNLBRefined_SameAsInfraRefined(t *testing.T) {
	openTestDB(t)
	createGroup(t, nlbGroup, nlbGroupConns())
	saveWebVM(t)
	n := awsNLB("internet-facing")
	if err := upsertSavedNLB("nlb", nlbInfoToNLB(&n)); err != nil {
		t.Fatal(err)
	}

	infraRec := groupRequest(t, nlbGroup.ID, GetInfraInfoSourceGroupRefined)
	want := refinedInfraBody(t, infraRec.Body.Bytes()).OnpremiseInfraModel.NLBs
	if len(want) != 2 {
		t.Fatalf("infra/refined nlbs = %+v, want the HAProxy and the CSP one", want)
	}

	rec := groupRequest(t, nlbGroup.ID, GetNLBInfoSourceGroupRefined)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if got := refinedNLBBody(t, rec.Body.Bytes()); !reflect.DeepEqual(got, want) {
		t.Errorf("group nlbs = %+v, want %+v", got, want)
	}

	for _, id := range []string{"nlb", "vm-web"} {
		infraRec := connRequest(t, http.MethodGet, nlbGroup.ID, id, GetInfraInfoRefined)
		want := refinedInfraBody(t, infraRec.Body.Bytes()).OnpremiseInfraModel.NLBs
		rec := connRequest(t, http.MethodGet, nlbGroup.ID, id, GetNLBInfoRefined)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: status = %d, body = %s", id, rec.Code, rec.Body.String())
		}
		if got := refinedNLBBody(t, rec.Body.Bytes()); len(want) != 1 || !reflect.DeepEqual(got, want) {
			t.Errorf("%s: nlbs = %+v, want %+v", id, got, want)
		}
	}
}

// A group without a load balancer answers with an empty list, not null.
func TestNLBRefined_Empty(t *testing.T) {
	openTestDB(t)
	createGroup(t, cspGroup, mixedCSPConns())

	rec := groupRequest(t, cspGroup.ID, GetNLBInfoSourceGroupRefined)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"nlbs": []`) {
		t.Errorf("status = %d, body = %s, want 200 with an empty list", rec.Code, rec.Body.String())
	}
}
