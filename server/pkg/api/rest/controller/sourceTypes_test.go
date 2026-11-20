package controller

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/cloud-barista/cm-honeybee/server/dao"
	"github.com/cloud-barista/cm-honeybee/server/db"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// fakeDBPassword and fakeOSSecret are placeholders; they are not credentials.
var (
	fakeDBPassword = "fake-" + "db-pw"
	fakeOSSecret   = "fake-" + "os-secret"
)

const testKubeconfig = "apiVersion: v1\nkind: Config\nclusters:\n- name: c1\n  cluster:\n    server: https://192.0.2.30:6443\n"

// testPrivateKey is a throwaway key in a form checkSSHKeyOnly parses.
func testPrivateKey(t *testing.T) string {
	t.Helper()

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
}

// sampleFieldValues are values each request field accepts.
func sampleFieldValues(t *testing.T) map[string]any {
	return map[string]any{
		"ip_address":           "192.0.2.10",
		"ssh_port":             "22",
		"user":                 "tester",
		"password":             fakeSSHPassword,
		"private_key":          testPrivateKey(t),
		"kubeconfig":           testKubeconfig,
		"resource_id":          "res-1",
		"zone":                 "zone-a",
		"fs_scan_path":         "/data",
		"db_type":              "postgresql",
		"db_host":              "192.0.2.20",
		"db_port":              "5432",
		"db_username":          "tester",
		"db_password":          fakeDBPassword,
		"db_access_type":       model.AccessTypeDirect,
		"db_tls_mode":          "require",
		"db_tls_ca_pem":        "",
		"db_auth_source":       "admin",
		"db_connect_timeout":   5,
		"db_name":              "app",
		"db_pg_schema":         []string{"public"},
		"os_access_key_id":     "test-access-key",
		"os_secret_access_key": fakeOSSecret,
		"os_endpoint":          "198.51.100.5:9000",
		"os_use_ssl":           true,
		"os_access_type":       model.AccessTypeDirect,
		"os_scan_bucket":       "bucket-a",
		"os_scan_prefix":       "logs/",
	}
}

// jsonFieldIndex maps the JSON field names of a struct type to the field.
func jsonFieldIndex(typ reflect.Type) map[string]int {
	out := map[string]int{}
	for i := 0; i < typ.NumField(); i++ {
		name := strings.Split(typ.Field(i).Tag.Get("json"), ",")[0]
		if name != "" && name != "-" {
			out[name] = i
		}
	}
	return out
}

func setReqField(t *testing.T, req *model.CreateConnectionInfoReq, name string, value any) {
	t.Helper()

	i, ok := jsonFieldIndex(reflect.TypeOf(*req))[name]
	if !ok {
		t.Fatalf("CreateConnectionInfoReq has no field %q", name)
	}
	f := reflect.ValueOf(req).Elem().Field(i)
	if value == nil {
		f.Set(reflect.Zero(f.Type()))
		return
	}
	f.Set(reflect.ValueOf(value))
}

// conditionMet reports whether a required_when condition holds for the
// values set. Conditions on a source group field are not judged here.
func conditionMet(cond string, set map[string]any) (judged bool, met bool) {
	if strings.Contains(cond, " in ") {
		return false, false
	}
	if name, want, ok := strings.Cut(cond, "="); ok {
		return true, set[name] == want
	}
	v := set[cond]
	return true, v != nil && v != ""
}

