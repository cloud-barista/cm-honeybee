package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"

	inframodel "github.com/cloud-barista/cm-beetle/imdl/on-premise-model"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/infra"
	"github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/cloud-barista/cm-honeybee/server/dao"
	"github.com/cloud-barista/cm-honeybee/server/lib/spider"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
)

var nlbGroup = &model.SourceGroup{ID: "sg-nlb", Name: "sg-nlb", Type: common.SourceGroupTypeCSP, ProviderName: "AWS"}

const nlbARN = "arn:aws:elasticloadbalancing:ap-northeast-2:000000000000:loadbalancer/net/web-nlb/0123456789abcdef"

// awsNLB is an NLB the way the aws driver reports it: Type is always PUBLIC,
// the real scheme is in KeyValueList, the VPC has only a SystemId and the
// backend VMs come with no name.
func awsNLB(scheme string) spider.NLBInfo {
	return spider.NLBInfo{
		IId:    spider.IID{NameId: "web-nlb", SystemId: nlbARN},
		VpcIID: spider.IID{SystemId: "vpc-0a1b"},
		Type:   "PUBLIC",
		Scope:  "region",
		Listener: spider.ListenerInfo{
			Protocol: "tcp", IP: "", Port: "80",
			DNSName: "web-nlb-0123.elb.example.com",
		},
		VMGroup: spider.VMGroupInfo{
			Protocol: "TCP", Port: "8080",
			VMs: []spider.IID{{SystemId: "i-0aaa"}, {NameId: "web-2", SystemId: "i-0bbb"}, {}},
		},
		HealthChecker: spider.HealthCheckerInfo{Protocol: "tcp", Port: "8080", Interval: 30, Timeout: -1, Threshold: 3},
		CreatedTime:   "2026-11-01T09:30:00Z",
		KeyValueList:  []spider.KeyValue{{Key: "Scheme", Value: scheme}, {Key: "State", Value: "{Code:active}"}},
	}
}

// stubNLBCSP stands in for the NLB calls of the CSP. GetNLB answers by
// SystemId only, as the aws driver does. It returns how many times each call
// was made.
func stubNLBCSP(t *testing.T, nlbs []spider.NLBInfo, listErr error) map[string]int {
	t.Helper()

	calls := map[string]int{}
	origGet, origList := getCSPNLB, listCSPNLB
	t.Cleanup(func() { getCSPNLB, listCSPNLB = origGet, origList })

	getCSPNLB = func(_ spider.Conn, iid spider.IID) (*spider.NLBInfo, error) {
		calls["get"]++
		for i := range nlbs {
			if iid.SystemId != "" && nlbs[i].IId.SystemId == iid.SystemId {
				n := nlbs[i]
				return &n, nil
			}
		}
		return nil, errors.New("LoadBalancerNotFound")
	}
	listCSPNLB = func(_ spider.Conn) ([]spider.NLBInfo, error) {
		calls["list"]++
		if listErr != nil {
			return nil, listErr
		}
		return nlbs, nil
	}

	return calls
}

var testCSPConn = spider.Conn{Provider: "AWS", Region: "ap-northeast-2"}

func TestMatchNLB(t *testing.T) {
	cspOnly := spider.NLBInfo{IId: spider.IID{NameId: "", SystemId: "lb-cspOnly"}}
	named := spider.NLBInfo{IId: spider.IID{NameId: "web-nlb", SystemId: nlbARN}}
	listing := []spider.NLBInfo{cspOnly, named}

	tests := []struct {
		name       string
		resourceID string
		wantSystem string
	}{
		{"by system id", nlbARN, nlbARN},
		{"CSP-only by system id", "lb-cspOnly", "lb-cspOnly"},
		{"by name id", "web-nlb", nlbARN},
		{"unknown id", "nope", ""},
		// An empty id must not match the empty NameId of a CSP-only NLB.
		{"empty id", "", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := matchNLB(listing, tc.resourceID)
			if tc.wantSystem == "" {
				if got != nil {
					t.Fatalf("matched %+v, want no match", got.IId)
				}
				return
			}
			if got == nil || got.IId.SystemId != tc.wantSystem {
				t.Fatalf("matched %+v, want system id %q", got, tc.wantSystem)
			}
		})
	}
}

