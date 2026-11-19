package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	rdbmsmodel "github.com/cloud-barista/cm-beetle/imdl/rdbms-model"
	storagemodel "github.com/cloud-barista/cm-beetle/imdl/storage-model"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/kubernetes"
	"github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/cloud-barista/cm-honeybee/server/dao"
	"github.com/cloud-barista/cm-honeybee/server/db"
	"github.com/cloud-barista/cm-honeybee/server/lib/spider"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"github.com/labstack/echo/v4"
)

// redirectErr is what every item of a bucket in another region answers with.
const redirectErr = "The bucket you are attempting to access must be addressed using the specified endpoint."

// unreadableBucket is a bucket none of whose six items could be read.
func unreadableBucket(name string) *spider.S3BucketDetail {
	d := &spider.S3BucketDetail{Name: name, ItemsTried: 6}
	for _, item := range []string{"versioning", "cors", "encryption", "policy", "tagging", "objects"} {
		d.Errors = append(d.Errors, item+": "+redirectErr)
	}
	return d
}

// partlyReadBucket is a bucket whose versioning was read and the rest failed.
func partlyReadBucket(name string) *spider.S3BucketDetail {
	d := unreadableBucket(name)
	d.ItemsRead = 1
	d.Errors = d.Errors[1:]
	d.VersioningEnabled = true
	return d
}

// stubRefreshBuckets stands in for the CSP: each bucket connection gets the
// detail returned for it, stored the way refreshCSPConnection stores it.
func stubRefreshBuckets(t *testing.T, details map[string]*spider.S3BucketDetail) {
	t.Helper()

	orig := refreshCSP
	t.Cleanup(func() { refreshCSP = orig })
	refreshCSP = func(_ *model.SourceGroup, ci *model.ConnectionInfo, _ importInfraOptions) error {
		return storeBucketDetail(ci.ID, "ap-northeast-2", details[ci.ID])
	}
}

func connRequest(t *testing.T, method string, sgID string, connID string, handler echo.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()

	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(httptest.NewRequest(method, "/", nil), rec)
	c.SetParamNames("sgId", "connId")
	c.SetParamValues(sgID, connID)
	if err := handler(c); err != nil {
		t.Fatal(err)
	}
	return rec
}

func getConnection(t *testing.T, connID string) *model.ConnectionInfo {
	t.Helper()

	ci, err := dao.ConnectionInfoGet(connID)
	if err != nil {
		t.Fatal(err)
	}
	return ci
}

// A bucket that was found but none of whose items could be read keeps its
// errors on record, and is stored, reported and refined as a failure.
func TestStoreBucketDetail_NothingReadIsFailed(t *testing.T) {
	openTestDB(t)
	createGroup(t, cspGroup, []model.ConnectionInfo{testConn("c1", cspGroup, common.ResourceTypeObjectStorage, "")})

	err := storeBucketDetail("c1", "ap-northeast-2", unreadableBucket("bucket-a"))
	if err == nil || !strings.Contains(err.Error(), "none of the details of bucket 'bucket-a'") ||
		!strings.Contains(err.Error(), redirectErr) {
		t.Errorf("err = %v, want the bucket named with the first reason", err)
	}

	saved, err := dao.SavedDataInfoGet("c1")
	if err != nil {
		t.Fatal(err)
	}
	if saved.Status != model.ConnectionInfoStatusFailed {
		t.Errorf("status = %q, want failed", saved.Status)
	}
	got, err := doGetDataInfo("c1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Errors) != 6 {
		t.Errorf("stored errors = %q, want all six", got.Errors)
	}

	if _, err := doGetRefinedBucket("c1"); !errors.Is(err, errBucketCollectFailed) {
		t.Errorf("refined: err = %v, want errBucketCollectFailed", err)
	}
}

