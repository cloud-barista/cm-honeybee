package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	softwaremodel "github.com/cloud-barista/cm-grasshopper/smdl"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/infra"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/software"
	"github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/cloud-barista/cm-honeybee/server/db"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"github.com/labstack/echo/v4"
)

var (
	onpremGroup = &model.SourceGroup{ID: "sg-onprem", Name: "sg-onprem", Type: common.SourceGroupTypeOnprem}
	cspGroup    = &model.SourceGroup{ID: "sg-csp", Name: "sg-csp", Type: common.SourceGroupTypeCSP}
	fsGroup     = &model.SourceGroup{ID: "sg-fs", Name: "sg-fs", Type: common.SourceGroupTypeFS}
	dbGroup     = &model.SourceGroup{ID: "sg-db", Name: "sg-db", Type: common.SourceGroupTypeDB}
)

func testConn(id string, sg *model.SourceGroup, resourceType string, ip string) model.ConnectionInfo {
	return model.ConnectionInfo{ID: id, Name: id, SourceGroupID: sg.ID, ResourceType: resourceType, IPAddress: ip}
}

// mixedCSPConns is a csp group holding one connection of every resource type,
// with a VM that has an IP address and one that has none.
func mixedCSPConns() []model.ConnectionInfo {
	return []model.ConnectionInfo{
		testConn("vm-ip", cspGroup, common.ResourceTypeVM, "198.51.100.10"),
		testConn("vm-noip", cspGroup, common.ResourceTypeVM, ""),
		testConn("bucket", cspGroup, common.ResourceTypeObjectStorage, ""),
		testConn("rdbms", cspGroup, common.ResourceTypeRDBMS, ""),
		testConn("k8s", cspGroup, common.ResourceTypeK8s, ""),
		testConn("nlb", cspGroup, common.ResourceTypeNLB, ""),
	}
}

func connIDs(conns []model.ConnectionInfo) []string {
	ids := []string{}
	for _, c := range conns {
		ids = append(ids, c.ID)
	}
	return ids
}

func TestConnectionClassification(t *testing.T) {
	tests := []struct {
		sg      *model.SourceGroup
		conn    model.ConnectionInfo
		isHost  bool
		wantErr error
	}{
		{onpremGroup, testConn("onprem-untyped", onpremGroup, "", "192.0.2.10"), true, nil},
		{onpremGroup, testConn("onprem-vm", onpremGroup, common.ResourceTypeVM, "192.0.2.11"), true, nil},
		// Only CSP connections are judged by their address; an on-prem one is
		// left to the agent call as before.
		{onpremGroup, testConn("onprem-vm-noip", onpremGroup, common.ResourceTypeVM, ""), true, nil},
		{onpremGroup, testConn("onprem-k8s", onpremGroup, common.ResourceTypeK8s, ""), false, ErrOnpremK8sAgent},
		{cspGroup, testConn("vm-ip", cspGroup, common.ResourceTypeVM, "198.51.100.10"), true, nil},
		{cspGroup, testConn("vm-noip", cspGroup, common.ResourceTypeVM, ""), true, ErrAgentNotApplicable},
		{cspGroup, testConn("bucket", cspGroup, common.ResourceTypeObjectStorage, ""), false, ErrAgentNotApplicable},
		{cspGroup, testConn("rdbms", cspGroup, common.ResourceTypeRDBMS, ""), false, ErrAgentNotApplicable},
		{cspGroup, testConn("k8s", cspGroup, common.ResourceTypeK8s, ""), false, ErrAgentNotApplicable},
		{cspGroup, testConn("nlb", cspGroup, common.ResourceTypeNLB, ""), false, ErrAgentNotApplicable},
		// A csp resource that is not a VM has no host even if an address was
		// typed in, so the address must not make it look collectable.
		{cspGroup, testConn("bucket-ip", cspGroup, common.ResourceTypeObjectStorage, "198.51.100.11"), false, ErrAgentNotApplicable},
		// fs, db and minio connections keep their earlier handling: they become
		// nodes once collected and the agent imports accept them.
		{fsGroup, testConn("fs-host", fsGroup, "", "192.0.2.12"), true, nil},
		{dbGroup, testConn("db-host", dbGroup, "", ""), true, nil},
	}

	for _, tt := range tests {
		t.Run(tt.conn.ID, func(t *testing.T) {
			if got := isHostConnection(tt.sg, &tt.conn); got != tt.isHost {
				t.Errorf("isHostConnection = %v, want %v", got, tt.isHost)
			}
			err := agentNotApplicable(tt.sg, &tt.conn)
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("agentNotApplicable = %v, want nil", err)
				}
			} else if !errors.Is(err, tt.wantErr) {
				t.Errorf("agentNotApplicable = %v, want %v", err, tt.wantErr)
			}
			if got := isAgentCollectable(tt.sg, &tt.conn); got != (tt.wantErr == nil) {
				t.Errorf("isAgentCollectable = %v, want %v", got, tt.wantErr == nil)
			}
		})
	}
}

