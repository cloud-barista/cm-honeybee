package controller

import (
	"bytes"
	"encoding/json"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/cloud-barista/cm-honeybee/server/lib/spider"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"github.com/labstack/echo/v4"
)

// fakeAWSKeyID and fakeAWSSecret are placeholders; they are not credentials.
var (
	fakeAWSKeyID  = "fake-" + "key-id-0001"
	fakeAWSSecret = "fake-" + "secret-0002"
)

// useSpiderRoot points $CBSPIDER_ROOT at a temp dir holding cb-spider's
// metadata files from the module cache, as lib/spider's own tests do.
func useSpiderRoot(t *testing.T) {
	t.Helper()

	out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "github.com/cloud-barista/cb-spider").Output()
	if err != nil {
		t.Fatalf("go list cb-spider: %v", err)
	}
	src := strings.TrimSpace(string(out))
	root := t.TempDir()
	for _, rel := range []string{
		"cloud-driver-libs/cloudos.yaml",
		"cloud-driver-libs/cloudos_meta.yaml",
		"conf/calllog_conf.yaml",
		"conf/log_conf.yaml",
	} {
		b, err := os.ReadFile(filepath.Join(src, rel))
		if err != nil {
			t.Fatal(err)
		}
		dst := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dst, b, 0644); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("CBSPIDER_ROOT", root)
}

// stubDriverCapability answers capabilities by provider; one not in caps has
// no driver.
func stubDriverCapability(t *testing.T, caps map[string]spider.DriverCapability) {
	t.Helper()

	orig := driverCapability
	t.Cleanup(func() { driverCapability = orig })
	driverCapability = func(provider string) (*spider.DriverCapability, error) {
		c, ok := caps[strings.ToUpper(provider)]
		if !ok {
			return nil, errors.New("unsupported CSP driver: " + provider)
		}
		return &c, nil
	}
}

var testDriverCaps = map[string]spider.DriverCapability{
	"AWS":    {VMHandler: true, ClusterHandler: true, NLBHandler: true, RDBMSHandler: true},
	"ORACLE": {VMHandler: true, ClusterHandler: true},
	"KT":     {VMHandler: true, NLBHandler: true},
}

// GET /csp keeps its name list and adds the resource types each CSP's
// connections can be collected as; a CSP without a driver supports none.
func TestListCSP_SupportedResourceTypes(t *testing.T) {
	stubDriverCapability(t, testDriverCaps)
	orig := listCloudOS
	t.Cleanup(func() { listCloudOS = orig })
	listCloudOS = func() ([]string, error) { return []string{"AWS", "ORACLE", "KT", "MOCK"}, nil }

	rec := queryRequest(t, "", "", ListCSP)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var res model.ListCSPRes
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if want := []string{"aws", "oracle", "kt", "mock"}; !reflect.DeepEqual(res.CSP, want) {
		t.Errorf("csp = %v, want %v", res.CSP, want)
	}
	want := map[string][]string{
		"aws":    {"vm", "k8s", "object_storage", "nlb", "rdbms"},
		"oracle": {"vm", "k8s"},
		"kt":     {"vm", "object_storage", "nlb"},
		"mock":   {},
	}
	if !reflect.DeepEqual(res.SupportedResourceTypes, want) {
		t.Errorf("supported_resource_types = %v, want %v", res.SupportedResourceTypes, want)
	}
}

// GET /csp/{name} carries the same field.
func TestGetCSP_SupportedResourceTypes(t *testing.T) {
	useSpiderRoot(t)
	stubDriverCapability(t, testDriverCaps)

	rec := connRequestWithName(t, http.MethodGet, "oracle", nil, GetCSP)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var res model.CSPInfo
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if want := []string{"vm", "k8s"}; !reflect.DeepEqual(res.SupportedResourceTypes, want) {
		t.Errorf("supported_resource_types = %v, want %v", res.SupportedResourceTypes, want)
	}
	if len(res.CredentialKeys) == 0 {
		t.Errorf("credential_keys is empty")
	}
}