// Some items failing still leaves a collected bucket: success, with the
// failures kept in its errors.
func TestStoreBucketDetail_PartlyReadIsSuccess(t *testing.T) {
	openTestDB(t)
	createGroup(t, cspGroup, []model.ConnectionInfo{testConn("c1", cspGroup, common.ResourceTypeObjectStorage, "")})

	if err := storeBucketDetail("c1", "ap-northeast-2", partlyReadBucket("bucket-a")); err != nil {
		t.Fatalf("err = %v, want nil", err)
	}

	saved, err := dao.SavedDataInfoGet("c1")
	if err != nil {
		t.Fatal(err)
	}
	if saved.Status != model.ConnectionInfoStatusSuccess {
		t.Errorf("status = %q, want success", saved.Status)
	}
	got, err := doGetDataInfo("c1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Errors) != 5 {
		t.Errorf("stored errors = %q, want five", got.Errors)
	}
	p, err := doGetRefinedBucket("c1")
	if err != nil {
		t.Fatal(err)
	}
	if p.BucketName != "bucket-a" || !p.VersioningEnabled {
		t.Errorf("refined = %#v", p)
	}
}

// The import of an unreadable bucket answers 400, leaves every record and the
// connection status failed with the reason, and the group refined leaves the
// bucket out. A later import that reads the bucket puts all of it back.
func TestImportInfra_UnreadableBucket(t *testing.T) {
	openTestDB(t)
	conns := []model.ConnectionInfo{
		testConn("bucket-far", cspGroup, common.ResourceTypeObjectStorage, ""),
		testConn("bucket-near", cspGroup, common.ResourceTypeObjectStorage, ""),
	}
	conns[0].ResourceID, conns[1].ResourceID = "bucket-far", "bucket-near"
	conns[0].ConnectionStatus, conns[1].ConnectionStatus = model.ConnectionInfoStatusSuccess, model.ConnectionInfoStatusSuccess
	createGroup(t, cspGroup, conns)
	details := map[string]*spider.S3BucketDetail{
		"bucket-far":  unreadableBucket("bucket-far"),
		"bucket-near": partlyReadBucket("bucket-near"),
	}
	stubRefreshBuckets(t, details)

	rec := connRequest(t, http.MethodPost, cspGroup.ID, "bucket-far", ImportInfra)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "none of the details") {
		t.Errorf("single import: %d %s, want 400 with the reason", rec.Code, rec.Body.String())
	}

	infraRec, err := dao.SavedInfraInfoGet("bucket-far")
	if err != nil {
		t.Fatal(err)
	}
	dataRec, err := dao.SavedDataInfoGet("bucket-far")
	if err != nil {
		t.Fatal(err)
	}
	if infraRec.Status != model.ConnectionInfoStatusFailed || dataRec.Status != model.ConnectionInfoStatusFailed {
		t.Errorf("infra status %q, data status %q, want both failed", infraRec.Status, dataRec.Status)
	}
	ci := getConnection(t, "bucket-far")
	if ci.ConnectionStatus != model.ConnectionInfoStatusFailed || !strings.Contains(ci.ConnectionFailedMessage, redirectErr) {
		t.Errorf("connection status %q (%q), want failed with the reason", ci.ConnectionStatus, ci.ConnectionFailedMessage)
	}

	got, err := importInfraGroup(cspGroup, conns, importInfraOptions{})
	if err != nil {
		t.Fatalf("group import: err = %v, want nil", err)
	}
	status := map[string]string{}
	for _, r := range got {
		status[r.ConnectionID] = r.Status
		if r.ConnectionID == "bucket-far" && !strings.Contains(r.ErrorMessage, "none of the details") {
			t.Errorf("group import: error_message = %q, want the reason", r.ErrorMessage)
		}
	}
	if status["bucket-far"] != "failed" || status["bucket-near"] != "success" {
		t.Errorf("group import statuses = %v", status)
	}
	if ci := getConnection(t, "bucket-near"); ci.ConnectionStatus != model.ConnectionInfoStatusSuccess || ci.ConnectionFailedMessage != "" {
		t.Errorf("readable bucket connection: %q (%q), want success", ci.ConnectionStatus, ci.ConnectionFailedMessage)
	}

	var m storagemodel.SourceObjectStorageModel
	rec = groupRequest(t, cspGroup.ID, GetDataInfoSourceGroupRefined)
	if rec.Code != http.StatusOK {
		t.Fatalf("group refined: %d %s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		t.Fatal(err)
	}
	if b := m.SourceObjectStorageModel.SourceObjectStorages; len(b) != 1 || b[0].BucketName != "bucket-near" {
		t.Errorf("group refined buckets = %#v, want bucket-near only", b)
	}
	rec = connRequest(t, http.MethodGet, cspGroup.ID, "bucket-far", GetDataInfoRefined)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "did not succeed") {
		t.Errorf("single refined: %d %s, want 400", rec.Code, rec.Body.String())
	}

	details["bucket-far"] = partlyReadBucket("bucket-far")
	if _, err := doImportInfra("bucket-far", importInfraOptions{}); err != nil {
		t.Fatalf("re-import: %v", err)
	}
	if ci := getConnection(t, "bucket-far"); ci.ConnectionStatus != model.ConnectionInfoStatusSuccess || ci.ConnectionFailedMessage != "" {
		t.Errorf("after re-import: connection %q (%q), want success with no message", ci.ConnectionStatus, ci.ConnectionFailedMessage)
	}
	if _, err := doGetRefinedBucket("bucket-far"); err != nil {
		t.Errorf("after re-import: refined err = %v", err)
	}
}

