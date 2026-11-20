package controller

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/cloud-barista/cm-honeybee/server/lib/spider"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"github.com/labstack/echo/v4"
)

// fakeMainValue, fakeS3Access and fakeS3Secret are placeholders; they are not
// credentials.
var (
	fakeMainValue = "fake-" + "main-0003"
	fakeS3Access  = "fake-" + "s3-access-0004"
	fakeS3Secret  = "fake-" + "s3-secret-0005"
)

// gcpMainKV is a gcp credential with the required keys only.
func gcpMainKV() []model.KeyValue {
	return []model.KeyValue{
		{Key: "private_key", Value: fakeMainValue},
		{Key: "project_id", Value: fakeMainValue},
		{Key: "client_email", Value: fakeMainValue},
	}
}

func azureMainKV() []model.KeyValue {
	return []model.KeyValue{
		{Key: "clientId", Value: fakeMainValue},
		{Key: "clientSecret", Value: fakeMainValue},
		{Key: "tenantId", Value: fakeMainValue},
		{Key: "subscriptionId", Value: fakeMainValue},
	}
}

func awsMainKV() []model.KeyValue {
	return []model.KeyValue{
		{Key: "aws_access_key_id", Value: fakeMainValue},
		{Key: "aws_secret_access_key", Value: fakeMainValue},
	}
}

func withKV(base []model.KeyValue, extra ...model.KeyValue) []model.KeyValue {
	return append(base, extra...)
}

func kvMap(in []model.KeyValue) map[string]string {
	out := make(map[string]string, len(in))
	for _, kv := range in {
		out[kv.Key] = kv.Value
	}
	return out
}

// A csp credential takes the S3 keys besides the required ones where buckets
// are read with them, under any case and, on azure, as the storage account
// name and key, stored as S3AccessKey and S3SecretKey. Both or none.
func TestCanonicalizeCredentialKV_OptionalS3Keys(t *testing.T) {
	useSpiderRoot(t)

	mainOnly := func(base []model.KeyValue) map[string]string { return kvMap(base) }
	withS3 := func(base []model.KeyValue) map[string]string {
		m := kvMap(base)
		m["S3AccessKey"] = fakeS3Access
		m["S3SecretKey"] = fakeS3Secret
		return m
	}

	tests := []struct {
		name     string
		provider string
		in       []model.KeyValue
		want     map[string]string
		wantErr  string
	}{
		{"gcp without s3 keys", "GCP", gcpMainKV(), mainOnly(gcpMainKV()), ""},
		{"gcp s3 keys in any case", "GCP", withKV(gcpMainKV(),
			model.KeyValue{Key: "s3accesskey", Value: fakeS3Access}, model.KeyValue{Key: " S3SECRETKEY ", Value: fakeS3Secret}),
			withS3(gcpMainKV()), ""},
		{"azure storage account aliases", "AZURE", withKV(azureMainKV(),
			model.KeyValue{Key: "StorageAccountName", Value: fakeS3Access}, model.KeyValue{Key: "storageaccountkey", Value: fakeS3Secret}),
			withS3(azureMainKV()), ""},
		{"azure s3 keys", "AZURE", withKV(azureMainKV(),
			model.KeyValue{Key: "S3AccessKey", Value: fakeS3Access}, model.KeyValue{Key: "S3SecretKey", Value: fakeS3Secret}),
			withS3(azureMainKV()), ""},
		{"blank s3 keys are dropped", "GCP", withKV(gcpMainKV(),
			model.KeyValue{Key: "S3AccessKey", Value: ""}, model.KeyValue{Key: "S3SecretKey", Value: " "}),
			mainOnly(gcpMainKV()), ""},
		{"access key only", "GCP", withKV(gcpMainKV(),
			model.KeyValue{Key: "S3AccessKey", Value: fakeS3Access}),
			nil, "S3AccessKey and S3SecretKey must be given together"},
		{"secret key with a blank access key", "NHN", []model.KeyValue{
			{Key: "IdentityEndpoint", Value: fakeMainValue}, {Key: "Username", Value: fakeMainValue},
			{Key: "Password", Value: fakeMainValue}, {Key: "DomainName", Value: fakeMainValue},
			{Key: "TenantId", Value: fakeMainValue},
			{Key: "S3AccessKey", Value: ""}, {Key: "S3SecretKey", Value: fakeS3Secret}},
			nil, "S3AccessKey and S3SecretKey must be given together"},
		{"azure alias and key twice", "AZURE", withKV(azureMainKV(),
			model.KeyValue{Key: "S3AccessKey", Value: fakeS3Access}, model.KeyValue{Key: "StorageAccountName", Value: fakeS3Access},
			model.KeyValue{Key: "S3SecretKey", Value: fakeS3Secret}),
			nil, "duplicate credential key: S3AccessKey"},
		{"azure alias on gcp", "GCP", withKV(gcpMainKV(),
			model.KeyValue{Key: "StorageAccountName", Value: fakeS3Access}, model.KeyValue{Key: "S3SecretKey", Value: fakeS3Secret}),
			nil, "credential key not accepted by GCP CSP: StorageAccountName"},
		{"s3 keys on aws", "AWS", withKV(awsMainKV(),
			model.KeyValue{Key: "S3AccessKey", Value: fakeS3Access}, model.KeyValue{Key: "S3SecretKey", Value: fakeS3Secret}),
			nil, "credential key not accepted by AWS CSP: S3AccessKey"},
		{"s3 keys do not stand in for a required key", "GCP", []model.KeyValue{
			{Key: "private_key", Value: fakeMainValue}, {Key: "project_id", Value: fakeMainValue},
			{Key: "S3AccessKey", Value: fakeS3Access}, {Key: "S3SecretKey", Value: fakeS3Secret}},
			nil, "missing required credential keys: client_email"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta, err := spider.GetCloudOSMetaInfo(tt.provider)
			if err != nil {
				t.Fatal(err)
			}
			got, err := canonicalizeCredentialKV(tt.provider, meta, tt.in)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("err = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if gotMap := kvMap(got); len(got) != len(gotMap) || !reflect.DeepEqual(gotMap, tt.want) {
				t.Errorf("credential = %v, want %v", got, tt.want)
			}
		})
	}
}

