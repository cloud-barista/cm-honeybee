package controller

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/cloud-barista/cm-honeybee/server/dao"
	"github.com/cloud-barista/cm-honeybee/server/lib/spider"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"github.com/labstack/echo/v4"
)

// fakeSSHPassword is a placeholder the tests store and read back; it is not
// a credential.
var fakeSSHPassword = "fake-" + "ssh-pw"

const azureVMID = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/RG-1/providers/Microsoft.Compute/virtualMachines/web-1"

// stubVMCSP stands in for the VM calls of the CSP. The account holds vms;
// GetVM answers getErr for every VM it does not hold by SystemId, and the
// listing fails with listErr when set. The calls made are counted.
func stubVMCSP(t *testing.T, vms []spider.VMInfo, getErr error, listErr error) map[string]int {
	t.Helper()

	calls := map[string]int{}
	origGet, origList := getCSPVM, listCSPVM
	t.Cleanup(func() { getCSPVM, listCSPVM = origGet, origList })

	getCSPVM = func(_ spider.Conn, id string) (*spider.VMInfo, error) {
		calls["get"]++
		for i := range vms {
			if vms[i].IId.SystemId == id {
				vm := vms[i]
				return &vm, nil
			}
		}
		return nil, getErr
	}
	listCSPVM = func(spider.Conn) ([]spider.VMInfo, error) {
		calls["list"]++
		if listErr != nil {
			return nil, listErr
		}
		return vms, nil
	}

	return calls
}

// A VM GetVM cannot read is missing only when the listing came back without
// it; when the listing fails too, or holds the VM under another spelling of
// its identifier, the GetVM error stays as it was.
func TestFindVMByID(t *testing.T) {
	held := []spider.VMInfo{
		{IId: spider.IID{NameId: "web-0", SystemId: "i-0123456789abcdef0"}},
		{IId: spider.IID{NameId: "web-1", SystemId: strings.ToLower(azureVMID)}},
	}
	getErr := errors.New("InvalidInstanceID.NotFound")

	t.Run("found by GetVM", func(t *testing.T) {
		calls := stubVMCSP(t, held, getErr, nil)
		vm, err := findVMByID(testCSPConn, "i-0123456789abcdef0")
		if err != nil || vm.IId.NameId != "web-0" {
			t.Fatalf("got %+v, %v; want web-0", vm, err)
		}
		if calls["list"] != 0 {
			t.Errorf("list calls = %d, want 0", calls["list"])
		}
	})

	t.Run("not found", func(t *testing.T) {
		stubVMCSP(t, held, getErr, nil)
		_, err := findVMByID(testCSPConn, "i-0000000000000dead")
		if !errors.Is(err, ErrCSPResourceNotFound) {
			t.Fatalf("err = %v, want ErrCSPResourceNotFound", err)
		}
		want := "vm 'i-0000000000000dead' was not found in aws region 'ap-northeast-2' (InvalidInstanceID.NotFound)"
		if err.Error() != want {
			t.Errorf("err = %q, want %q", err, want)
		}
	})

	t.Run("malformed id", func(t *testing.T) {
		stubVMCSP(t, held, errors.New("InvalidInstanceID.Malformed"), nil)
		if _, err := findVMByID(testCSPConn, "no-such-vm"); !errors.Is(err, ErrCSPResourceNotFound) {
			t.Fatalf("err = %v, want ErrCSPResourceNotFound", err)
		}
	})

	t.Run("list fails", func(t *testing.T) {
		stubVMCSP(t, held, errors.New("RequestTimeout"), errors.New("RequestTimeout"))
		_, err := findVMByID(testCSPConn, "i-0000000000000dead")
		if err == nil || errors.Is(err, ErrCSPResourceNotFound) || err.Error() != "RequestTimeout" {
			t.Fatalf("err = %v, want the GetVM error unmarked", err)
		}
	})

	t.Run("listed under another spelling", func(t *testing.T) {
		stubVMCSP(t, held, errors.New("ResourceNotFound"), nil)
		for _, id := range []string{azureVMID, "web-1", "WEB-0"} {
			_, err := findVMByID(testCSPConn, id)
			if err == nil || errors.Is(err, ErrCSPResourceNotFound) {
				t.Errorf("%s: err = %v, want the GetVM error unmarked", id, err)
			}
		}
	})
}

