package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	storagemodel "github.com/cloud-barista/cm-beetle/imdl/storage-model"
	"github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/cloud-barista/cm-honeybee/server/dao"
	"github.com/cloud-barista/cm-honeybee/server/db"
	"github.com/cloud-barista/cm-honeybee/server/lib/spider"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
)

// osakaConn is the connection of a source group bound to ap-northeast-3.
var osakaConn = spider.Conn{Provider: "aws", Region: "ap-northeast-3"}

// osakaGroup is a csp source group in ap-northeast-3.
var osakaGroup = &model.SourceGroup{ID: "sg-osaka", Name: "sg-osaka", Type: common.SourceGroupTypeCSP,
	ProviderName: "aws", RegionName: "ap-northeast-3"}

// stubBucketCSP stands in for the bucket calls of the CSP. The account holds
// the buckets of regions, a bucket mapped to "" is one whose region cannot be
// told, and one listed in lookupErr fails its region lookup. Detail calls
// record the region they were made in.
func stubBucketCSP(t *testing.T, regions map[string]string, lookupErr map[string]bool, known bool) map[string]string {
	t.Helper()

	origList, origLoc, origRegions, origDetail := listS3Buckets, getS3BucketLocation, getS3BucketRegions, getS3BucketDetail
	t.Cleanup(func() {
		listS3Buckets, getS3BucketLocation, getS3BucketRegions, getS3BucketDetail = origList, origLoc, origRegions, origDetail
	})

	answer := func(name string) spider.S3BucketRegion {
		if lookupErr[name] {
			return spider.S3BucketRegion{Err: errors.New("failed to read the region of bucket '" + name + "'")}
		}
		return spider.S3BucketRegion{Region: regions[name]}
	}
	listS3Buckets = func(spider.Conn) ([]spider.S3BucketInfo, error) {
		var out []spider.S3BucketInfo
		for _, name := range []string{"near", "far", "broken", "unknown"} {
			if _, has := regions[name]; has {
				out = append(out, spider.S3BucketInfo{Name: name, CreationDate: "2025-01-02T03:04:05Z"})
			}
		}
		return out, nil
	}
	getS3BucketRegions = func(_ spider.Conn, names []string) (map[string]spider.S3BucketRegion, error) {
		if !known {
			return nil, nil
		}
		out := map[string]spider.S3BucketRegion{}
		for _, name := range names {
			out[name] = answer(name)
		}
		return out, nil
	}
	getS3BucketLocation = func(_ spider.Conn, name string) (*spider.S3BucketInfo, error) {
		r := answer(name)
		if !known {
			r = spider.S3BucketRegion{}
		}
		if r.Err != nil {
			return nil, r.Err
		}
		return &spider.S3BucketInfo{Name: name, Region: r.Region}, nil
	}
	detailRegion := map[string]string{}
	getS3BucketDetail = func(c spider.Conn, name string, _ bool) (*spider.S3BucketDetail, error) {
		detailRegion[name] = c.Region
		return &spider.S3BucketDetail{Name: name, ObjectsCounted: true, ObjectCount: 1, ItemsTried: 6, ItemsRead: 6}, nil
	}

	return detailRegion
}

// Discovery lists the buckets of the group's region with their region, leaves
// out the ones of another region, and keeps a bucket whose lookup failed with
// the reason. Where the CSP cannot tell regions, every bucket is listed.
func TestDiscoverBuckets_GroupRegionOnly(t *testing.T) {
	stubBucketCSP(t, map[string]string{"near": "ap-northeast-3", "far": "ap-northeast-2", "broken": ""},
		map[string]bool{"broken": true}, true)

	items, err := discoverByType(osakaConn, common.ResourceTypeObjectStorage)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]model.DiscoveredResource{}
	for _, it := range items {
		got[it.ResourceID] = it
	}
	if len(items) != 2 {
		t.Errorf("items = %#v, want near and broken", items)
	}
	if _, has := got["far"]; has {
		t.Error("a bucket of another region was listed")
	}
	if it := got["near"]; it.Region != "ap-northeast-3" || it.Extra["creation_date"] == "" || it.Extra["region_error"] != "" {
		t.Errorf("near = %#v, want region ap-northeast-3 and no region error", it)
	}
	if it := got["broken"]; it.Region != "" || !strings.Contains(it.Extra["region_error"], "'broken'") {
		t.Errorf("broken = %#v, want no region and the lookup error", it)
	}

	stubBucketCSP(t, map[string]string{"near": "", "far": ""}, nil, false)
	items, err = discoverByType(osakaConn, common.ResourceTypeObjectStorage)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].Region != "" || items[1].Region != "" {
		t.Errorf("unknown regions: items = %#v, want both buckets without a region", items)
	}
}