func TestFindNLBByID(t *testing.T) {
	nlbs := []spider.NLBInfo{awsNLB("internet-facing")}

	t.Run("get answers", func(t *testing.T) {
		calls := stubNLBCSP(t, nlbs, nil)
		got, err := findNLBByID(testCSPConn, nlbARN)
		if err != nil || got.IId.SystemId != nlbARN {
			t.Fatalf("got %+v, %v", got, err)
		}
		if calls["list"] != 0 {
			t.Errorf("listed %d times, want none when get finds the NLB", calls["list"])
		}
	})

	t.Run("name falls back to the listing", func(t *testing.T) {
		calls := stubNLBCSP(t, nlbs, nil)
		got, err := findNLBByID(testCSPConn, "web-nlb")
		if err != nil || got.IId.SystemId != nlbARN {
			t.Fatalf("got %+v, %v", got, err)
		}
		if calls["get"] != 1 || calls["list"] != 1 {
			t.Errorf("calls = %v, want one get and one list", calls)
		}
	})

	t.Run("get answering another NLB is not taken", func(t *testing.T) {
		stubNLBCSP(t, nlbs, nil)
		getCSPNLB = func(_ spider.Conn, _ spider.IID) (*spider.NLBInfo, error) {
			return &spider.NLBInfo{IId: spider.IID{SystemId: "lb-other"}}, nil
		}
		got, err := findNLBByID(testCSPConn, nlbARN)
		if err != nil || got.IId.SystemId != nlbARN {
			t.Fatalf("got %+v, %v, want the listed NLB", got, err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		stubNLBCSP(t, nlbs, nil)
		_, err := findNLBByID(testCSPConn, "lb-gone")
		if err == nil || !strings.Contains(err.Error(), "nlb 'lb-gone' was not found in aws region 'ap-northeast-2'") {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("list fails", func(t *testing.T) {
		stubNLBCSP(t, nlbs, errors.New("AccessDenied"))
		_, err := findNLBByID(testCSPConn, "lb-gone")
		if err == nil || !strings.Contains(err.Error(), "get: LoadBalancerNotFound, list: AccessDenied") {
			t.Fatalf("err = %v", err)
		}
	})
}

// Registration and refresh check an nlb connection through the driver, where
// they used to answer "unsupported resource_type: nlb".
func TestCheckCSPResource_NLB(t *testing.T) {
	stubNLBCSP(t, []spider.NLBInfo{awsNLB("internet-facing")}, nil)

	ci := &model.ConnectionInfo{ID: "nlb", ResourceType: common.ResourceTypeNLB, ResourceID: nlbARN}
	if err := checkCSPResource(testCSPConn, ci); err != nil {
		t.Fatalf("check = %v, want success", err)
	}

	ci.ResourceID = "lb-gone"
	if err := checkCSPResource(testCSPConn, ci); err == nil || !strings.Contains(err.Error(), "was not found") {
		t.Fatalf("check = %v, want not found", err)
	}
}

func TestNormalizeNLBType(t *testing.T) {
	tests := []struct {
		typ    string
		scheme string
		want   string
	}{
		{"PUBLIC", "", "PUBLIC"},
		{"public", "", "PUBLIC"},
		{" INTERNAL ", "", "INTERNAL"},
		{"PRIVATE", "", "INTERNAL"},        // alibaba intranet
		{"EXTERNAL", "", "PUBLIC"},         // gcp forwarding rule scheme
		{"PUBLIC", "internal", "INTERNAL"}, // aws reports PUBLIC for every NLB
		{"PUBLIC", "internet-facing", "PUBLIC"},
		{"", "", ""},
		{"GLOBAL", "", ""},
	}

	for _, tc := range tests {
		n := &spider.NLBInfo{Type: tc.typ}
		if tc.scheme != "" {
			n.KeyValueList = []spider.KeyValue{{Key: "Scheme", Value: tc.scheme}}
		}
		if got := normalizeNLBType(n); got != tc.want {
			t.Errorf("type %q scheme %q: got %q, want %q", tc.typ, tc.scheme, got, tc.want)
		}
	}
}

func TestNLBInfoToNLB(t *testing.T) {
	n := awsNLB("internal")
	got := nlbInfoToNLB(&n)

	want := model.CSPNLBRecord{
		SystemID: nlbARN,
		Name:     "web-nlb",
		VPC:      "vpc-0a1b",
		Type:     "INTERNAL",
		Scope:    "REGION",
		Listener: model.CSPNLBListener{Protocol: "TCP", Port: "80", DNSName: "web-nlb-0123.elb.example.com"},
		VMGroup: model.CSPNLBVMGroup{Protocol: "TCP", Port: "8080", VMs: []model.CSPNLBVM{
			{SystemID: "i-0aaa"}, {SystemID: "i-0bbb", Name: "web-2"},
		}},
		HealthChecker: model.CSPNLBHealthChecker{Protocol: "TCP", Port: "8080", Interval: 30, Timeout: 0, Threshold: 3},
		CreatedTime:   "2026-11-01T09:30:00Z",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("record =\n%+v\nwant\n%+v", got, want)
	}

	n.IId.NameId = ""
	n.CreatedTime = "0001-01-01T00:00:00Z"
	n.VMGroup.VMs = nil
	got = nlbInfoToNLB(&n)
	if got.Name != nlbARN || got.CreatedTime != "" {
		t.Errorf("name %q created %q, want the SystemId and no time", got.Name, got.CreatedTime)
	}
	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"vms":[]`) {
		t.Errorf("an NLB without backend VMs must store an empty list: %s", raw)
	}
}

func TestCSPNLBToNlbProperty(t *testing.T) {
	vmIPs := map[string]string{"i-0aaa": "192.0.2.21"}

	t.Run("public", func(t *testing.T) {
		n := awsNLB("internet-facing")
		rec := nlbInfoToNLB(&n)
		got := cspNLBToNlbProperty(&rec, "AWS", vmIPs)

		want := inframodel.NlbProperty{
			Software: "aws-nlb",
			Listener: inframodel.NlbListenerProperty{BindAddress: "*", Port: 80, Protocol: "tcp"},
			Backend: inframodel.NlbBackendProperty{
				Name:     "web-nlb",
				Protocol: "tcp",
				Servers: []inframodel.NlbServerProperty{
					{Name: "i-0aaa", IP: "192.0.2.21", Port: 8080},
					{Name: "web-2", IP: "", Port: 8080},
				},
			},
			HealthCheck: inframodel.NlbHealthCheckProperty{Enabled: true, Protocol: "tcp", Port: 8080, Interval: 30, Threshold: 3},
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("nlb =\n%+v\nwant\n%+v", got, want)
		}
	})

	t.Run("internal", func(t *testing.T) {
		n := awsNLB("internal")
		n.Listener.IP = " 192.0.2.50, 192.0.2.51"
		rec := nlbInfoToNLB(&n)
		if got := cspNLBToNlbProperty(&rec, "aws", vmIPs).Listener.BindAddress; got != "192.0.2.50" {
			t.Errorf("bind address = %q, want the first listener IP", got)
		}

		rec.Listener.IP = ""
		if got := cspNLBToNlbProperty(&rec, "aws", vmIPs).Listener.BindAddress; got != "web-nlb-0123.elb.example.com" {
			t.Errorf("bind address = %q, want the DNS name when there is no IP", got)
		}
	})

	t.Run("unknown type and provider", func(t *testing.T) {
		n := awsNLB("")
		n.Type = ""
		n.Listener.IP = "198.51.100.5"
		rec := nlbInfoToNLB(&n)
		got := cspNLBToNlbProperty(&rec, "", nil)
		if got.Listener.BindAddress != "" || got.Software != "csp-nlb" {
			t.Errorf("bind %q software %q, want empty and csp-nlb", got.Listener.BindAddress, got.Software)
		}
		if got.Backend.Servers[0].IP != "" {
			t.Errorf("server = %+v, want no IP without CSP VMs", got.Backend.Servers[0])
		}
	})
}

// nlbGroupConns is a csp group with a VM behind the NLB, an NLB connection and
// a bucket.
func nlbGroupConns() []model.ConnectionInfo {
	conns := []model.ConnectionInfo{
		testConn("vm-web", nlbGroup, common.ResourceTypeVM, "198.51.100.10"),
		testConn("nlb", nlbGroup, common.ResourceTypeNLB, ""),
		testConn("bucket", nlbGroup, common.ResourceTypeObjectStorage, ""),
	}
	conns[0].ResourceID = "i-0aaa"
	conns[1].ResourceID = nlbARN
	conns[2].ResourceID = "bucket"
	return conns
}

// importNLBThrough makes the infra import collect through the stubbed NLB
// calls, as refreshCSPConnection does once it has the CSP connection.
func importNLBThrough(t *testing.T) {
	t.Helper()

	orig := refreshCSP
	t.Cleanup(func() { refreshCSP = orig })
	refreshCSP = func(sg *model.SourceGroup, ci *model.ConnectionInfo, opts importInfraOptions) error {
		return collectCSPResource(testCSPConn, sg, ci, opts)
	}
}

// The infra import of an nlb connection stores the NLB, and GET answers with
// it; when the NLB is gone a later import fails, the record is marked failed
// with the last NLB kept for GET, and refined and the group GET leave it out.
func TestDoImportInfra_NLB(t *testing.T) {
	openTestDB(t)
	createGroup(t, nlbGroup, nlbGroupConns())
	importNLBThrough(t)
	stubNLBCSP(t, []spider.NLBInfo{awsNLB("internet-facing")}, nil)

	if _, err := doImportInfra("nlb", importInfraOptions{}); err != nil {
		t.Fatalf("import = %v", err)
	}
	saved, err := dao.SavedNLBInfoGet("nlb")
	if err != nil || saved.Status != model.ConnectionInfoStatusSuccess {
		t.Fatalf("saved = %+v, %v", saved, err)
	}
	if ci := getConnection(t, "nlb"); ci.ConnectionStatus != model.ConnectionInfoStatusSuccess {
		t.Errorf("connection status = %q (%q)", ci.ConnectionStatus, ci.ConnectionFailedMessage)
	}

	rec := connRequest(t, http.MethodGet, nlbGroup.ID, "nlb", GetNLBInfo)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var got model.CSPNLBRecord
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.SystemID != nlbARN || got.Type != "PUBLIC" || len(got.VMGroup.VMs) != 2 {
		t.Errorf("GET nlb = %s", rec.Body.String())
	}

	stubNLBCSP(t, nil, nil)
	if _, err := doImportInfra("nlb", importInfraOptions{}); err == nil {
		t.Fatal("import of a deleted NLB must fail")
	}
	saved, err = dao.SavedNLBInfoGet("nlb")
	if err != nil || saved.Status != model.ConnectionInfoStatusFailed || !strings.Contains(saved.NLBData, nlbARN) {
		t.Fatalf("saved = %+v, %v, want failed with the NLB kept", saved, err)
	}
	if ci := getConnection(t, "nlb"); ci.ConnectionStatus != model.ConnectionInfoStatusFailed ||
		!strings.Contains(ci.ConnectionFailedMessage, "was not found") {
		t.Errorf("connection status = %q (%q), want failed with the reason", ci.ConnectionStatus, ci.ConnectionFailedMessage)
	}
	if rec := connRequest(t, http.MethodGet, nlbGroup.ID, "nlb", GetNLBInfo); rec.Code != http.StatusOK ||
		!strings.Contains(rec.Body.String(), nlbARN) {
		t.Errorf("GET after a failed import: status = %d, body = %s, want the last NLB", rec.Code, rec.Body.String())
	}
	if tryGetCSPNLB("nlb") != nil {
		t.Error("a failed record must not be refined")
	}
	rec = connRequest(t, http.MethodGet, nlbGroup.ID, "nlb", GetInfraInfoRefined)
	if n := len(refinedInfraBody(t, rec.Body.Bytes()).OnpremiseInfraModel.NLBs); rec.Code != http.StatusOK || n != 0 {
		t.Errorf("refined after a failed import: status = %d, nlbs = %d, want none", rec.Code, n)
	}
}

// A first import that fails leaves a failed record with no NLB, which GET
// refuses.
func TestMarkCSPResourceFailed_NLBFirstImport(t *testing.T) {
	openTestDB(t)
	createGroup(t, nlbGroup, nlbGroupConns())
	importNLBThrough(t)
	stubNLBCSP(t, nil, errors.New("AccessDenied"))

	if _, err := doImportInfra("nlb", importInfraOptions{}); err == nil {
		t.Fatal("import must fail")
	}
	saved, err := dao.SavedNLBInfoGet("nlb")
	if err != nil || saved.Status != model.ConnectionInfoStatusFailed || saved.NLBData != "" {
		t.Fatalf("saved = %+v, %v, want an empty failed record", saved, err)
	}
	rec := connRequest(t, http.MethodGet, nlbGroup.ID, "nlb", GetNLBInfo)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "Status = failed") {
		t.Errorf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestGetNLBInfo_NotNLB(t *testing.T) {
	openTestDB(t)
	createGroup(t, nlbGroup, nlbGroupConns())

	rec := connRequest(t, http.MethodGet, nlbGroup.ID, "bucket", GetNLBInfo)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "not available for resource_type 'object_storage'") {
		t.Errorf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	rec = connRequest(t, http.MethodGet, nlbGroup.ID, "nlb", GetNLBInfo)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "has not been collected") {
		t.Errorf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

// webVMWithHAProxy is the VM behind the NLB as the agent and the CSP collected
// it, running an HAProxy of its own.
func saveWebVM(t *testing.T) {
	t.Helper()

	host := hostInfra("web-1")
	host.Compute.OS.Node.Machineid = "m-web-1"
	host.HAProxy = infra.HAProxy{
		Frontends: []infra.HAProxyFrontend{{Name: "fe", Bind: "*:8080", Options: map[string]string{}}},
		Backends: []infra.HAProxyBackend{{
			Name:    "app",
			Options: map[string]string{},
			Servers: []infra.HAProxyServer{{Name: "app-1", Address: "192.0.2.31:9000"}},
		}},
	}
	csp, err := json.Marshal(infra.CSPInfo{ID: "i-0aaa", Name: "web-1", PrivateIP: "192.0.2.21"})
	if err != nil {
		t.Fatal(err)
	}
	saveInfra(t, "vm-web", "success", host, string(csp))
}

// The group refined infra holds the HAProxy NLB found on a node and the CSP
// NLB side by side, the CSP one with its backend VM's private IP; the group
// GET nlb lists only the collected NLBs.
func TestGetInfraInfoSourceGroupRefined_HAProxyAndCSPNLB(t *testing.T) {
	openTestDB(t)
	conns := append(nlbGroupConns(), testConn("nlb-failed", nlbGroup, common.ResourceTypeNLB, ""))
	createGroup(t, nlbGroup, conns)
	saveWebVM(t)
	n := awsNLB("internet-facing")
	if err := upsertSavedNLB("nlb", nlbInfoToNLB(&n)); err != nil {
		t.Fatal(err)
	}
	other := awsNLB("internal")
	other.IId = spider.IID{NameId: "old-nlb", SystemId: "lb-old"}
	if err := upsertSavedNLB("nlb-failed", nlbInfoToNLB(&other)); err != nil {
		t.Fatal(err)
	}
	if err := markCSPResourceFailed(&model.ConnectionInfo{ID: "nlb-failed", ResourceType: common.ResourceTypeNLB}); err != nil {
		t.Fatal(err)
	}

	rec := groupRequest(t, nlbGroup.ID, GetInfraInfoSourceGroupRefined)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	got := refinedInfraBody(t, rec.Body.Bytes()).OnpremiseInfraModel
	if len(got.Nodes) != 1 || got.Nodes[0].Hostname != "web-1" {
		t.Errorf("nodes = %+v, want only web-1", got.Nodes)
	}
	var sw []string
	for _, nlb := range got.NLBs {
		sw = append(sw, nlb.Software+"/"+nlb.Backend.Name)
	}
	if w := []string{"haproxy/app", "aws-nlb/web-nlb"}; !reflect.DeepEqual(sw, w) {
		t.Fatalf("nlbs = %v, want %v", sw, w)
	}
	csp := got.NLBs[1]
	if csp.HostMachineId != "" || csp.Listener.BindAddress != "*" || csp.Backend.Servers[0].IP != "192.0.2.21" {
		t.Errorf("csp nlb = %+v", csp)
	}

	rec = groupRequest(t, nlbGroup.ID, GetNLBInfoSourceGroup)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var list model.NLBInfoList
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Servers) != 1 || list.Servers[0].SystemID != nlbARN {
		t.Errorf("group GET nlb = %s, want only the collected web-nlb", rec.Body.String())
	}
}

// The refined infra of an nlb connection has no node and the one NLB, its
// backend servers given the private IPs of the group's CSP VMs.
func TestGetInfraInfoRefined_CSPNLB(t *testing.T) {
	openTestDB(t)
	createGroup(t, nlbGroup, nlbGroupConns())
	saveWebVM(t)
	n := awsNLB("internal")
	if err := upsertSavedNLB("nlb", nlbInfoToNLB(&n)); err != nil {
		t.Fatal(err)
	}

	rec := connRequest(t, http.MethodGet, nlbGroup.ID, "nlb", GetInfraInfoRefined)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	got := refinedInfraBody(t, rec.Body.Bytes()).OnpremiseInfraModel
	if len(got.Nodes) != 0 {
		t.Errorf("nodes = %+v, want none", got.Nodes)
	}
	if len(got.NLBs) != 1 {
		t.Fatalf("nlbs = %+v, want one", got.NLBs)
	}
	nlb := got.NLBs[0]
	if nlb.Software != "aws-nlb" || nlb.Listener.BindAddress != "web-nlb-0123.elb.example.com" {
		t.Errorf("nlb = %+v", nlb)
	}
	servers := []string{}
	for _, s := range nlb.Backend.Servers {
		servers = append(servers, s.Name+"="+s.IP)
	}
	if w := []string{"i-0aaa=192.0.2.21", "web-2="}; !reflect.DeepEqual(servers, w) {
		t.Errorf("servers = %v, want %v", servers, w)
	}

	rec = connRequest(t, http.MethodGet, nlbGroup.ID, "vm-web", GetInfraInfoRefined)
	if got := refinedInfraBody(t, rec.Body.Bytes()).OnpremiseInfraModel.NLBs; len(got) != 1 || got[0].Software != "haproxy" {
		t.Errorf("vm refined nlbs = %+v, want only its HAProxy", got)
	}
}