// Only a resource the CSP answered for is refused: a missing one of every
// kind and a bucket of another region. A failure to ask is not.
func TestCSPResourceRejected(t *testing.T) {
	stubBucketCSP(t, map[string]string{"near": "ap-northeast-3", "far": "ap-northeast-2"}, nil, true)
	stubNLBCSP(t, []spider.NLBInfo{awsNLB("internet-facing")}, nil)
	stubVMCSP(t, nil, errors.New("InvalidInstanceID.NotFound"), nil)

	_, bucketGone := checkBucket(osakaConn, "absent")
	_, bucketFar := checkBucket(osakaConn, "far")
	_, nlbGone := findNLBByID(testCSPConn, "lb-gone")
	_, vmGone := findVMByID(testCSPConn, "i-gone")

	for name, err := range map[string]error{"bucket": bucketGone, "far bucket": bucketFar, "nlb": nlbGone, "vm": vmGone} {
		if !cspResourceRejected(err) {
			t.Errorf("%s: %v not rejected", name, err)
		}
	}
	if !errors.Is(bucketFar, ErrBucketRegionMismatch) || errors.Is(bucketFar, ErrCSPResourceNotFound) {
		t.Errorf("far bucket: err = %v, want ErrBucketRegionMismatch only", bucketFar)
	}

	stubNLBCSP(t, nil, errors.New("AccessDenied"))
	_, nlbDenied := findNLBByID(testCSPConn, "lb-gone")
	for name, err := range map[string]error{
		"nil": nil, "nlb list denied": nlbDenied,
		"credential": errors.New("OpenBao is required"), "timeout": errors.New("context deadline exceeded"),
	} {
		if cspResourceRejected(err) {
			t.Errorf("%s: %v rejected", name, err)
		}
	}
}

// fakeBao stands in for OpenBao: the secrets of each connection by its ID.
type fakeBao map[string]map[string]string

// stubConnectionSide stands in for OpenBao, the CSP check and the agent run.
// check answers for the CSP; every agent run is recorded with the connection
// it was given.
func stubConnectionSide(t *testing.T, check func(*model.ConnectionInfo) error) (fakeBao, *[]model.ConnectionInfo, *int) {
	t.Helper()

	bao := fakeBao{}
	var agentRuns []model.ConnectionInfo
	checks := 0

	origStore, origHydrate, origCheck, origAgent, origKey := storeSecrets, hydrateSecrets, checkCSP, runCSPAgent, common.PubKey
	t.Cleanup(func() {
		storeSecrets, hydrateSecrets, checkCSP, runCSPAgent, common.PubKey = origStore, origHydrate, origCheck, origAgent, origKey
	})

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	common.PubKey = &key.PublicKey

	storeSecrets = func(ci *model.ConnectionInfo) error {
		if ci.Password == "" && (ci.PrivateKey == "" || ci.PrivateKey == "-") {
			return nil
		}
		bao[ci.ID] = map[string]string{"password": ci.Password, "private_key": ci.PrivateKey}
		ci.Password = ""
		ci.PrivateKey = ""
		return nil
	}
	hydrateSecrets = func(ci *model.ConnectionInfo) error {
		data, ok := bao[ci.ID]
		if !ok {
			return nil
		}
		ci.Password = data["password"]
		ci.PrivateKey = data["private_key"]
		if ci.PrivateKey == "" {
			ci.PrivateKey = "-"
		}
		return nil
	}
	checkCSP = func(_ *model.SourceGroup, ci *model.ConnectionInfo) error {
		checks++
		return check(ci)
	}
	runCSPAgent = func(ci model.ConnectionInfo) error {
		agentRuns = append(agentRuns, ci)
		return nil
	}

	return bao, &agentRuns, &checks
}

