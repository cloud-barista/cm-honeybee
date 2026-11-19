package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"testing"

	"github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/cloud-barista/cm-honeybee/server/db"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"github.com/labstack/echo/v4"
)

var (
	sshGroup   = &model.SourceGroup{ID: "sg-ssh", Name: "sg-ssh", Type: common.SourceGroupTypeSSH}
	minioGroup = &model.SourceGroup{ID: "sg-minio", Name: "sg-minio", Type: common.SourceGroupTypeMinIO}
)

// queryRequest calls a list handler with the query string given and the
// source group ID, when one is given, as the sgId parameter.
func queryRequest(t *testing.T, sgID string, query string, handler echo.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()

	rec := httptest.NewRecorder()
	c := echo.New().NewContext(httptest.NewRequest(http.MethodGet, "/?"+query, nil), rec)
	if sgID != "" {
		c.SetParamNames("sgId")
		c.SetParamValues(sgID)
	}
	if err := handler(c); err != nil {
		t.Fatal(err)
	}
	return rec
}

func listedGroupNames(t *testing.T, query string) []string {
	t.Helper()

	rec := queryRequest(t, "", query, ListSourceGroup)
	if rec.Code != http.StatusOK {
		t.Fatalf("%s: status = %d, body = %s", query, rec.Code, rec.Body.String())
	}
	var res model.ListSourceGroupRes
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	names := []string{}
	for _, sg := range res.SourceGroup {
		names = append(names, sg.Name)
	}
	return names
}

// listedConns gives the resource_type of each listed connection by its name.
func listedConns(t *testing.T, sgID string, query string) map[string]string {
	t.Helper()

	rec := queryRequest(t, sgID, query, ListConnectionInfo)
	if rec.Code != http.StatusOK {
		t.Fatalf("%s: status = %d, body = %s", query, rec.Code, rec.Body.String())
	}
	var res model.ListConnectionInfoRes
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, ci := range res.ConnectionInfo {
		got[ci.Name] = ci.ResourceType
	}
	if res.ConnectionInfoStatusCount.ConnectionInfoTotal != len(got) {
		t.Errorf("%s: total = %d, want %d", query, res.ConnectionInfoStatusCount.ConnectionInfoTotal, len(got))
	}
	return got
}

// The group list narrows by type. onprem matches the legacy ssh type and the
// empty type of a group made before the type existed; a type that names no
// group type is refused.
func TestListSourceGroup_TypeFilter(t *testing.T) {
	openTestDB(t)
	for _, sg := range []*model.SourceGroup{onpremGroup, sshGroup, cspGroup, fsGroup, dbGroup, minioGroup} {
		createGroup(t, sg, nil)
	}
	untyped := &model.SourceGroup{ID: "sg-untyped", Name: "sg-untyped"}
	createGroup(t, untyped, nil)
	if err := db.DB.Model(&model.SourceGroup{}).Where("id = ?", untyped.ID).Update("type", "").Error; err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		query string
		want  []string
	}{
		{"", []string{"sg-csp", "sg-db", "sg-fs", "sg-minio", "sg-onprem", "sg-ssh", "sg-untyped"}},
		{"type=onprem", []string{"sg-onprem", "sg-ssh", "sg-untyped"}},
		{"type=csp", []string{"sg-csp"}},
		{"type=CSP", []string{"sg-csp"}},
		{"type=fs", []string{"sg-fs"}},
		{"type=db", []string{"sg-db"}},
		{"type=minio", []string{"sg-minio"}},
		{"type=csp&name=onprem", []string{}},
		{"type=onprem&page=1&row=2", []string{"sg-onprem", "sg-ssh"}},
	}
	for _, tt := range tests {
		if got := listedGroupNames(t, tt.query); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%q: groups = %v, want %v", tt.query, got, tt.want)
		}
	}

	for _, query := range []string{"type=vm", "type=ssh", "type=on-prem"} {
		if rec := queryRequest(t, "", query, ListSourceGroup); rec.Code != http.StatusBadRequest {
			t.Errorf("%q: status = %d, want 400 (body %s)", query, rec.Code, rec.Body.String())
		}
	}
}