// Discovery and the CSP metadata judge NLB and RDBMS support alike, and a
// real driver's capability reads as its handlers say.
func TestDriverSupports(t *testing.T) {
	c := &spider.DriverCapability{VMHandler: true, RDBMSHandler: true}
	got := map[string]bool{}
	for _, rt := range cspResourceTypes {
		got[rt] = driverSupports(c, rt)
	}
	want := map[string]bool{"vm": true, "k8s": false, "object_storage": false, "nlb": false, "rdbms": true}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("driverSupports = %v, want %v", got, want)
	}
	if driverSupports(nil, "vm") {
		t.Errorf("a nil capability supports vm")
	}

	aws, err := spider.GetDriverCapability("AWS")
	if err != nil {
		t.Fatal(err)
	}
	if driverSupports(aws, "nlb") != aws.NLBHandler || driverSupports(aws, "vm") != aws.VMHandler {
		t.Errorf("aws capability %+v read wrong", aws)
	}
}

// s3RuleProviders reads the providers lib/spider's s3ConnRule has a case for
// out of its source. Calling the rule through ListS3Buckets is not an option:
// for several providers it goes on to contact the CSP.
func s3RuleProviders(t *testing.T) []string {
	t.Helper()

	file, err := parser.ParseFile(token.NewFileSet(), filepath.Join("..", "..", "..", "..", "lib", "spider", "direct_s3.go"), nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "s3ConnRule" {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			cc, ok := n.(*ast.CaseClause)
			if !ok {
				return true
			}
			for _, e := range cc.List {
				if lit, ok := e.(*ast.BasicLit); ok && lit.Kind == token.STRING {
					v, _ := strconv.Unquote(lit.Value)
					out = append(out, v)
				}
			}
			return true
		})
	}
	if len(out) == 0 {
		t.Fatal("no provider case found in s3ConnRule")
	}
	sort.Strings(out)
	return out
}

// objectStorageSupported names the providers lib/spider's S3 rules handle,
// for every CSP cb-spider lists.
func TestObjectStorageSupported_MatchesS3Rules(t *testing.T) {
	useSpiderRoot(t)

	rule := map[string]bool{}
	for _, p := range s3RuleProviders(t) {
		rule[p] = true
	}
	list, err := spider.ListCloudOS()
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range list {
		if objectStorageSupported(p) != rule[strings.ToUpper(p)] {
			t.Errorf("%s: objectStorageSupported = %v, s3ConnRule handles it: %v", p, objectStorageSupported(p), rule[strings.ToUpper(p)])
		}
		delete(rule, strings.ToUpper(p))
	}
	if len(rule) != 0 {
		t.Errorf("s3ConnRule handles providers cb-spider does not list: %v", rule)
	}
}

func connRequestWithName(t *testing.T, method string, name string, body []byte, handler echo.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, "/", bytes.NewReader(body))
	if body != nil {
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	}
	c := echo.New().NewContext(req, rec)
	c.SetParamNames("name")
	c.SetParamValues(name)
	if err := handler(c); err != nil {
		t.Fatal(err)
	}
	return rec
}

// stubRegionZone stands in for the region query and records the credential
// it was given.
func stubRegionZone(t *testing.T, list []spider.RegionZoneInfo, err error) *[]spider.KeyValue {
	t.Helper()

	var got []spider.KeyValue
	orig := listRegionZone
	t.Cleanup(func() { listRegionZone = orig })
	listRegionZone = func(provider string, credential []spider.KeyValue) ([]spider.RegionZoneInfo, error) {
		got = credential
		return list, err
	}
	return &got
}