// fillConnReq builds a request of the connection type with every field the
// spec requires, the first field of each one_of group, and the conditional
// fields whose condition the values in extra meet.
func fillConnReq(t *testing.T, ct model.ConnectionTypeSpec, extra map[string]any) (*model.CreateConnectionInfoReq, map[string]any) {
	t.Helper()

	samples := sampleFieldValues(t)
	set := map[string]any{}
	for k, v := range extra {
		set[k] = v
	}
	all := append(append([]model.SourceFieldSpec{}, ct.AuthFields...), ct.Fields...)
	oneOf := map[string]bool{}
	for _, f := range all {
		if _, ok := set[f.Name]; ok {
			continue
		}
		judged, met := conditionMet(f.RequiredWhen, set)
		needed := f.Required || (f.RequiredWhen != "" && judged && met)
		if f.OneOf != "" && !oneOf[f.OneOf] && (f.RequiredWhen == "" || met) {
			oneOf[f.OneOf] = true
			needed = true
		}
		if needed {
			set[f.Name] = samples[f.Name]
		}
	}

	req := &model.CreateConnectionInfoReq{Name: "conn"}
	if ct.ResourceType != nil {
		req.ResourceType = *ct.ResourceType
	}
	for k, v := range set {
		setReqField(t, req, k, v)
	}
	return req, set
}

func groupForType(sgType string) *model.SourceGroup {
	sg := &model.SourceGroup{ID: "sg-" + sgType, Name: "sg-" + sgType, Type: sgType}
	if sgType == common.SourceGroupTypeMinIO {
		sg.ProviderName, sg.RegionName = "aws", "ap-northeast-2"
	}
	if sgType == common.SourceGroupTypeDB {
		sg.ProviderName = "aws"
	}
	return sg
}

// The types API says what registration checks: a connection built from the
// required fields of a connection type registers, and leaving out any one of
// them, every field of a one_of group, or the fields a condition requires,
// is refused. Every field named exists in the request.
func TestSourceGroupTypes_MatchValidation(t *testing.T) {
	specs := sourceGroupTypeSpecs()

	types := []string{}
	for _, gt := range specs {
		types = append(types, gt.Type)
	}
	if want := []string{"onprem", "csp", "fs", "db", "minio"}; !reflect.DeepEqual(types, want) {
		t.Fatalf("types = %v, want %v", types, want)
	}

	groupFields := jsonFieldIndex(reflect.TypeOf(model.CreateSourceGroupReq{}))
	for _, gt := range specs {
		for _, alias := range append([]string{gt.Type}, gt.Aliases...) {
			if !common.IsValidSourceGroupType(alias) {
				t.Errorf("%s: %q is not a valid source group type", gt.Type, alias)
			}
		}
		if _, err := storedSourceGroupTypes(gt.Type); err != nil {
			t.Errorf("%s: the group list filter refuses the type: %v", gt.Type, err)
		}
		for _, f := range gt.GroupFields {
			if _, ok := groupFields[f.Name]; !ok {
				t.Errorf("%s: group field %q is not in CreateSourceGroupReq", gt.Type, f.Name)
			}
		}

		sg := groupForType(gt.Type)
		for _, ct := range gt.ConnectionTypes {
			name := gt.Type + "/" + ct.Type
			if (ct.ResourceType == nil) != (gt.ResourceTypeField == model.ResourceTypeFieldNone) {
				t.Errorf("%s: resource_type %v does not fit resource_type_field %q", name, ct.ResourceType, gt.ResourceTypeField)
			}

			req, set := fillConnReq(t, ct, nil)
			if _, err := checkCreateConnectionInfoReq(sg, req); err != nil {
				t.Errorf("%s: the required fields %v are refused: %v", name, keys(set), err)
				continue
			}

			oneOf := map[string][]string{}
			for _, f := range append(append([]model.SourceFieldSpec{}, ct.AuthFields...), ct.Fields...) {
				if f.Required {
					blank, _ := fillConnReq(t, ct, map[string]any{f.Name: nil})
					if _, err := checkCreateConnectionInfoReq(sg, blank); err == nil {
						t.Errorf("%s: required %s left out, but registration accepts it", name, f.Name)
					}
				}
				if f.OneOf != "" {
					oneOf[f.OneOf] = append(oneOf[f.OneOf], f.Name)
				}
				if f.RequiredWhen != "" && !strings.Contains(f.RequiredWhen, " in ") {
					cond := map[string]any{}
					if n, v, ok := strings.Cut(f.RequiredWhen, "="); ok {
						cond[n] = v
					} else {
						cond[f.RequiredWhen] = sampleFieldValues(t)[f.RequiredWhen]
					}
					full, fullSet := fillConnReq(t, ct, cond)
					if _, err := checkCreateConnectionInfoReq(sg, full); err != nil {
						t.Errorf("%s: with %v the fields %v are refused: %v", name, cond, keys(fullSet), err)
					}
					if f.OneOf == "" {
						cond[f.Name] = nil
						blank, _ := fillConnReq(t, ct, cond)
						if _, err := checkCreateConnectionInfoReq(sg, blank); err == nil {
							t.Errorf("%s: %s left out under %s, but registration accepts it", name, f.Name, f.RequiredWhen)
						}
					}
				}
				for _, v := range f.Enum {
					withEnum, _ := fillConnReq(t, ct, map[string]any{f.Name: v})
					if _, err := checkCreateConnectionInfoReq(sg, withEnum); err != nil {
						t.Errorf("%s: %s=%s from enum is refused: %v", name, f.Name, v, err)
					}
				}
			}
			for group, members := range oneOf {
				cond := map[string]any{}
				for _, m := range members {
					cond[m] = nil
				}
				blank, _ := fillConnReq(t, ct, cond)
				if gt.Type == common.SourceGroupTypeCSP {
					// The SSH fields of a csp vm are required only with ip_address.
					setReqField(t, blank, "ip_address", "192.0.2.10")
					setReqField(t, blank, "user", "tester")
				}
				if _, err := checkCreateConnectionInfoReq(sg, blank); err == nil {
					t.Errorf("%s: no field of one_of %s %v given, but registration accepts it", name, group, members)
				}
			}
		}
	}
}

