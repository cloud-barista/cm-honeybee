package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"github.com/labstack/echo/v4"
)

// stubSealedSecrets makes every agent call fail before SSH, as with OpenBao
// sealed, so the agent imports run and fail without a host. It counts the
// calls.
func stubSealedSecrets(t *testing.T) *int {
	t.Helper()

	calls := 0
	orig := hydrateSecrets
	t.Cleanup(func() { hydrateSecrets = orig })
	hydrateSecrets = func(*model.ConnectionInfo) error {
		calls++
		return errors.New("openbao: sealed")
	}
	return &calls
}

// recordRefreshCSP stands in for the CSP collection and lists the connections
// it was asked for.
func recordRefreshCSP(t *testing.T) *[]string {
	t.Helper()

	called := []string{}
	orig := refreshCSP
	t.Cleanup(func() { refreshCSP = orig })
	refreshCSP = func(_ *model.SourceGroup, ci *model.ConnectionInfo, _ importInfraOptions) error {
		called = append(called, ci.ID)
		return nil
	}
	return &called
}

func importResults(t *testing.T, body []byte) []model.ImportConnectionResult {
	t.Helper()

	var got []model.ImportConnectionResult
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("%v: %s", err, body)
	}
	return got
}

// kindStatuses gives each connection's results as kind:status, in order.
func kindStatuses(results []model.ImportConnectionResult) map[string][]string {
	out := map[string][]string{}
	for _, r := range results {
		out[r.ConnectionID] = []string{}
		for _, k := range r.Results {
			out[r.ConnectionID] = append(out[r.ConnectionID], k.Kind+":"+k.Status)
		}
	}
	return out
}

// import/all runs on each connection of a mixed csp group only what fits it:
// the CSP collection of a resource under its kind, and the agent imports on
// the VM with an address. A failing collection, the agent ones of an
// unreachable VM or the CSP one of a missing bucket, does not stop the others.
func TestImportAllSourceGroup_MixedCSPGroup(t *testing.T) {
	openTestDB(t)
	conns := append(mixedCSPConns(), testConn("bucket-gone", cspGroup, common.ResourceTypeObjectStorage, ""))
	for i := range conns {
		conns[i].ResourceID = "res-" + conns[i].ID
	}
	createGroup(t, cspGroup, conns)
	stubRefreshCSP(t, map[string]bool{"bucket-gone": true})
	stubSealedSecrets(t)

	rec := groupRequest(t, cspGroup.ID, ImportAllSourceGroup)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	got := importResults(t, rec.Body.Bytes())

	ids := []string{}
	for _, r := range got {
		ids = append(ids, r.ConnectionID)
	}
	wantIDs := connIDs(conns)
	sort.Strings(wantIDs)
	if !reflect.DeepEqual(ids, wantIDs) {
		t.Errorf("connections = %v, want every connection in list order %v", ids, wantIDs)
	}

	want := map[string][]string{
		"vm-ip":       {"infra:failed", "software:failed", "kubernetes:failed", "helm:failed"},
		"vm-noip":     {"infra:success"},
		"bucket":      {"bucket:success"},
		"rdbms":       {"rdbms:success"},
		"k8s":         {"kubernetes:success"},
		"nlb":         {"nlb:success"},
		"bucket-gone": {"bucket:failed"},
	}
	if s := kindStatuses(got); !reflect.DeepEqual(s, want) {
		t.Errorf("results = %v, want %v", s, want)
	}

	for _, r := range got {
		if r.ConnectionID == "bucket" && r.ResourceType != common.ResourceTypeObjectStorage {
			t.Errorf("bucket resource_type = %q", r.ResourceType)
		}
		for _, k := range r.Results {
			switch {
			case k.Status == model.ConnectionInfoStatusSuccess && k.ErrorMessage != "":
				t.Errorf("%s %s: error_message = %q on success", r.ConnectionID, k.Kind, k.ErrorMessage)
			case r.ConnectionID == "bucket-gone" && !strings.Contains(k.ErrorMessage, "res-bucket-gone not found"):
				t.Errorf("bucket-gone: error_message = %q, want the CSP error", k.ErrorMessage)
			case r.ConnectionID == "vm-ip" && !strings.Contains(k.ErrorMessage, "openbao: sealed"):
				t.Errorf("vm-ip %s: error_message = %q, want the secret error", k.Kind, k.ErrorMessage)
			}
		}
	}

	if strings.Count(rec.Body.String(), `"error_message"`) != 5 {
		t.Errorf("error_message must appear on the 5 failed results only: %s", rec.Body.String())
	}
}