// Registering and updating a csp group check the credential the same way:
// the S3 keys go through, one of them alone is refused before anything is
// stored.
func TestSourceGroupCredential_OptionalS3Keys(t *testing.T) {
	useSpiderRoot(t)

	sg := &model.SourceGroup{ProviderName: "gcp", RegionName: "asia-northeast3"}
	in := withKV(gcpMainKV(),
		model.KeyValue{Key: "s3accesskey", Value: fakeS3Access}, model.KeyValue{Key: "s3secretkey", Value: fakeS3Secret})
	if err := validateAndCanonicalizeCSP(sg, in); err != nil {
		t.Fatal(err)
	}
	if got := kvMap(sg.Credential); got["S3AccessKey"] != fakeS3Access || got["S3SecretKey"] != fakeS3Secret {
		t.Errorf("credential = %v, want the S3 keys under their names", sg.Credential)
	}

	oneSided, err := json.Marshal(model.CreateSourceGroupReq{Name: "g", Type: common.SourceGroupTypeCSP,
		ProviderName: "gcp", RegionName: "asia-northeast3",
		Credential: withKV(gcpMainKV(), model.KeyValue{Key: "S3SecretKey", Value: fakeS3Secret})})
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(oneSided))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	if err := CreateSourceGroup(echo.New().NewContext(req, rec)); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "S3AccessKey and S3SecretKey must be given together") {
		t.Errorf("create: %d %s, want 400 for the lone key", rec.Code, rec.Body.String())
	}

	openTestDB(t)
	gcpGroup := &model.SourceGroup{ID: "sg-gcp", Name: "sg-gcp", Type: common.SourceGroupTypeCSP,
		ProviderName: "gcp", RegionName: "asia-northeast3"}
	createGroup(t, gcpGroup, nil)
	rec = jsonConnRequest(t, http.MethodPut, gcpGroup.ID, "", model.UpdateSourceGroupReq{
		Credential: withKV(gcpMainKV(), model.KeyValue{Key: "S3AccessKey", Value: fakeS3Access}),
	}, UpdateSourceGroup)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "S3AccessKey and S3SecretKey must be given together") {
		t.Errorf("update: %d %s, want 400 for the lone key", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), fakeS3Access) {
		t.Errorf("the response carries the credential: %s", rec.Body.String())
	}
}