// A failed CSP collection marks the resource's own table, keeping its data,
// and creates a failed record when there was none; the refined paths then
// leave the resource out.
func TestDoImportInfra_RefreshFailMarksResourceTable(t *testing.T) {
	openTestDB(t)
	conns := []model.ConnectionInfo{
		testConn("k8s", cspGroup, common.ResourceTypeK8s, ""),
		testConn("rdbms", cspGroup, common.ResourceTypeRDBMS, ""),
		testConn("bucket", cspGroup, common.ResourceTypeObjectStorage, ""),
		testConn("rdbms-new", cspGroup, common.ResourceTypeRDBMS, ""),
		testConn("k8s-new", cspGroup, common.ResourceTypeK8s, ""),
		testConn("bucket-new", cspGroup, common.ResourceTypeObjectStorage, ""),
	}
	for i := range conns {
		conns[i].ResourceID = conns[i].ID
	}
	createGroup(t, cspGroup, conns)

	k8sData := kubernetes.Kubernetes{Cluster: kubernetes.Cluster{Name: "c", Version: "1.30"}}
	if err := upsertSavedK8s("k8s", k8sData); err != nil {
		t.Fatal(err)
	}
	if err := upsertSavedRDBMS("rdbms", model.RDBMSInfo{RDBMS: &spider.RDBMSInfo{IId: spider.IID{SystemId: "db-1"}, DBEngine: "mysql"}}); err != nil {
		t.Fatal(err)
	}
	if err := storeBucketDetail("bucket", "ap-northeast-2", partlyReadBucket("bucket")); err != nil {
		t.Fatal(err)
	}
	if tryGetKubernetesInfo("k8s") == nil {
		t.Fatal("a successful record must be read before the failure")
	}
	if _, err := doGetRefinedRDBMSInfo("rdbms"); err != nil {
		t.Fatalf("rdbms refined before the failure: %v", err)
	}

	stubRefreshCSP(t, map[string]bool{"k8s": true, "rdbms": true, "bucket": true,
		"rdbms-new": true, "k8s-new": true, "bucket-new": true})
	for _, c := range conns {
		if _, err := doImportInfra(c.ID, importInfraOptions{}); err == nil {
			t.Errorf("%s: import err = nil, want the refresh error", c.ID)
		}
		if ci := getConnection(t, c.ID); ci.ConnectionStatus != model.ConnectionInfoStatusFailed ||
			!strings.Contains(ci.ConnectionFailedMessage, "not found") {
			t.Errorf("%s: connection %q (%q), want failed with the reason", c.ID, ci.ConnectionStatus, ci.ConnectionFailedMessage)
		}
	}

	k, err := dao.SavedKubernetesInfoGet("k8s")
	if err != nil {
		t.Fatal(err)
	}
	if k.Status != model.ConnectionInfoStatusFailed || !strings.Contains(k.KubernetesData, "1.30") {
		t.Errorf("k8s record = %#v, want failed with the data kept", k)
	}
	r, err := dao.SavedRDBMSInfoGet("rdbms")
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != model.ConnectionInfoStatusFailed || !strings.Contains(r.RDBMSData, "db-1") {
		t.Errorf("rdbms record = %#v, want failed with the data kept", r)
	}
	d, err := dao.SavedDataInfoGet("bucket")
	if err != nil {
		t.Fatal(err)
	}
	if d.Status != model.ConnectionInfoStatusFailed || d.CSPData == "" {
		t.Errorf("bucket record = %#v, want failed with the data kept", d)
	}
	if k, err := dao.SavedKubernetesInfoGet("k8s-new"); err != nil || k.Status != model.ConnectionInfoStatusFailed {
		t.Errorf("k8s-new record = %#v, %v, want a failed record", k, err)
	}
	if r, err := dao.SavedRDBMSInfoGet("rdbms-new"); err != nil || r.Status != model.ConnectionInfoStatusFailed {
		t.Errorf("rdbms-new record = %#v, %v, want a failed record", r, err)
	}
	if d, err := dao.SavedDataInfoGet("bucket-new"); err != nil || d.Status != model.ConnectionInfoStatusFailed {
		t.Errorf("bucket-new record = %#v, %v, want a failed record", d, err)
	}

	if got := tryGetKubernetesInfo("k8s"); got != nil {
		t.Errorf("k8s: got %#v, want nil for a failed record", got)
	}
	if _, err := doGetRefinedRDBMSInfo("rdbms"); !errors.Is(err, errRDBMSNotCollected) || !strings.Contains(err.Error(), "did not succeed") {
		t.Errorf("rdbms refined: err = %v, want a not-succeeded errRDBMSNotCollected", err)
	}
	if _, err := doGetRefinedRDBMSInfo("rdbms-new"); !errors.Is(err, errRDBMSNotCollected) {
		t.Errorf("rdbms-new refined: err = %v, want errRDBMSNotCollected", err)
	}
	for _, id := range []string{"bucket", "bucket-new"} {
		if _, err := doGetRefinedBucket(id); !errors.Is(err, errBucketCollectFailed) {
			t.Errorf("%s refined: err = %v, want errBucketCollectFailed", id, err)
		}
	}

	var m rdbmsmodel.SourceRDBMSModel
	rec := groupRequest(t, cspGroup.ID, GetRDBMSInfoSourceGroupRefined)
	if rec.Code != http.StatusOK {
		t.Fatalf("group rdbms refined: %d %s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		t.Fatal(err)
	}
	if n := len(m.SourceRDBMSModel.SourceRDBMSInstances); n != 0 {
		t.Errorf("group rdbms refined: %d instances, want 0", n)
	}
}

// An agent record whose software import failed or is still running is left
// out of the group rdbms refined instead of failing it on its empty data.
func TestGetRDBMSInfoSourceGroupRefined_SkipsUnsuccessfulSoftware(t *testing.T) {
	openTestDB(t)
	createGroup(t, onpremGroup, []model.ConnectionInfo{
		testConn("vm-failed", onpremGroup, common.ResourceTypeVM, "192.0.2.10"),
		testConn("vm-importing", onpremGroup, common.ResourceTypeVM, "192.0.2.11"),
	})
	saveSoftware(t, "vm-failed", "failed", "")
	saveSoftware(t, "vm-importing", "importing", "")

	rec := groupRequest(t, onpremGroup.ID, GetRDBMSInfoSourceGroupRefined)
	if rec.Code != http.StatusOK {
		t.Errorf("group rdbms refined: %d %s, want 200", rec.Code, rec.Body.String())
	}
	if _, err := doGetRefinedRDBMSInfo("vm-failed"); err == nil || !strings.Contains(err.Error(), "did not succeed") {
		t.Errorf("single: err = %v, want the import named as not succeeded", err)
	}
}

// A software record that is a success but holds data that does not parse is
// reported in rdbms terms.
func TestDoGetInstalledSoftwareForRDBMS_ParseErrorWording(t *testing.T) {
	openTestDB(t)
	saveSoftware(t, "vm", "success", "{not json")

	_, err := doGetInstalledSoftwareForRDBMS("vm")
	if err == nil || !strings.Contains(err.Error(), "to find the installed rdbms") {
		t.Errorf("err = %v, want the rdbms named", err)
	}
}

// Importing a resource other than a VM again moves saved_time even when the
// record is already a success.
func TestDoImportInfra_ReimportRefreshesSavedTime(t *testing.T) {
	openTestDB(t)
	conns := []model.ConnectionInfo{testConn("bucket", cspGroup, common.ResourceTypeObjectStorage, "")}
	conns[0].ResourceID = "bucket"
	createGroup(t, cspGroup, conns)
	old := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	if err := db.DB.Create(&model.SavedInfraInfo{ConnectionID: "bucket", Status: "success", SavedTime: old}).Error; err != nil {
		t.Fatal(err)
	}
	stubRefreshCSP(t, nil)

	got, err := doImportInfra("bucket", importInfraOptions{})
	if err != nil {
		t.Fatal(err)
	}
	stored, err := dao.SavedInfraInfoGet("bucket")
	if err != nil {
		t.Fatal(err)
	}
	if !got.SavedTime.After(old) || !stored.SavedTime.After(old) {
		t.Errorf("saved_time returned %v, stored %v, want later than %v", got.SavedTime, stored.SavedTime, old)
	}
}

// When the secrets cannot be loaded the import ends as failed rather than
// leaving the record it registered as importing.
func TestDoImport_SecretFailureMarksFailed(t *testing.T) {
	openTestDB(t)
	createGroup(t, onpremGroup, []model.ConnectionInfo{testConn("vm", onpremGroup, common.ResourceTypeVM, "192.0.2.10")})

	orig := hydrateSecrets
	t.Cleanup(func() { hydrateSecrets = orig })
	hydrateSecrets = func(*model.ConnectionInfo) error { return errors.New("openbao: sealed") }

	imports := map[string]func() (string, error){
		"infra": func() (string, error) {
			_, err := doImportInfra("vm", importInfraOptions{})
			r, _ := dao.SavedInfraInfoGet("vm")
			return r.Status, err
		},
		"software": func() (string, error) {
			_, err := doImportSoftware("vm", false)
			r, _ := dao.SavedSoftwareInfoGet("vm")
			return r.Status, err
		},
		"kubernetes": func() (string, error) {
			_, err := doImportKubernetes("vm")
			r, _ := dao.SavedKubernetesInfoGet("vm")
			return r.Status, err
		},
		"helm": func() (string, error) {
			_, err := doImportHelm("vm")
			r, _ := dao.SavedHelmInfoGet("vm")
			return r.Status, err
		},
		"data": func() (string, error) {
			_, err := doImportData("vm")
			r, _ := dao.SavedDataInfoGet("vm")
			return r.Status, err
		},
	}
	for name, do := range imports {
		status, err := do()
		if err == nil || !strings.Contains(err.Error(), "sealed") {
			t.Errorf("%s: err = %v, want the secret error", name, err)
		}
		if status != model.ConnectionInfoStatusFailed {
			t.Errorf("%s: stored status = %q, want failed", name, status)
		}
	}
}