// An onprem group collects each host as infra and through the agent, and
// leaves out a k8s connection, which has nothing to collect. When no
// collection of any connection succeeds the request fails, as the other group
// imports do.
func TestImportAllSourceGroup_Onprem(t *testing.T) {
	openTestDB(t)
	conns := []model.ConnectionInfo{
		testConn("onprem-vm", onpremGroup, "", "192.0.2.10"),
		testConn("onprem-k8s", onpremGroup, common.ResourceTypeK8s, ""),
	}
	createGroup(t, onpremGroup, conns)
	calls := stubSealedSecrets(t)

	plans := importAllPlans(onpremGroup, conns, importAllOptions{})
	if len(plans) != 1 || plans[0].conn.ID != "onprem-vm" {
		t.Fatalf("plans = %+v, want onprem-vm only", plans)
	}
	var kinds []string
	for _, s := range plans[0].steps {
		kinds = append(kinds, s.kind)
	}
	if w := []string{"infra", "software", "kubernetes", "helm"}; !reflect.DeepEqual(kinds, w) {
		t.Errorf("kinds = %v, want %v", kinds, w)
	}

	rec := groupRequest(t, onpremGroup.ID, ImportAllSourceGroup)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "infra import failed: openbao: sealed") {
		t.Errorf("body = %s, want the first failure", rec.Body.String())
	}
	if *calls != 4 {
		t.Errorf("agent calls = %d, want 4: every collection runs past a failure", *calls)
	}
}

func TestImportAllSourceGroup_NothingToCollect(t *testing.T) {
	openTestDB(t)
	createGroup(t, onpremGroup, []model.ConnectionInfo{testConn("onprem-k8s", onpremGroup, common.ResourceTypeK8s, "")})

	rec := groupRequest(t, onpremGroup.ID, ImportAllSourceGroup)
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Errorf("status = %d, body = %s, want 200 []", rec.Code, rec.Body.String())
	}
}

// fs, db and minio groups are data sources import/all does not collect; the
// request is refused with the import to use instead, and nothing runs.
func TestImportAllSourceGroup_DataGroupsRefused(t *testing.T) {
	tests := []struct {
		sg   *model.SourceGroup
		want string
	}{
		{fsGroup, "use import/fs"},
		{dbGroup, "use import/db"},
		{minioGroup, "use import/objectstorage"},
	}

	for _, tt := range tests {
		t.Run(tt.sg.Type, func(t *testing.T) {
			openTestDB(t)
			createGroup(t, tt.sg, []model.ConnectionInfo{testConn(tt.sg.Type+"-conn", tt.sg, "", "192.0.2.10")})
			calls := stubSealedSecrets(t)
			refreshed := recordRefreshCSP(t)

			rec := groupRequest(t, tt.sg.ID, ImportAllSourceGroup)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400, body = %s", rec.Code, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), tt.want) {
				t.Errorf("body = %s, want %q", rec.Body.String(), tt.want)
			}
			if *calls != 0 || len(*refreshed) != 0 {
				t.Errorf("a refused group ran %d agent and %d CSP collections", *calls, len(*refreshed))
			}
		})
	}
}

// Each import by kind collects from the CSP only the connections of its
// resource type, and refuses a group that is not csp.
func TestImportCSPKindSourceGroup_OnlyThatType(t *testing.T) {
	tests := []struct {
		handler      echo.HandlerFunc
		resourceType string
		kind         string
	}{
		{ImportBucketSourceGroup, common.ResourceTypeObjectStorage, model.ImportKindBucket},
		{ImportRDBMSSourceGroup, common.ResourceTypeRDBMS, model.ImportKindRDBMS},
		{ImportK8sSourceGroup, common.ResourceTypeK8s, model.ImportKindKubernetes},
		{ImportNLBSourceGroup, common.ResourceTypeNLB, model.ImportKindNLB},
	}

	for _, tt := range tests {
		t.Run(tt.resourceType, func(t *testing.T) {
			openTestDB(t)
			conns := append(mixedCSPConns(), testConn("second", cspGroup, tt.resourceType, ""))
			createGroup(t, cspGroup, conns)
			calls := stubSealedSecrets(t)
			refreshed := recordRefreshCSP(t)

			rec := groupRequest(t, cspGroup.ID, tt.handler)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
			}
			wantIDs := []string{}
			for _, c := range conns {
				if c.ResourceType == tt.resourceType {
					wantIDs = append(wantIDs, c.ID)
				}
			}
			if !reflect.DeepEqual(*refreshed, wantIDs) {
				t.Errorf("collected = %v, want %v", *refreshed, wantIDs)
			}
			if *calls != 0 {
				t.Errorf("agent calls = %d, want none", *calls)
			}
			want := map[string][]string{}
			for _, id := range wantIDs {
				want[id] = []string{tt.kind + ":success"}
			}
			if got := kindStatuses(importResults(t, rec.Body.Bytes())); !reflect.DeepEqual(got, want) {
				t.Errorf("results = %v, want %v", got, want)
			}

			createGroup(t, onpremGroup, []model.ConnectionInfo{testConn("onprem-"+tt.resourceType, onpremGroup, tt.resourceType, "")})
			*refreshed = (*refreshed)[:0]
			rec = groupRequest(t, onpremGroup.ID, tt.handler)
			if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "csp source group") {
				t.Errorf("onprem: status = %d, body = %s, want 400", rec.Code, rec.Body.String())
			}
			if len(*refreshed) != 0 {
				t.Errorf("onprem: collected %v, want nothing", *refreshed)
			}
		})
	}
}