func TestGroupImportTargets(t *testing.T) {
	csp := mixedCSPConns()

	if got, want := connIDs(agentImportTargets(cspGroup, csp)), []string{"vm-ip"}; !reflect.DeepEqual(got, want) {
		t.Errorf("csp agent targets = %v, want %v", got, want)
	}
	if got, want := connIDs(infraImportTargets(cspGroup, csp)), connIDs(csp); !reflect.DeepEqual(got, want) {
		t.Errorf("csp infra targets = %v, want %v", got, want)
	}

	onprem := []model.ConnectionInfo{
		testConn("onprem-vm", onpremGroup, "", "192.0.2.10"),
		testConn("onprem-k8s", onpremGroup, common.ResourceTypeK8s, ""),
	}
	if got, want := connIDs(agentImportTargets(onpremGroup, onprem)), []string{"onprem-vm"}; !reflect.DeepEqual(got, want) {
		t.Errorf("onprem agent targets = %v, want %v", got, want)
	}
	if got, want := connIDs(infraImportTargets(onpremGroup, onprem)), []string{"onprem-vm"}; !reflect.DeepEqual(got, want) {
		t.Errorf("onprem infra targets = %v, want %v", got, want)
	}

	fs := []model.ConnectionInfo{testConn("fs-host", fsGroup, "", "192.0.2.12")}
	if got, want := connIDs(agentImportTargets(fsGroup, fs)), []string{"fs-host"}; !reflect.DeepEqual(got, want) {
		t.Errorf("fs agent targets = %v, want %v", got, want)
	}
}

type fakeResult struct {
	ID     string
	Status string
	Error  string
}