func jsonConnRequest(t *testing.T, method string, sgID string, connID string, body any, handler echo.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()

	b, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(method, "/", bytes.NewReader(b))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(req, rec)
	c.SetParamNames("sgId", "connId")
	c.SetParamValues(sgID, connID)
	if err := handler(c); err != nil {
		t.Fatal(err)
	}
	return rec
}

func groupConnections(t *testing.T, sgID string) []model.ConnectionInfo {
	t.Helper()

	conns, err := dao.ConnectionInfoGetList(&model.ConnectionInfo{SourceGroupID: sgID}, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	return *conns
}

// Registration refuses a resource the CSP does not have, a bucket of another
// region, and SSH fields on a resource other than a VM, writing no connection.
func TestCreateCSPConnection_Refused(t *testing.T) {
	openTestDB(t)
	createGroup(t, osakaGroup, nil)
	stubBucketCSP(t, map[string]string{"near": "ap-northeast-3", "far": "ap-northeast-2"}, nil, true)
	stubVMCSP(t, nil, errors.New("InvalidInstanceID.NotFound"), nil)
	_, _, checks := stubConnectionSide(t, func(ci *model.ConnectionInfo) error { return checkCSPResource(osakaConn, ci) })

	tests := []struct {
		name string
		req  model.CreateConnectionInfoReq
		want string
	}{
		{"vm gone", model.CreateConnectionInfoReq{Name: "vm", ResourceType: "vm", ResourceID: "i-gone"},
			"vm 'i-gone' was not found in aws region 'ap-northeast-3'"},
		{"bucket gone", model.CreateConnectionInfoReq{Name: "b1", ResourceType: "object_storage", ResourceID: "absent"},
			"bucket 'absent' was not found"},
		{"bucket far", model.CreateConnectionInfoReq{Name: "b2", ResourceType: "object_storage", ResourceID: "far"},
			"bucket 'far' is in region 'ap-northeast-2', source group region is 'ap-northeast-3'"},
		{"bucket with ssh", model.CreateConnectionInfoReq{Name: "b3", ResourceType: "object_storage", ResourceID: "near",
			IPAddress: "198.51.100.20", User: "ubuntu"},
			"ip_address, user apply to a csp vm connection only; a csp object_storage connection has no host"},
	}
	for _, tc := range tests {
		rec := jsonConnRequest(t, http.MethodPost, osakaGroup.ID, "", tc.req, CreateConnectionInfo)
		if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), tc.want) {
			t.Errorf("%s: %d %s, want 400 with %q", tc.name, rec.Code, rec.Body.String(), tc.want)
		}
	}
	if conns := groupConnections(t, osakaGroup.ID); len(conns) != 0 {
		t.Errorf("connections = %d, want none", len(conns))
	}
	if *checks != 3 {
		t.Errorf("CSP checks = %d, want 3 (the SSH fields are refused before)", *checks)
	}

	rec := jsonConnRequest(t, http.MethodPost, osakaGroup.ID, "",
		model.CreateConnectionInfoReq{Name: "b4", ResourceType: "object_storage", ResourceID: "near"}, CreateConnectionInfo)
	if rec.Code != http.StatusOK {
		t.Fatalf("near: %d %s, want 200", rec.Code, rec.Body.String())
	}
	conns := groupConnections(t, osakaGroup.ID)
	if len(conns) != 1 || conns[0].ConnectionStatus != model.ConnectionInfoStatusSuccess {
		t.Errorf("connections = %+v, want b4 with success", conns)
	}
}