// Registration refuses a bucket of another region with both regions in the
// reason, and accepts a bucket of the group's region, of an unknown one, or
// whose region lookup failed (left unknown, as discovery lists it).
func TestCheckBucket_RegionMismatch(t *testing.T) {
	stubBucketCSP(t, map[string]string{"near": "ap-northeast-3", "far": "ap-northeast-2", "unknown": "", "broken": ""},
		map[string]bool{"broken": true}, true)

	_, err := checkBucket(osakaConn, "far")
	want := "bucket 'far' is in region 'ap-northeast-2', source group region is 'ap-northeast-3'"
	if err == nil || err.Error() != want {
		t.Errorf("far: err = %v, want %q", err, want)
	}
	if b, err := checkBucket(osakaConn, "near"); err != nil || b.Region != "ap-northeast-3" {
		t.Errorf("near: %+v, %v; want accepted with its region", b, err)
	}
	if b, err := checkBucket(osakaConn, "unknown"); err != nil || b.Region != "" {
		t.Errorf("unknown: %+v, %v; want accepted without a region", b, err)
	}
	if b, err := checkBucket(osakaConn, "broken"); err != nil || b.Region != "" {
		t.Errorf("broken: %+v, %v; want accepted without a region", b, err)
	}
	if _, err := checkBucket(osakaConn, "absent"); err == nil || !strings.Contains(err.Error(), "was not found") {
		t.Errorf("absent: err = %v, want not found", err)
	}
}

// Import reads a bucket at its region's endpoint and stores that region in the
// data address and in csp_data; refined names it as the source region. A
// bucket of another region fails the import with the reason, on the
// connection status too, and nothing is read from it.
func TestCollectBucket_StoresBucketRegion(t *testing.T) {
	openTestDB(t)
	createGroup(t, osakaGroup, []model.ConnectionInfo{
		testConn("near", osakaGroup, common.ResourceTypeObjectStorage, ""),
		testConn("far", osakaGroup, common.ResourceTypeObjectStorage, ""),
	})
	detailRegion := stubBucketCSP(t, map[string]string{"near": "ap-northeast-3", "far": "ap-northeast-2"}, nil, true)

	if err := collectBucket(osakaConn, "near", "near", true); err != nil {
		t.Fatal(err)
	}
	if detailRegion["near"] != "ap-northeast-3" {
		t.Errorf("detail read in %q, want ap-northeast-3", detailRegion["near"])
	}
	got, err := doGetDataInfo("near")
	if err != nil {
		t.Fatal(err)
	}
	if got.Address != "ap-northeast-3" || len(got.Buckets) != 1 || got.Buckets[0].Objects != 1 {
		t.Errorf("data = %#v, want address ap-northeast-3 and one object", got)
	}
	saved, err := dao.SavedDataInfoGet("near")
	if err != nil {
		t.Fatal(err)
	}
	var stored map[string]any
	if err := json.Unmarshal([]byte(saved.CSPData), &stored); err != nil {
		t.Fatal(err)
	}
	if stored["region"] != "ap-northeast-3" || stored["bucketName"] != "near" {
		t.Errorf("csp_data = %s, want bucketName near and region ap-northeast-3", saved.CSPData)
	}

	rec := connRequest(t, http.MethodGet, osakaGroup.ID, "near", GetDataInfoRefined)
	var m storagemodel.SourceObjectStorageModel
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		t.Fatalf("%d %s: %v", rec.Code, rec.Body.String(), err)
	}
	if sc := m.SourceObjectStorageModel.SourceCloud; sc == nil || sc.Csp != "aws" || sc.Region != "ap-northeast-3" {
		t.Errorf("refined sourceCloud = %#v, want aws ap-northeast-3", sc)
	}

	orig := refreshCSP
	t.Cleanup(func() { refreshCSP = orig })
	refreshCSP = func(_ *model.SourceGroup, ci *model.ConnectionInfo, opts importInfraOptions) error {
		return collectBucket(osakaConn, ci.ID, ci.ResourceID, opts.CountObjects)
	}
	if err := db.DB.Model(&model.ConnectionInfo{}).Where("id = ?", "far").Update("resource_id", "far").Error; err != nil {
		t.Fatal(err)
	}
	reason := "bucket 'far' is in region 'ap-northeast-2', source group region is 'ap-northeast-3'"
	if _, err := doImportInfra("far", importInfraOptions{}); err == nil || !strings.Contains(err.Error(), reason) {
		t.Errorf("import far: err = %v, want the region reason", err)
	}
	if _, read := detailRegion["far"]; read {
		t.Error("a bucket of another region was read")
	}
	if ci := getConnection(t, "far"); ci.ConnectionStatus != model.ConnectionInfoStatusFailed || ci.ConnectionFailedMessage != reason {
		t.Errorf("far connection: %q (%q), want failed with the reason", ci.ConnectionStatus, ci.ConnectionFailedMessage)
	}
	if rec, _ := dao.SavedDataInfoGet("far"); rec == nil || rec.Status != model.ConnectionInfoStatusFailed {
		t.Errorf("far data record = %#v, want failed", rec)
	}
}