func keys(m map[string]any) []string {
	out := []string{}
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// The resource types listed are the ones registration accepts: the csp ones
// are exactly what IsValidCSPResourceType takes and its error lists, and an
// on-prem type outside the list is refused.
func TestSourceGroupTypes_ResourceTypes(t *testing.T) {
	want := "resource_type must be one of " + strings.Join(cspResourceTypes, " | ")
	if common.CSPResourceTypesMsg != want {
		t.Errorf("CSPResourceTypesMsg = %q, want %q", common.CSPResourceTypesMsg, want)
	}
	for _, rt := range cspResourceTypes {
		if !common.IsValidCSPResourceType(rt) {
			t.Errorf("%s is listed but not accepted", rt)
		}
	}
	for _, rt := range []string{"", "bucket", "vm2", "VM", "fs"} {
		if common.IsValidCSPResourceType(rt) {
			t.Errorf("%q is accepted but not listed", rt)
		}
	}

	var onprem, csp model.SourceGroupTypeSpec
	for _, gt := range sourceGroupTypeSpecs() {
		switch gt.Type {
		case common.SourceGroupTypeOnprem:
			onprem = gt
		case common.SourceGroupTypeCSP:
			csp = gt
		}
	}
	listed := map[string]bool{}
	for _, ct := range onprem.ConnectionTypes {
		listed[*ct.ResourceType] = true
	}
	for _, rt := range cspResourceTypes {
		_, err := checkCreateConnectionInfoReq(onpremGroup, &model.CreateConnectionInfoReq{
			Name: "c", ResourceType: rt, IPAddress: "192.0.2.10", SSHPort: "22", User: "u",
			PrivateKey: "k", Kubeconfig: testKubeconfig,
		})
		if listed[rt] != (err == nil) {
			t.Errorf("onprem %s: listed %v, registration error %v", rt, listed[rt], err)
		}
	}
	if onprem.DefaultResourceType != common.ResourceTypeVM {
		t.Errorf("onprem default resource_type = %q", onprem.DefaultResourceType)
	}

	got := []string{}
	for _, ct := range csp.ConnectionTypes {
		got = append(got, *ct.ResourceType)
		if *ct.ResourceType != common.ResourceTypeVM && len(ct.AuthFields) != 0 {
			t.Errorf("csp %s lists auth fields %v, but registration refuses SSH fields on it", ct.Type, ct.AuthFields)
		}
	}
	if !reflect.DeepEqual(got, cspResourceTypes) {
		t.Errorf("csp connection types = %v, want %v", got, cspResourceTypes)
	}
	for _, f := range csp.GroupFields {
		if f.Name == "credential" && f.CredentialSource != "/csp/{name}" {
			t.Errorf("csp credential_source = %q", f.CredentialSource)
		}
	}
}

// The handler serves the specs, with null for the resource_type of a
// connection that takes none.
func TestListSourceGroupTypes(t *testing.T) {
	rec := queryRequest(t, "", "", ListSourceGroupTypes)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var raw struct {
		Types []struct {
			Type            string                   `json:"type"`
			ConnectionTypes []map[string]interface{} `json:"connection_types"`
		} `json:"types"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	for _, gt := range raw.Types {
		for _, ct := range gt.ConnectionTypes {
			rt, present := ct["resource_type"]
			switch gt.Type {
			case "fs", "db", "minio":
				if !present || rt != nil {
					t.Errorf("%s: resource_type = %v (present %v), want null", gt.Type, rt, present)
				}
			default:
				if rt == nil {
					t.Errorf("%s/%v: resource_type is null", gt.Type, ct["type"])
				}
			}
		}
	}
}

// capCase is a group with its connections and what its response must say.
type capCase struct {
	sg     *model.SourceGroup
	conns  []model.ConnectionInfo
	caps   model.SourceGroupCapabilities
	counts map[string]int
}

func capCases() []capCase {
	g := func(id, sgType string) *model.SourceGroup {
		return &model.SourceGroup{ID: id, Name: id, Type: sgType}
	}
	cspMixed := g("cap-csp-mixed", common.SourceGroupTypeCSP)
	cspNoIP := g("cap-csp-noip", common.SourceGroupTypeCSP)
	cspBucket := g("cap-csp-bucket", common.SourceGroupTypeCSP)
	onprem := g("cap-onprem", common.SourceGroupTypeSSH)
	onpremK8s := g("cap-onprem-k8s", common.SourceGroupTypeOnprem)
	fs := g("cap-fs", common.SourceGroupTypeFS)
	dbg := g("cap-db", common.SourceGroupTypeDB)
	minio := g("cap-minio", common.SourceGroupTypeMinIO)
	empty := g("cap-empty", common.SourceGroupTypeCSP)

	mixed := []model.ConnectionInfo{}
	for _, c := range mixedCSPConns() {
		c.ID, c.Name, c.SourceGroupID = "mixed-"+c.ID, "mixed-"+c.Name, cspMixed.ID
		mixed = append(mixed, c)
	}

	return []capCase{
		{cspMixed, mixed, model.SourceGroupCapabilities{IsCollectInfra: true, IsCollectSoftware: true,
			IsCollectKubernetes: true, IsCollectObjectStorage: true, IsCollectDb: true, IsCollectNlb: true},
			map[string]int{"vm": 2, "object_storage": 1, "rdbms": 1, "k8s": 1, "nlb": 1}},
		{cspNoIP, []model.ConnectionInfo{testConn("noip-vm", cspNoIP, common.ResourceTypeVM, "")},
			model.SourceGroupCapabilities{IsCollectInfra: true}, map[string]int{"vm": 1}},
		{cspBucket, []model.ConnectionInfo{testConn("only-bucket", cspBucket, common.ResourceTypeObjectStorage, "")},
			model.SourceGroupCapabilities{IsCollectInfra: true, IsCollectObjectStorage: true}, map[string]int{"object_storage": 1}},
		{onprem, []model.ConnectionInfo{
			testConn("op-untyped", onprem, "", "192.0.2.10"),
			testConn("op-vm", onprem, common.ResourceTypeVM, "192.0.2.11"),
			testConn("op-k8s", onprem, common.ResourceTypeK8s, ""),
		}, model.SourceGroupCapabilities{IsCollectInfra: true, IsCollectSoftware: true, IsCollectKubernetes: true, IsCollectDb: true},
			map[string]int{"vm": 2, "k8s": 1}},
		{onpremK8s, []model.ConnectionInfo{testConn("opk-k8s", onpremK8s, common.ResourceTypeK8s, "")},
			model.SourceGroupCapabilities{}, map[string]int{"k8s": 1}},
		{fs, []model.ConnectionInfo{testConn("fs-1", fs, "", "192.0.2.20"), testConn("fs-2", fs, "", "192.0.2.21")},
			model.SourceGroupCapabilities{IsCollectFs: true}, map[string]int{"fs": 2}},
		{dbg, []model.ConnectionInfo{testConn("db-1", dbg, "", "")},
			model.SourceGroupCapabilities{IsCollectDb: true}, map[string]int{"db": 1}},
		{minio, []model.ConnectionInfo{testConn("minio-1", minio, "", "")},
			model.SourceGroupCapabilities{IsCollectObjectStorage: true}, map[string]int{"minio": 1}},
		{empty, nil, model.SourceGroupCapabilities{}, map[string]int{}},
	}
}

func createCapCases(t *testing.T) []capCase {
	t.Helper()

	cases := capCases()
	for _, tc := range cases {
		createGroup(t, tc.sg, tc.conns)
	}
	// Statuses to check the status counts the aggregation also gives.
	for _, id := range []string{"mixed-vm-ip", "op-vm"} {
		if err := db.DB.Model(&model.ConnectionInfo{}).Where("id = ?", id).
			Updates(map[string]any{"connection_status": "success", "agent_status": "success"}).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.DB.Model(&model.ConnectionInfo{}).Where("id = ?", "mixed-bucket").
		Update("connection_status", "success").Error; err != nil {
		t.Fatal(err)
	}
	return cases
}

// countQueries counts the statements run against connection_infos.
func countQueries(t *testing.T) *int {
	t.Helper()

	n := 0
	count := func(tx *gorm.DB) {
		if tx.Statement.Table == "connection_infos" {
			n++
		}
	}
	if err := db.DB.Callback().Query().After("gorm:query").Register("test:count_query", count); err != nil {
		t.Fatal(err)
	}
	if err := db.DB.Callback().Row().After("gorm:row").Register("test:count_row", count); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = db.DB.Callback().Query().Remove("test:count_query")
		_ = db.DB.Callback().Row().Remove("test:count_row")
	})
	return &n
}

// The group list reports capabilities and connection counts by type for
// groups of every kind, an empty one included, reading the connections of all
// listed groups in one query. The status counts stay what they were.
func TestListSourceGroup_CapabilitiesAndCounts(t *testing.T) {
	openTestDB(t)
	cases := createCapCases(t)
	queries := countQueries(t)

	rec := queryRequest(t, "", "", ListSourceGroup)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if *queries != 1 {
		t.Errorf("connection_infos queried %d times for %d groups, want 1", *queries, len(cases))
	}
	var res model.ListSourceGroupRes
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	byID := map[string]model.SourceGroupRes{}
	for _, sg := range res.SourceGroup {
		byID[sg.ID] = sg
	}
	total := 0
	for _, tc := range cases {
		got, ok := byID[tc.sg.ID]
		if !ok {
			t.Errorf("%s: not listed", tc.sg.ID)
			continue
		}
		if got.Capabilities != tc.caps {
			t.Errorf("%s: capabilities = %+v, want %+v", tc.sg.ID, got.Capabilities, tc.caps)
		}
		if !reflect.DeepEqual(got.ConnectionCountByType, tc.counts) {
			t.Errorf("%s: connection_count_by_type = %v, want %v", tc.sg.ID, got.ConnectionCountByType, tc.counts)
		}
		if got.ConnectionInfoStatusCount.ConnectionInfoTotal != len(tc.conns) {
			t.Errorf("%s: total = %d, want %d", tc.sg.ID, got.ConnectionInfoStatusCount.ConnectionInfoTotal, len(tc.conns))
		}
		total += len(tc.conns)
	}

	mixed := byID["cap-csp-mixed"].ConnectionInfoStatusCount
	if want := (model.ConnectionInfoStatusCount{CountConnectionSuccess: 2, CountConnectionFailed: 4,
		CountAgentSuccess: 1, CountAgentFailed: 5, ConnectionInfoTotal: 6}); mixed != want {
		t.Errorf("mixed status count = %+v, want %+v", mixed, want)
	}
	if want := (model.ConnectionInfoStatusCount{CountConnectionSuccess: 3, CountConnectionFailed: total - 3,
		CountAgentSuccess: 2, CountAgentFailed: total - 2, ConnectionInfoTotal: total}); res.ConnectionInfoStatusCount != want {
		t.Errorf("list status count = %+v, want %+v", res.ConnectionInfoStatusCount, want)
	}

	// An empty group gives an empty object, not null.
	if !strings.Contains(rec.Body.String(), `"connection_count_by_type": {}`) {
		t.Errorf("an empty group's connection_count_by_type is not {}: %s", rec.Body.String())
	}
}

// A single group reports the same as the list, and counting rows already read
// (as the create response does) gives what the aggregate query gives.
func TestGetSourceGroup_CapabilitiesAndCounts(t *testing.T) {
	openTestDB(t)
	cases := createCapCases(t)

	for _, tc := range cases {
		rec := queryRequest(t, tc.sg.ID, "", GetSourceGroup)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: status = %d, body = %s", tc.sg.ID, rec.Code, rec.Body.String())
		}
		var got model.SourceGroupRes
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if got.Capabilities != tc.caps || !reflect.DeepEqual(got.ConnectionCountByType, tc.counts) {
			t.Errorf("%s: got %+v %v, want %+v %v", tc.sg.ID, got.Capabilities, got.ConnectionCountByType, tc.caps, tc.counts)
		}

		fromDB, err := dao.ConnectionInfoCountByType([]string{tc.sg.ID})
		if err != nil {
			t.Fatal(err)
		}
		// Read by equality: ConnectionInfoGetList matches source_group_id with
		// LIKE, which takes cap-onprem-k8s for cap-onprem as well.
		var conns []model.ConnectionInfo
		if err := db.DB.Where("source_group_id = ?", tc.sg.ID).Find(&conns).Error; err != nil {
			t.Fatal(err)
		}
		inMemory := countConnectionsByType(tc.sg.ID, conns)
		sortCounts := func(c []dao.ConnectionTypeCount) {
			sort.Slice(c, func(i, j int) bool { return c[i].ResourceType < c[j].ResourceType })
		}
		sortCounts(fromDB)
		sortCounts(inMemory)
		if !reflect.DeepEqual(fromDB, inMemory) {
			t.Errorf("%s: query counts %+v, in-memory counts %+v", tc.sg.ID, fromDB, inMemory)
		}
	}
}

// The update response carries the capabilities and counts as well.
func TestUpdateSourceGroup_CapabilitiesAndCounts(t *testing.T) {
	openTestDB(t)
	cases := createCapCases(t)
	tc := cases[0]

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/", bytes.NewReader([]byte(`{"description":"changed"}`)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	c := echo.New().NewContext(req, rec)
	c.SetParamNames("sgId")
	c.SetParamValues(tc.sg.ID)
	if err := UpdateSourceGroup(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var got model.SourceGroup
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Description != "changed" || got.Capabilities == nil || *got.Capabilities != tc.caps ||
		!reflect.DeepEqual(got.ConnectionCountByType, tc.counts) {
		t.Errorf("update response = %+v %v", got.Capabilities, got.ConnectionCountByType)
	}
}

// A group created without connections reports no capability and an empty
// count, not null.
func TestCreateSourceGroup_EmptyCapabilities(t *testing.T) {
	openTestDB(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(`{"name":"new-fs","type":"fs"}`)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	if err := CreateSourceGroup(echo.New().NewContext(req, rec)); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var got model.SourceGroupRes
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Capabilities != (model.SourceGroupCapabilities{}) || got.ConnectionCountByType == nil || len(got.ConnectionCountByType) != 0 {
		t.Errorf("created group = %+v %v", got.Capabilities, got.ConnectionCountByType)
	}
}