func TestImportEach(t *testing.T) {
	targets := []model.ConnectionInfo{{ID: "a"}, {ID: "b"}, {ID: "c"}}
	failed := func(id string, err error) fakeResult { return fakeResult{ID: id, Status: "failed", Error: err.Error()} }

	t.Run("one failure does not stop the rest", func(t *testing.T) {
		do := func(id string) (*fakeResult, error) {
			if id == "b" {
				return nil, errors.New("b is unreachable")
			}
			return &fakeResult{ID: id, Status: "success"}, nil
		}
		got, err := importEach(targets, do, failed)
		if err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
		want := []fakeResult{{"a", "success", ""}, {"b", "failed", "b is unreachable"}, {"c", "success", ""}}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("all failing returns the first error", func(t *testing.T) {
		do := func(id string) (*fakeResult, error) { return nil, errors.New(id + " failed") }
		got, err := importEach(targets, do, failed)
		if err == nil || err.Error() != "a failed" {
			t.Errorf("err = %v, want the first error", err)
		}
		if got != nil {
			t.Errorf("got %v, want nil", got)
		}
	})

	t.Run("no target is an empty result", func(t *testing.T) {
		called := false
		do := func(id string) (*fakeResult, error) { called = true; return nil, errors.New("x") }
		got, err := importEach(nil, do, failed)
		if err != nil || called {
			t.Fatalf("err = %v, called = %v", err, called)
		}
		raw, _ := json.Marshal(got)
		if string(raw) != "[]" {
			t.Errorf("json = %s, want []", raw)
		}
	})
}

func createGroup(t *testing.T, sg *model.SourceGroup, conns []model.ConnectionInfo) {
	t.Helper()

	sgCopy := *sg
	if err := db.DB.Create(&sgCopy).Error; err != nil {
		t.Fatal(err)
	}
	for i := range conns {
		c := conns[i]
		if err := db.DB.Create(&c).Error; err != nil {
			t.Fatal(err)
		}
	}
}

// An agent import asked for a connection without an agent is refused before
// any record is written, so what the CSP collection stored stays as it was.
func TestDoImportAgent_RefusesWithoutTouchingRecords(t *testing.T) {
	openTestDB(t)
	createGroup(t, cspGroup, mixedCSPConns())

	if err := db.DB.Create(&model.SavedKubernetesInfo{ConnectionID: "k8s", KubernetesData: `{"node_count":{"total":2}}`, Status: "success"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.DB.Create(&model.SavedDataInfo{ConnectionID: "bucket", DataData: `{"minio":{}}`, Status: "success"}).Error; err != nil {
		t.Fatal(err)
	}

	if _, err := doImportKubernetes("k8s"); !errors.Is(err, ErrAgentNotApplicable) {
		t.Errorf("kubernetes on csp k8s: err = %v, want ErrAgentNotApplicable", err)
	}
	if _, err := doImportData("bucket"); !errors.Is(err, ErrAgentNotApplicable) {
		t.Errorf("data on csp bucket: err = %v, want ErrAgentNotApplicable", err)
	}
	if _, err := doImportSoftware("vm-noip", false); !errors.Is(err, ErrAgentNotApplicable) {
		t.Errorf("software on csp vm without ip: err = %v, want ErrAgentNotApplicable", err)
	}
	if _, err := doImportHelm("rdbms"); !errors.Is(err, ErrAgentNotApplicable) {
		t.Errorf("helm on csp rdbms: err = %v, want ErrAgentNotApplicable", err)
	}

	var k model.SavedKubernetesInfo
	if err := db.DB.First(&k, "connection_id = ?", "k8s").Error; err != nil {
		t.Fatal(err)
	}
	if k.Status != "success" || k.KubernetesData != `{"node_count":{"total":2}}` {
		t.Errorf("csp k8s record changed: %#v", k)
	}
	var d model.SavedDataInfo
	if err := db.DB.First(&d, "connection_id = ?", "bucket").Error; err != nil {
		t.Fatal(err)
	}
	if d.Status != "success" || d.DataData != `{"minio":{}}` {
		t.Errorf("csp bucket record changed: %#v", d)
	}
	var n int64
	db.DB.Model(&model.SavedSoftwareInfo{}).Where("connection_id = ?", "vm-noip").Count(&n)
	if n != 0 {
		t.Errorf("a refused import must not register a record, found %d", n)
	}
	db.DB.Model(&model.SavedHelmInfo{}).Where("connection_id = ?", "rdbms").Count(&n)
	if n != 0 {
		t.Errorf("a refused import must not register a record, found %d", n)
	}
}

func stubRefreshCSP(t *testing.T, fail map[string]bool) {
	t.Helper()

	orig := refreshCSP
	t.Cleanup(func() { refreshCSP = orig })
	refreshCSP = func(_ *model.SourceGroup, ci *model.ConnectionInfo, _ importInfraOptions) error {
		if fail[ci.ID] {
			return errors.New("resource " + ci.ResourceID + " not found")
		}
		return nil
	}
}

// One connection failing to import leaves the others collected, and the failed
// one is listed, and stored, with status failed.
func TestImportInfraGroup_PartialFailure(t *testing.T) {
	openTestDB(t)
	conns := []model.ConnectionInfo{
		testConn("bucket-ok", cspGroup, common.ResourceTypeObjectStorage, ""),
		testConn("bucket-gone", cspGroup, common.ResourceTypeObjectStorage, ""),
	}
	conns[0].ResourceID, conns[1].ResourceID = "bucket-a", "bucket-b"
	createGroup(t, cspGroup, conns)
	stubRefreshCSP(t, map[string]bool{"bucket-gone": true})

	got, err := importInfraGroup(cspGroup, conns, importInfraOptions{})
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	status := map[string]string{}
	errMsg := map[string]string{}
	for _, r := range got {
		status[r.ConnectionID] = r.Status
		errMsg[r.ConnectionID] = r.ErrorMessage
	}
	want := map[string]string{"bucket-ok": "success", "bucket-gone": "failed"}
	if !reflect.DeepEqual(status, want) {
		t.Errorf("statuses = %v, want %v", status, want)
	}
	if errMsg["bucket-ok"] != "" {
		t.Errorf("a successful connection must carry no error, got %q", errMsg["bucket-ok"])
	}
	if !strings.Contains(errMsg["bucket-gone"], "resource bucket-b not found") {
		t.Errorf("error_message = %q, want the import error", errMsg["bucket-gone"])
	}

	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	var decoded []map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	for _, d := range decoded {
		_, has := d["error_message"]
		if wantHas := d["connection_id"] == "bucket-gone"; has != wantHas {
			t.Errorf("%v: error_message present = %v, want %v", d["connection_id"], has, wantHas)
		}
	}

	var rec model.SavedInfraInfo
	if err := db.DB.First(&rec, "connection_id = ?", "bucket-gone").Error; err != nil {
		t.Fatal(err)
	}
	if rec.Status != "failed" {
		t.Errorf("stored status = %q, want failed", rec.Status)
	}
}

func TestImportInfraGroup_AllFailOrNone(t *testing.T) {
	openTestDB(t)
	conns := []model.ConnectionInfo{
		testConn("bucket-1", cspGroup, common.ResourceTypeObjectStorage, ""),
		testConn("bucket-2", cspGroup, common.ResourceTypeObjectStorage, ""),
	}
	conns[0].ResourceID, conns[1].ResourceID = "bucket-1", "bucket-2"
	createGroup(t, cspGroup, conns)
	stubRefreshCSP(t, map[string]bool{"bucket-1": true, "bucket-2": true})

	if _, err := importInfraGroup(cspGroup, conns, importInfraOptions{}); err == nil ||
		!strings.Contains(err.Error(), "bucket-1") {
		t.Errorf("err = %v, want the first connection's error", err)
	}

	onprem := []model.ConnectionInfo{testConn("onprem-k8s", onpremGroup, common.ResourceTypeK8s, "")}
	createGroup(t, onpremGroup, onprem)
	got, err := importInfraGroup(onpremGroup, onprem, importInfraOptions{})
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if got == nil || len(got) != 0 {
		t.Errorf("got %#v, want an empty list", got)
	}
}

func saveInfra(t *testing.T, connID string, status string, infraInfo *infra.Infra, cspData string) {
	t.Helper()

	rec := &model.SavedInfraInfo{ConnectionID: connID, Status: status, CSPData: cspData}
	if infraInfo != nil {
		raw, err := json.Marshal(infraInfo)
		if err != nil {
			t.Fatal(err)
		}
		rec.InfraData = string(raw)
	}
	if err := db.DB.Create(rec).Error; err != nil {
		t.Fatal(err)
	}
}

func hostInfra(hostname string) *infra.Infra {
	var i infra.Infra
	i.Compute.OS.Node.Hostname = hostname
	return &i
}

func groupRequest(t *testing.T, sgID string, handler echo.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()

	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), rec)
	c.SetParamNames("sgId")
	c.SetParamValues(sgID)
	if err := handler(c); err != nil {
		t.Fatal(err)
	}
	return rec
}

// The group refined infra holds one node per collected VM. The rows a CSP
// import leaves for a bucket, a database, a cluster or a load balancer are
// empty successes and must not turn into nodes, and a VM that was never
// collected or whose import failed is skipped instead of failing the group.
func TestGetInfraInfoSourceGroupRefined_OnlyCollectedVMs(t *testing.T) {
	openTestDB(t)
	conns := append(mixedCSPConns(),
		testConn("vm-failed", cspGroup, common.ResourceTypeVM, "198.51.100.12"),
		testConn("vm-never", cspGroup, common.ResourceTypeVM, "198.51.100.13"),
		testConn("bucket-ip", cspGroup, common.ResourceTypeObjectStorage, "198.51.100.16"),
	)
	createGroup(t, cspGroup, conns)

	saveInfra(t, "vm-ip", "success", hostInfra("node-ip"), "{}")
	saveInfra(t, "vm-noip", "success", nil, "{}")
	saveInfra(t, "vm-failed", "failed", hostInfra("node-failed"), "")
	for _, id := range []string{"bucket", "rdbms", "k8s", "nlb"} {
		saveInfra(t, id, "success", nil, "")
	}
	// A bucket registered with an address once went through the agent import
	// and holds host data; it is still not a machine of the source.
	saveInfra(t, "bucket-ip", "success", hostInfra("bucket-host"), "")

	srcs := refinedInfraSources(cspGroup, conns)
	var ids []string
	for _, s := range srcs {
		ids = append(ids, s.connID)
	}
	if want := []string{"vm-ip", "vm-noip"}; !reflect.DeepEqual(ids, want) {
		t.Errorf("sources = %v, want %v", ids, want)
	}

	rec := groupRequest(t, cspGroup.ID, GetInfraInfoSourceGroupRefined)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var got model.RefinedOnpremiseInfraModel
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	var hostnames []string
	for _, n := range got.OnpremiseInfraModel.Nodes {
		hostnames = append(hostnames, n.Hostname)
	}
	if want := []string{"node-ip", ""}; !reflect.DeepEqual(hostnames, want) {
		t.Errorf("node hostnames = %q, want %q", hostnames, want)
	}
}

func TestGetInfraInfoSourceGroupRefined_OnpremSkipsUncollected(t *testing.T) {
	openTestDB(t)
	conns := []model.ConnectionInfo{
		testConn("onprem-a", onpremGroup, "", "192.0.2.10"),
		testConn("onprem-b", onpremGroup, common.ResourceTypeVM, "192.0.2.11"),
		testConn("onprem-k8s", onpremGroup, common.ResourceTypeK8s, ""),
	}
	createGroup(t, onpremGroup, conns)
	saveInfra(t, "onprem-a", "success", hostInfra("host-a"), "")

	rec := groupRequest(t, onpremGroup.ID, GetInfraInfoSourceGroupRefined)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var got model.RefinedOnpremiseInfraModel
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.OnpremiseInfraModel.Nodes) != 1 || got.OnpremiseInfraModel.Nodes[0].Hostname != "host-a" {
		t.Errorf("nodes = %#v, want only host-a", got.OnpremiseInfraModel.Nodes)
	}
}

func saveSoftware(t *testing.T, connID string, status string, data string) {
	t.Helper()

	if err := db.DB.Create(&model.SavedSoftwareInfo{ConnectionID: connID, SoftwareData: data, Status: status}).Error; err != nil {
		t.Fatal(err)
	}
}

func TestGetSoftwareInfoSourceGroupRefined_OnlyCollected(t *testing.T) {
	openTestDB(t)
	conns := append(mixedCSPConns(),
		testConn("vm-ip2", cspGroup, common.ResourceTypeVM, "198.51.100.14"),
		testConn("vm-failed", cspGroup, common.ResourceTypeVM, "198.51.100.15"),
	)
	createGroup(t, cspGroup, conns)

	raw, err := json.Marshal(software.Software{})
	if err != nil {
		t.Fatal(err)
	}
	saveSoftware(t, "vm-ip", "success", string(raw))
	saveSoftware(t, "vm-failed", "failed", "")
	// Left by an agent import on a connection without an agent before such
	// imports were refused; it must not reach the refined output.
	saveSoftware(t, "vm-noip", "success", string(raw))

	rec := groupRequest(t, cspGroup.ID, GetSoftwareInfoSourceGroupRefined)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var got softwaremodel.SourceSoftwareModel
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, ci := range got.SourceSoftwareModel.ConnectionInfoList {
		ids = append(ids, ci.ConnectionId)
	}
	if want := []string{"vm-ip"}; !reflect.DeepEqual(ids, want) {
		t.Errorf("connections = %v, want %v", ids, want)
	}
}

// A group with nothing to refine answers with an empty list, not null.
func TestGetSoftwareInfoSourceGroupRefined_Empty(t *testing.T) {
	openTestDB(t)
	createGroup(t, cspGroup, []model.ConnectionInfo{testConn("bucket", cspGroup, common.ResourceTypeObjectStorage, "")})

	rec := groupRequest(t, cspGroup.ID, GetSoftwareInfoSourceGroupRefined)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"connection_info_list": []`) {
		t.Errorf("body = %s, want an empty connection_info_list", rec.Body.String())
	}
}

// The error of an import is only part of the response; it must not become a
// column of the stored records.
func TestSavedInfoErrorMessageNotStored(t *testing.T) {
	openTestDB(t)

	for _, m := range []any{
		&model.SavedInfraInfo{}, &model.SavedSoftwareInfo{}, &model.SavedKubernetesInfo{},
		&model.SavedHelmInfo{}, &model.SavedDataInfo{},
	} {
		if !db.DB.Migrator().HasTable(m) {
			t.Fatalf("%T: table missing", m)
		}
		if db.DB.Migrator().HasColumn(m, "error_message") {
			t.Errorf("%T: error_message must not be a column", m)
		}
	}

	if err := db.DB.Create(&model.SavedInfraInfo{ConnectionID: "c1", Status: "failed", ErrorMessage: "boom"}).Error; err != nil {
		t.Fatal(err)
	}
	var rec model.SavedInfraInfo
	if err := db.DB.First(&rec, "connection_id = ?", "c1").Error; err != nil {
		t.Fatal(err)
	}
	if rec.ErrorMessage != "" {
		t.Errorf("error_message read back as %q, want it unstored", rec.ErrorMessage)
	}
}

// An fs group holds hosts, so once their infra is collected they are nodes of
// the refined infra as they were before the node selection was added.
func TestGetInfraInfoSourceGroupRefined_FSHostsAreNodes(t *testing.T) {
	openTestDB(t)
	conns := []model.ConnectionInfo{
		testConn("fs-a", fsGroup, "", "192.0.2.20"),
		testConn("fs-b", fsGroup, "", "192.0.2.21"),
	}
	createGroup(t, fsGroup, conns)
	saveInfra(t, "fs-a", "success", hostInfra("fs-host-a"), "")

	rec := groupRequest(t, fsGroup.ID, GetInfraInfoSourceGroupRefined)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var got model.RefinedOnpremiseInfraModel
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.OnpremiseInfraModel.Nodes) != 1 || got.OnpremiseInfraModel.Nodes[0].Hostname != "fs-host-a" {
		t.Errorf("nodes = %#v, want only fs-host-a", got.OnpremiseInfraModel.Nodes)
	}
}