// A resource the CSP could not be asked about is registered as before, failed
// with the reason, and the CSP is asked once, not again by the refresh.
func TestCreateCSPConnection_TransientIsRegisteredFailed(t *testing.T) {
	openTestDB(t)
	createGroup(t, cspGroup, nil)
	stubVMCSP(t, nil, errors.New("RequestTimeout"), errors.New("RequestTimeout"))
	_, agentRuns, checks := stubConnectionSide(t, func(ci *model.ConnectionInfo) error { return checkCSPResource(testCSPConn, ci) })

	rec := jsonConnRequest(t, http.MethodPost, cspGroup.ID, "",
		model.CreateConnectionInfoReq{Name: "vm", ResourceType: "vm", ResourceID: "i-0123456789abcdef0"}, CreateConnectionInfo)
	if rec.Code != http.StatusOK {
		t.Fatalf("%d %s, want 200", rec.Code, rec.Body.String())
	}
	conns := groupConnections(t, cspGroup.ID)
	if len(conns) != 1 {
		t.Fatalf("connections = %d, want 1", len(conns))
	}
	if conns[0].ConnectionStatus != model.ConnectionInfoStatusFailed || conns[0].ConnectionFailedMessage != "RequestTimeout" {
		t.Errorf("status = %q %q, want failed with RequestTimeout", conns[0].ConnectionStatus, conns[0].ConnectionFailedMessage)
	}
	if *checks != 1 {
		t.Errorf("CSP checks = %d, want 1", *checks)
	}
	if len(*agentRuns) != 0 {
		t.Errorf("agent runs = %d, want none without ip_address", len(*agentRuns))
	}
}

// A csp VM registered without SSH gets it by an update: the address and user
// land on the row, the key in OpenBao, and the agent is installed with them.
// A later update replaces what it sends and keeps what it leaves empty.
func TestUpdateCSPConnection_SSH(t *testing.T) {
	openTestDB(t)
	createGroup(t, cspGroup, []model.ConnectionInfo{{ID: "vm", Name: "vm", SourceGroupID: cspGroup.ID,
		ResourceType: common.ResourceTypeVM, ResourceID: "i-0123456789abcdef0"}})
	bao, agentRuns, checks := stubConnectionSide(t, func(*model.ConnectionInfo) error { return nil })

	rec := jsonConnRequest(t, http.MethodPut, cspGroup.ID, "vm", model.CreateConnectionInfoReq{
		IPAddress: "198.51.100.30", User: "ubuntu", PrivateKey: "test-key-1",
	}, UpdateConnectionInfo)
	if rec.Code != http.StatusOK {
		t.Fatalf("add: %d %s, want 200", rec.Code, rec.Body.String())
	}
	got := getConnection(t, "vm")
	if got.IPAddress != "198.51.100.30" || got.SSHPort != "22" || got.User != "ubuntu" || got.PrivateKey != "" ||
		got.ResourceID != "i-0123456789abcdef0" {
		t.Errorf("row = %+v, want ip, port 22, user and no key on the row", got)
	}
	if bao["vm"]["private_key"] != "test-key-1" {
		t.Errorf("stored key = %q, want test-key-1", bao["vm"]["private_key"])
	}
	if len(*agentRuns) != 1 || (*agentRuns)[0].IPAddress != "198.51.100.30" || (*agentRuns)[0].PrivateKey != "test-key-1" {
		t.Fatalf("agent runs = %+v, want one with the new ip and key", *agentRuns)
	}
	if got.AgentStatus != model.ConnectionInfoStatusSuccess {
		t.Errorf("agent_status = %q, want success", got.AgentStatus)
	}
	if *checks != 1 {
		t.Errorf("CSP checks = %d, want 1 (the refresh only, the resource did not change)", *checks)
	}

	rec = jsonConnRequest(t, http.MethodPut, cspGroup.ID, "vm", model.CreateConnectionInfoReq{
		IPAddress: "198.51.100.31", SSHPort: "2222", Password: fakeSSHPassword,
	}, UpdateConnectionInfo)
	if rec.Code != http.StatusOK {
		t.Fatalf("change: %d %s, want 200", rec.Code, rec.Body.String())
	}
	got = getConnection(t, "vm")
	if got.IPAddress != "198.51.100.31" || got.SSHPort != "2222" || got.User != "ubuntu" {
		t.Errorf("row = %+v, want the new ip and port, the user kept", got)
	}
	if bao["vm"]["private_key"] != "test-key-1" || bao["vm"]["password"] != fakeSSHPassword {
		t.Errorf("stored = %v, want the key kept and the password added", bao["vm"])
	}
	if last := (*agentRuns)[len(*agentRuns)-1]; last.IPAddress != "198.51.100.31" || last.SSHPort != "2222" {
		t.Errorf("last agent run = %+v, want the new ip and port", last)
	}

	rec = jsonConnRequest(t, http.MethodPut, cspGroup.ID, "vm", model.CreateConnectionInfoReq{User: "ec2-user"}, UpdateConnectionInfo)
	if rec.Code != http.StatusOK {
		t.Fatalf("user only: %d %s, want 200", rec.Code, rec.Body.String())
	}
	got = getConnection(t, "vm")
	if got.IPAddress != "198.51.100.31" || got.SSHPort != "2222" || got.User != "ec2-user" {
		t.Errorf("row = %+v, want the user changed and the rest kept", got)
	}
	if bao["vm"]["private_key"] != "test-key-1" || bao["vm"]["password"] != fakeSSHPassword {
		t.Errorf("stored = %v, want both secrets kept", bao["vm"])
	}
}