// POST /csp/{name}/region takes the S3 keys as the group registration does
// and hands them on under their names.
func TestListCSPRegions_OptionalS3Keys(t *testing.T) {
	useSpiderRoot(t)
	got := stubRegionZone(t, []spider.RegionZoneInfo{{Name: "koreacentral"}}, nil)

	body, err := json.Marshal(model.ListCSPRegionReq{Credential: withKV(azureMainKV(),
		model.KeyValue{Key: "StorageAccountName", Value: fakeS3Access}, model.KeyValue{Key: "StorageAccountKey", Value: fakeS3Secret})})
	if err != nil {
		t.Fatal(err)
	}
	rec := connRequestWithName(t, http.MethodPost, "azure", body, ListCSPRegions)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	creds := map[string]string{}
	for _, kv := range *got {
		creds[kv.Key] = kv.Value
	}
	if creds["S3AccessKey"] != fakeS3Access || creds["S3SecretKey"] != fakeS3Secret || len(creds) != 6 {
		t.Errorf("credential handed to the query = %v", creds)
	}

	called := stubRegionZone(t, nil, nil)
	body, err = json.Marshal(model.ListCSPRegionReq{Credential: withKV(azureMainKV(),
		model.KeyValue{Key: "StorageAccountKey", Value: fakeS3Secret})})
	if err != nil {
		t.Fatal(err)
	}
	rec = connRequestWithName(t, http.MethodPost, "azure", body, ListCSPRegions)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "S3AccessKey and S3SecretKey must be given together") {
		t.Errorf("lone key: %d %s, want 400", rec.Code, rec.Body.String())
	}
	if *called != nil {
		t.Errorf("the CSP was queried")
	}
}

func getCSPInfo(t *testing.T, name string) (model.CSPInfo, string) {
	t.Helper()

	rec := connRequestWithName(t, http.MethodGet, name, nil, GetCSP)
	if rec.Code != http.StatusOK {
		t.Fatalf("%s: status = %d, body = %s", name, rec.Code, rec.Body.String())
	}
	var res model.CSPInfo
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	return res, rec.Body.String()
}

// GET /csp/{name} keeps credential_keys to the required keys and lists the
// S3 keys apart, as optional fields for object_storage.
func TestGetCSP_OptionalCredentialKeys(t *testing.T) {
	useSpiderRoot(t)
	stubDriverCapability(t, testDriverCaps)

	azure, _ := getCSPInfo(t, "azure")
	if want := []string{"clientId", "clientSecret", "tenantId", "subscriptionId"}; !reflect.DeepEqual(azure.CredentialKeys, want) {
		t.Errorf("azure credential_keys = %v, want %v", azure.CredentialKeys, want)
	}
	if want := []string{"S3AccessKey", "S3SecretKey"}; !reflect.DeepEqual(azure.OptionalCredentialKeys, want) {
		t.Errorf("azure optional_credential_keys = %v, want %v", azure.OptionalCredentialKeys, want)
	}
	if len(azure.Credentials) != 6 {
		t.Fatalf("azure credentials = %+v, want 4 required and 2 optional", azure.Credentials)
	}
	for i, f := range azure.Credentials {
		if i < 4 {
			if f.Optional || f.ResourceTypes != nil || f.Key != azure.CredentialKeys[i] {
				t.Errorf("azure credentials[%d] = %+v, want required %s", i, f, azure.CredentialKeys[i])
			}
			continue
		}
		if !f.Optional || !reflect.DeepEqual(f.ResourceTypes, []string{"object_storage"}) || f.Key != azure.OptionalCredentialKeys[i-4] {
			t.Errorf("azure credentials[%d] = %+v, want optional %s for object_storage", i, f, azure.OptionalCredentialKeys[i-4])
		}
	}
	if !strings.Contains(azure.Credentials[4].Description, "storage account name") ||
		!strings.Contains(azure.Credentials[5].Description, "storage account key") {
		t.Errorf("azure optional descriptions = %q, %q", azure.Credentials[4].Description, azure.Credentials[5].Description)
	}

	aws, raw := getCSPInfo(t, "aws")
	if !strings.Contains(raw, `"optional_credential_keys": []`) {
		t.Errorf("aws optional_credential_keys is not an empty list: %s", raw)
	}
	for _, f := range aws.Credentials {
		if f.Optional {
			t.Errorf("aws credential %+v is optional", f)
		}
	}
	if len(aws.Credentials) != len(aws.CredentialKeys) {
		t.Errorf("aws credentials = %+v, want the required keys only", aws.Credentials)
	}
}

// Every CSP with optional keys has an example and a description for each, and
// none without has examples.
func TestOptionalCredentialExamples_MatchOptionalKeys(t *testing.T) {
	useSpiderRoot(t)

	list, err := spider.ListCloudOS()
	if err != nil {
		t.Fatal(err)
	}
	var withKeys []string
	for _, p := range list {
		keys := optionalCredentialKeys(p)
		if len(keys) == 0 {
			continue
		}
		withKeys = append(withKeys, p)
		for _, k := range keys {
			if ex, ok := optionalCredentialExamples[p][k]; !ok || ex.example == "" || ex.description == "" {
				t.Errorf("%s: no example for optional key %s", p, k)
			}
		}
	}
	var examples []string
	for p := range optionalCredentialExamples {
		examples = append(examples, p)
	}
	sort.Strings(withKeys)
	sort.Strings(examples)
	if !reflect.DeepEqual(withKeys, examples) {
		t.Errorf("CSPs with optional keys = %v, with optional examples = %v", withKeys, examples)
	}
}