// The group refined names the group's region and holds only buckets found
// there or of an unknown region; a record of another region, stored before
// import refused one, is left out.
func TestGetDataInfoSourceGroupRefined_BucketRegions(t *testing.T) {
	openTestDB(t)
	createGroup(t, osakaGroup, []model.ConnectionInfo{
		testConn("near", osakaGroup, common.ResourceTypeObjectStorage, ""),
		testConn("legacy", osakaGroup, common.ResourceTypeObjectStorage, ""),
		testConn("far", osakaGroup, common.ResourceTypeObjectStorage, ""),
	})
	read := &spider.S3BucketDetail{ItemsTried: 1, ItemsRead: 1}
	for id, region := range map[string]string{"near": "ap-northeast-3", "legacy": "", "far": "ap-northeast-2"} {
		d := *read
		d.Name = id
		if err := storeBucketDetail(id, "ap-northeast-3", region, &d); err != nil {
			t.Fatal(err)
		}
	}

	rec := groupRequest(t, osakaGroup.ID, GetDataInfoSourceGroupRefined)
	var m storagemodel.SourceObjectStorageModel
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		t.Fatalf("%d %s: %v", rec.Code, rec.Body.String(), err)
	}
	if sc := m.SourceObjectStorageModel.SourceCloud; sc == nil || sc.Region != "ap-northeast-3" {
		t.Errorf("sourceCloud = %#v, want ap-northeast-3", sc)
	}
	names := map[string]bool{}
	for _, b := range m.SourceObjectStorageModel.SourceObjectStorages {
		names[b.BucketName] = true
	}
	if len(names) != 2 || !names["near"] || !names["legacy"] {
		t.Errorf("buckets = %v, want near and legacy", names)
	}

	rec = connRequest(t, http.MethodGet, osakaGroup.ID, "far", GetDataInfoRefined)
	m = storagemodel.SourceObjectStorageModel{}
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		t.Fatalf("%d %s: %v", rec.Code, rec.Body.String(), err)
	}
	if sc := m.SourceObjectStorageModel.SourceCloud; sc == nil || sc.Region != "ap-northeast-2" {
		t.Errorf("single refined of far: sourceCloud = %#v, want the bucket's region", sc)
	}
}

// A bucket whose region lookup fails is still collected, as before regions
// were read: it is read at the group's region, its data address is the group's
// region, csp_data holds no region, and refined names the group's region.
func TestCollectBucket_RegionLookupFailedIsBestEffort(t *testing.T) {
	openTestDB(t)
	createGroup(t, osakaGroup, []model.ConnectionInfo{
		testConn("broken", osakaGroup, common.ResourceTypeObjectStorage, ""),
	})
	detailRegion := stubBucketCSP(t, map[string]string{"broken": ""}, map[string]bool{"broken": true}, true)

	if err := collectBucket(osakaConn, "broken", "broken", true); err != nil {
		t.Fatalf("err = %v, want the bucket collected", err)
	}
	if detailRegion["broken"] != "ap-northeast-3" {
		t.Errorf("detail read in %q, want the group region ap-northeast-3", detailRegion["broken"])
	}
	got, err := doGetDataInfo("broken")
	if err != nil {
		t.Fatal(err)
	}
	if got.Address != "ap-northeast-3" {
		t.Errorf("address = %q, want ap-northeast-3", got.Address)
	}
	saved, err := dao.SavedDataInfoGet("broken")
	if err != nil {
		t.Fatal(err)
	}
	if saved.Status != model.ConnectionInfoStatusSuccess {
		t.Errorf("status = %q, want success", saved.Status)
	}
	var stored map[string]any
	if err := json.Unmarshal([]byte(saved.CSPData), &stored); err != nil {
		t.Fatal(err)
	}
	if _, has := stored["region"]; has {
		t.Errorf("csp_data = %s, want no region", saved.CSPData)
	}

	rec := connRequest(t, http.MethodGet, osakaGroup.ID, "broken", GetDataInfoRefined)
	var m storagemodel.SourceObjectStorageModel
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		t.Fatalf("%d %s: %v", rec.Code, rec.Body.String(), err)
	}
	if sc := m.SourceObjectStorageModel.SourceCloud; sc == nil || sc.Region != "ap-northeast-3" {
		t.Errorf("refined sourceCloud = %#v, want the group region", sc)
	}
}