// POST /csp/{name}/region lists regions with the credential in the body,
// normalized to the keys a csp group stores, and answers as the group region
// API does.
func TestListCSPRegions(t *testing.T) {
	useSpiderRoot(t)
	got := stubRegionZone(t, []spider.RegionZoneInfo{
		{Name: "ap-northeast-2", DisplayName: "Seoul", ZoneList: []spider.ZoneInfo{{Name: "ap-northeast-2a"}}},
	}, nil)

	body := `{"credential":[{"key":"ClientId","value":"` + fakeAWSKeyID + `"},{"key":"aws_secret_access_key","value":"` + fakeAWSSecret + `"}]}`
	rec := connRequestWithName(t, http.MethodPost, "AWS", []byte(body), ListCSPRegions)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var res model.ListRegionRes
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	want := model.ListRegionRes{Provider: "aws", Regions: []model.CSPRegion{
		{Name: "ap-northeast-2", DisplayName: "Seoul", Zones: []string{"ap-northeast-2a"}},
	}}
	if !reflect.DeepEqual(res, want) {
		t.Errorf("response = %+v, want %+v", res, want)
	}
	creds := map[string]string{}
	for _, kv := range *got {
		creds[kv.Key] = kv.Value
	}
	if creds["aws_access_key_id"] != fakeAWSKeyID || creds["aws_secret_access_key"] != fakeAWSSecret || len(creds) != 2 {
		t.Errorf("credential handed to the query = %v", creds)
	}
	if strings.Contains(rec.Body.String(), fakeAWSSecret) || strings.Contains(rec.Body.String(), fakeAWSKeyID) {
		t.Errorf("the response carries the credential: %s", rec.Body.String())
	}
}

// A request the CSP cannot be asked with is refused with 400 before the
// query: a missing or unknown key, no credential, a body that is not JSON, or
// an unknown CSP.
func TestListCSPRegions_BadRequest(t *testing.T) {
	useSpiderRoot(t)

	tests := []struct {
		name, csp, body, wantErr string
	}{
		{"missing key", "aws", `{"credential":[{"key":"aws_access_key_id","value":"` + fakeAWSKeyID + `"}]}`, "missing required credential keys: aws_secret_access_key"},
		{"unknown key", "aws", `{"credential":[{"key":"aws_access_key_id","value":"a"},{"key":"aws_secret_access_key","value":"b"},{"key":"token","value":"c"}]}`, "not accepted"},
		{"no credential", "aws", `{}`, "credential is empty"},
		{"not json", "aws", `{"credential":`, "not valid JSON"},
		{"unknown csp", "nocloud", `{"credential":[{"key":"k","value":"v"}]}`, "nocloud"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := stubRegionZone(t, nil, nil)
			rec := connRequestWithName(t, http.MethodPost, tt.csp, []byte(tt.body), ListCSPRegions)
			if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), tt.wantErr) {
				t.Errorf("status = %d, body = %s, want 400 with %q", rec.Code, rec.Body.String(), tt.wantErr)
			}
			if *called != nil {
				t.Errorf("the CSP was queried")
			}
			if strings.Contains(rec.Body.String(), fakeAWSKeyID) {
				t.Errorf("the response carries the credential: %s", rec.Body.String())
			}
		})
	}
}

// A driver error that quotes the credential is masked before it is logged
// and returned: ReturnInternalError logs the same error text it returns.
func TestListCSPRegions_MasksCredentialInError(t *testing.T) {
	useSpiderRoot(t)
	stubRegionZone(t, nil, errors.New("AuthFailure: key "+fakeAWSKeyID+" with secret "+fakeAWSSecret+" was not accepted"))

	body := `{"credential":[{"key":"aws_access_key_id","value":"` + fakeAWSKeyID + `"},{"key":"aws_secret_access_key","value":"` + fakeAWSSecret + `"}]}`
	rec := connRequestWithName(t, http.MethodPost, "aws", []byte(body), ListCSPRegions)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), fakeAWSKeyID) || strings.Contains(rec.Body.String(), fakeAWSSecret) {
		t.Errorf("the error carries the credential: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "AuthFailure: key **** with secret **** was not accepted") {
		t.Errorf("the error lost its text: %s", rec.Body.String())
	}
}