// gcpNoS3Conn is a gcp connection whose credential has no S3 keys: its bucket
// calls fail in lib/spider before any CSP is contacted.
var gcpNoS3Conn = spider.Conn{Provider: "GCP", Region: "asia-northeast3", Credential: []spider.KeyValue{
	{Key: "private_key", Value: fakeMainValue}, {Key: "project_id", Value: fakeMainValue}, {Key: "client_email", Value: fakeMainValue},
}}

const gcpNoS3Msg = "object_storage on gcp needs S3AccessKey and S3SecretKey in the source group credential"

// A bucket of a group without the S3 keys is refused with what the group
// lacks, as a resource that cannot be the connection's.
func TestCheckBucket_NoS3Keys(t *testing.T) {
	useSpiderRoot(t)

	_, err := checkBucket(gcpNoS3Conn, "b1")
	if err == nil || err.Error() != gcpNoS3Msg {
		t.Fatalf("err = %v, want %q", err, gcpNoS3Msg)
	}
	if !cspResourceRejected(err) {
		t.Errorf("%v not rejected", err)
	}
}

// Registering an object_storage connection on a group without the S3 keys,
// or pointing one at another bucket, is refused with 400 and nothing is
// written.
func TestCSPConnection_ObjectStorageWithoutS3Keys(t *testing.T) {
	openTestDB(t)
	gcpGroup := &model.SourceGroup{ID: "sg-gcp", Name: "sg-gcp", Type: common.SourceGroupTypeCSP,
		ProviderName: "gcp", RegionName: "asia-northeast3"}
	createGroup(t, gcpGroup, []model.ConnectionInfo{
		{ID: "b", Name: "b", SourceGroupID: gcpGroup.ID, ResourceType: common.ResourceTypeObjectStorage, ResourceID: "b1"},
	})
	useSpiderRoot(t)
	stubDriverCapability(t, map[string]spider.DriverCapability{"GCP": {VMHandler: true}})
	_, _, checks := stubConnectionSide(t, func(ci *model.ConnectionInfo) error { return checkCSPResource(gcpNoS3Conn, ci) })

	rec := jsonConnRequest(t, http.MethodPost, gcpGroup.ID, "",
		model.CreateConnectionInfoReq{Name: "b2", ResourceType: "object_storage", ResourceID: "b2"}, CreateConnectionInfo)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), gcpNoS3Msg) {
		t.Errorf("create: %d %s, want 400 with %q", rec.Code, rec.Body.String(), gcpNoS3Msg)
	}
	if conns := groupConnections(t, gcpGroup.ID); len(conns) != 1 {
		t.Errorf("connections = %d, want the stored one only", len(conns))
	}

	rec = jsonConnRequest(t, http.MethodPut, gcpGroup.ID, "b",
		model.CreateConnectionInfoReq{ResourceID: "b3"}, UpdateConnectionInfo)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), gcpNoS3Msg) {
		t.Errorf("update: %d %s, want 400 with %q", rec.Code, rec.Body.String(), gcpNoS3Msg)
	}
	if got := getConnection(t, "b"); got.ResourceID != "b1" {
		t.Errorf("b row = %+v, want unchanged", got)
	}
	if *checks != 2 {
		t.Errorf("CSP checks = %d, want 2", *checks)
	}
}

// Discovering buckets of a group without the S3 keys reports them
// unsupported with what the group lacks, not a failure.
func TestDiscoverBuckets_NoS3Keys(t *testing.T) {
	useSpiderRoot(t)

	_, err := discoverByType(gcpNoS3Conn, common.ResourceTypeObjectStorage)
	var unsupported errDriverUnsupported
	if !errors.As(err, &unsupported) || unsupported.Error() != gcpNoS3Msg {
		t.Fatalf("err = %v, want errDriverUnsupported %q", err, gcpNoS3Msg)
	}

	stubBucketCSP(t, map[string]string{"near": "asia-northeast3"}, nil, false)
	items, err := discoverByType(gcpNoS3Conn, common.ResourceTypeObjectStorage)
	if err != nil || len(items) != 1 {
		t.Errorf("listed: items = %+v, err = %v, want the bucket", items, err)
	}
}