// Updates refused before anything is written: SSH fields on a csp resource
// other than a VM, SSH fields without an address, a value that does not parse,
// and a resource the CSP does not have.
func TestUpdateCSPConnection_Refused(t *testing.T) {
	openTestDB(t)
	createGroup(t, cspGroup, []model.ConnectionInfo{
		{ID: "bucket", Name: "bucket", SourceGroupID: cspGroup.ID, ResourceType: common.ResourceTypeObjectStorage, ResourceID: "near"},
		{ID: "vm", Name: "vm", SourceGroupID: cspGroup.ID, ResourceType: common.ResourceTypeVM, ResourceID: "i-0123456789abcdef0"},
	})
	stubVMCSP(t, []spider.VMInfo{{IId: spider.IID{SystemId: "i-0123456789abcdef0"}}}, errors.New("InvalidInstanceID.NotFound"), nil)
	stubNLBCSP(t, nil, nil)
	_, agentRuns, _ := stubConnectionSide(t, func(ci *model.ConnectionInfo) error { return checkCSPResource(testCSPConn, ci) })

	tests := []struct {
		name   string
		connID string
		req    model.CreateConnectionInfoReq
		want   string
	}{
		{"ssh on bucket", "bucket", model.CreateConnectionInfoReq{IPAddress: "198.51.100.40", PrivateKey: "test-key"},
			"ip_address, private_key apply to a csp vm connection only; a csp object_storage connection has no host"},
		{"ssh with a type turned to rdbms", "vm", model.CreateConnectionInfoReq{ResourceType: "rdbms", User: "ubuntu"},
			"user apply to a csp vm connection only; a csp rdbms connection"},
		{"no address", "vm", model.CreateConnectionInfoReq{User: "ubuntu", PrivateKey: "test-key"},
			"ip_address is empty (required to set user, private_key)"},
		{"no credential", "vm", model.CreateConnectionInfoReq{IPAddress: "198.51.100.40", User: "ubuntu"},
			"password or private_key must be provided when ip_address is provided"},
		{"bad address", "vm", model.CreateConnectionInfoReq{IPAddress: "198.51.100", User: "ubuntu", PrivateKey: "test-key"},
			"ip_address is invalid"},
		{"bad port", "vm", model.CreateConnectionInfoReq{IPAddress: "198.51.100.40", SSHPort: "70000", User: "ubuntu", PrivateKey: "test-key"},
			"port value is invalid"},
		{"vm gone", "vm", model.CreateConnectionInfoReq{ResourceID: "i-gone"},
			"vm 'i-gone' was not found in aws region 'ap-northeast-2'"},
		{"type turned to nlb", "vm", model.CreateConnectionInfoReq{ResourceType: "nlb"},
			"nlb 'i-0123456789abcdef0' was not found"},
	}
	for _, tc := range tests {
		rec := jsonConnRequest(t, http.MethodPut, cspGroup.ID, tc.connID, tc.req, UpdateConnectionInfo)
		if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), tc.want) {
			t.Errorf("%s: %d %s, want 400 with %q", tc.name, rec.Code, rec.Body.String(), tc.want)
		}
	}

	if got := getConnection(t, "vm"); got.ResourceType != common.ResourceTypeVM || got.ResourceID != "i-0123456789abcdef0" ||
		got.IPAddress != "" || got.User != "" {
		t.Errorf("vm row = %+v, want unchanged", got)
	}
	if got := getConnection(t, "bucket"); got.IPAddress != "" {
		t.Errorf("bucket row = %+v, want unchanged", got)
	}
	if len(*agentRuns) != 0 {
		t.Errorf("agent runs = %d, want none", len(*agentRuns))
	}
}