// The connection list narrows by resource_type, and every on-prem connection
// carries one: a row stored before resource_type existed reads as vm and is
// matched by resource_type=vm. Connections of an fs, db or minio group have no
// resource_type and are not taken for VMs.
func TestListConnectionInfo_ResourceTypeFilter(t *testing.T) {
	openTestDB(t)
	createGroup(t, onpremGroup, []model.ConnectionInfo{
		testConn("onprem-untyped", onpremGroup, "", "192.0.2.10"),
		testConn("onprem-vm", onpremGroup, common.ResourceTypeVM, "192.0.2.11"),
		testConn("onprem-k8s", onpremGroup, common.ResourceTypeK8s, ""),
	})
	createGroup(t, cspGroup, mixedCSPConns())
	createGroup(t, fsGroup, []model.ConnectionInfo{testConn("fs-host", fsGroup, "", "192.0.2.20")})

	onpremAll := map[string]string{"onprem-untyped": "vm", "onprem-vm": "vm", "onprem-k8s": "k8s"}
	tests := []struct {
		sg    *model.SourceGroup
		query string
		want  map[string]string
	}{
		{onpremGroup, "", onpremAll},
		{onpremGroup, "resource_type=vm", map[string]string{"onprem-untyped": "vm", "onprem-vm": "vm"}},
		{onpremGroup, "resource_type=k8s", map[string]string{"onprem-k8s": "k8s"}},
		{onpremGroup, "resource_type=nlb", map[string]string{}},
		{cspGroup, "resource_type=vm", map[string]string{"vm-ip": "vm", "vm-noip": "vm"}},
		{cspGroup, "resource_type=VM", map[string]string{"vm-ip": "vm", "vm-noip": "vm"}},
		{cspGroup, "resource_type=object_storage", map[string]string{"bucket": "object_storage"}},
		{cspGroup, "resource_type=nlb", map[string]string{"nlb": "nlb"}},
		{cspGroup, "resource_type=rdbms", map[string]string{"rdbms": "rdbms"}},
		{cspGroup, "resource_type=k8s", map[string]string{"k8s": "k8s"}},
		{fsGroup, "", map[string]string{"fs-host": ""}},
		{fsGroup, "resource_type=vm", map[string]string{}},
	}
	for _, tt := range tests {
		if got := listedConns(t, tt.sg.ID, tt.query); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s %q: connections = %v, want %v", tt.sg.ID, tt.query, got, tt.want)
		}
	}

	if rec := queryRequest(t, onpremGroup.ID, "resource_type=bucket", ListConnectionInfo); rec.Code != http.StatusBadRequest {
		t.Errorf("resource_type=bucket: status = %d, want 400 (body %s)", rec.Code, rec.Body.String())
	}

	// The filling happens on the response only; the stored row is left as it is.
	stored := getConnection(t, "onprem-untyped")
	if stored.ResourceType != "" {
		t.Errorf("stored resource_type = %q, want the row untouched", stored.ResourceType)
	}

	single := connRequest(t, http.MethodGet, onpremGroup.ID, "onprem-untyped", GetConnectionInfo)
	if single.Code != http.StatusOK {
		t.Fatalf("single status = %d, body = %s", single.Code, single.Body.String())
	}
	var one model.ConnectionInfo
	if err := json.Unmarshal(single.Body.Bytes(), &one); err != nil {
		t.Fatal(err)
	}
	if one.ResourceType != common.ResourceTypeVM {
		t.Errorf("single resource_type = %q, want vm", one.ResourceType)
	}
}

// ssh_port matches the port, not a pattern of it: the filter compared the
// column with "%22%" and so never matched.
func TestListConnectionInfo_SSHPortFilter(t *testing.T) {
	openTestDB(t)
	conns := []model.ConnectionInfo{
		testConn("port-22", onpremGroup, common.ResourceTypeVM, "192.0.2.10"),
		testConn("port-2222", onpremGroup, common.ResourceTypeVM, "192.0.2.11"),
		testConn("port-122", onpremGroup, common.ResourceTypeVM, "192.0.2.12"),
	}
	conns[0].SSHPort, conns[1].SSHPort, conns[2].SSHPort = "22", "2222", "122"
	createGroup(t, onpremGroup, conns)

	tests := []struct {
		query string
		want  []string
	}{
		{"ssh_port=22", []string{"port-22"}},
		{"ssh_port=2222", []string{"port-2222"}},
		{"ssh_port=2", []string{}},
		{"ssh_port=22&resource_type=vm", []string{"port-22"}},
		{"ssh_port=0", []string{"port-122", "port-22", "port-2222"}},
	}
	for _, tt := range tests {
		got := []string{}
		for name := range listedConns(t, onpremGroup.ID, tt.query) {
			got = append(got, name)
		}
		sort.Strings(got)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%q: connections = %v, want %v", tt.query, got, tt.want)
		}
	}
}

// An on-prem connection registered without resource_type is stored as a vm,
// so the stored row and the response agree.
func TestCheckCreateConnectionInfoReq_OnpremStoresVM(t *testing.T) {
	for _, rt := range []string{"", "vm", " VM "} {
		ci, err := checkCreateConnectionInfoReq(onpremGroup, &model.CreateConnectionInfoReq{
			Name: "host", IPAddress: "192.0.2.10", SSHPort: "22", User: "ubuntu",
			PrivateKey: "test-key", ResourceType: rt,
		})
		if err != nil {
			t.Fatalf("resource_type %q: %v", rt, err)
		}
		if ci.ResourceType != common.ResourceTypeVM {
			t.Errorf("resource_type %q: stored %q, want vm", rt, ci.ResourceType)
		}
	}

	ci, err := checkCreateConnectionInfoReq(onpremGroup, &model.CreateConnectionInfoReq{
		Name: "cluster", ResourceType: "k8s",
		Kubeconfig: "apiVersion: v1\nkind: Config\nclusters:\n- name: c1\n  cluster:\n    server: https://192.0.2.30:6443\n",
	})
	if err != nil {
		t.Fatal(err)
	}
	if ci.ResourceType != common.ResourceTypeK8s {
		t.Errorf("k8s connection stored resource_type %q, want k8s", ci.ResourceType)
	}
}