// Pointing a connection at another resource the CSP answers for is checked
// once, and the refresh takes that answer. A VM turned into another resource
// type leaves its SSH fields behind.
func TestUpdateCSPConnection_ResourceChange(t *testing.T) {
	openTestDB(t)
	createGroup(t, cspGroup, []model.ConnectionInfo{{ID: "vm", Name: "vm", SourceGroupID: cspGroup.ID,
		ResourceType: common.ResourceTypeVM, ResourceID: "i-0123456789abcdef0",
		IPAddress: "198.51.100.50", SSHPort: "22", User: "ubuntu"}})
	stubVMCSP(t, nil, errors.New("RequestTimeout"), errors.New("RequestTimeout"))
	stubNLBCSP(t, []spider.NLBInfo{awsNLB("internet-facing")}, nil)
	bao, agentRuns, checks := stubConnectionSide(t, func(ci *model.ConnectionInfo) error { return checkCSPResource(testCSPConn, ci) })
	bao["vm"] = map[string]string{"private_key": "test-key"}

	rec := jsonConnRequest(t, http.MethodPut, cspGroup.ID, "vm",
		model.CreateConnectionInfoReq{ResourceID: "i-0fedcba9876543210"}, UpdateConnectionInfo)
	if rec.Code != http.StatusOK {
		t.Fatalf("transient: %d %s, want 200", rec.Code, rec.Body.String())
	}
	got := getConnection(t, "vm")
	if got.ResourceID != "i-0fedcba9876543210" || got.ConnectionStatus != model.ConnectionInfoStatusFailed ||
		got.ConnectionFailedMessage != "RequestTimeout" {
		t.Errorf("row = %+v, want the new id, failed with RequestTimeout", got)
	}
	if *checks != 1 {
		t.Errorf("CSP checks = %d, want 1", *checks)
	}
	if len(*agentRuns) != 1 {
		t.Errorf("agent runs = %d, want 1", len(*agentRuns))
	}

	rec = jsonConnRequest(t, http.MethodPut, cspGroup.ID, "vm",
		model.CreateConnectionInfoReq{ResourceType: "nlb", ResourceID: nlbARN}, UpdateConnectionInfo)
	if rec.Code != http.StatusOK {
		t.Fatalf("to nlb: %d %s, want 200", rec.Code, rec.Body.String())
	}
	got = getConnection(t, "vm")
	if got.ResourceType != common.ResourceTypeNLB || got.ConnectionStatus != model.ConnectionInfoStatusSuccess ||
		got.IPAddress != "" || got.SSHPort != "" || got.User != "" || got.PrivateKey != "" {
		t.Errorf("row = %+v, want an nlb with success and no SSH fields", got)
	}
	if len(*agentRuns) != 1 {
		t.Errorf("agent runs = %d, want no new run for the nlb", len(*agentRuns))
	}
}
