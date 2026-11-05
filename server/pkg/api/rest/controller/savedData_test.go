package controller

import (
	"testing"

	"github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/cloud-barista/cm-honeybee/server/db"
	"github.com/cloud-barista/cm-honeybee/server/lib/spider"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
)

func createConnection(t *testing.T, sg *model.SourceGroup, conn *model.ConnectionInfo) {
	t.Helper()

	if err := db.DB.Create(sg).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.DB.Create(conn).Error; err != nil {
		t.Fatal(err)
	}
}

// The stored data record keeps the wrapper ({"minio": {...}}) that both the
// agent's /data response and the CSP bucket adapter produce, so the reader has
// to unwrap it rather than parse the record as the inner MinIO data.
func TestDoGetDataInfo_ReadsStoredWrapper(t *testing.T) {
	openTestDB(t)
	createConnection(t,
		&model.SourceGroup{ID: "sg1", Name: "sg1", Type: common.SourceGroupTypeCSP},
		&model.ConnectionInfo{ID: "c1", Name: "c1", SourceGroupID: "sg1"})

	d := &spider.S3BucketDetail{Name: "bucket-a"}
	if err := upsertSavedData("c1", bucketToData("ap-northeast-2", d), nil); err != nil {
		t.Fatal(err)
	}

	got, err := doGetDataInfo("c1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Buckets) != 1 || got.Buckets[0].Name != "bucket-a" {
		t.Errorf("buckets = %#v, want one bucket named bucket-a", got.Buckets)
	}
	if got.Address != "ap-northeast-2" {
		t.Errorf("address = %q, want ap-northeast-2", got.Address)
	}
}

// A record that holds no MinIO section reads back as empty data, not an error.
func TestDoGetDataInfo_NoMinIOSection(t *testing.T) {
	openTestDB(t)
	createConnection(t,
		&model.SourceGroup{ID: "sg1", Name: "sg1"},
		&model.ConnectionInfo{ID: "c1", Name: "c1", SourceGroupID: "sg1"})
	if err := db.DB.Create(&model.SavedDataInfo{ConnectionID: "c1", DataData: "{}", Status: "success"}).Error; err != nil {
		t.Fatal(err)
	}

	got, err := doGetDataInfo("c1")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || len(got.Buckets) != 0 {
		t.Errorf("got %#v, want empty data", got)
	}
}

// A CSP resource without an in-guest agent (an object-storage bucket has no IP)
// ends the infra import after the CSP collection, and the record it returns
// must say the import finished.
func TestDoImportInfra_CSPWithoutAgentFinishes(t *testing.T) {
	openTestDB(t)
	createConnection(t,
		&model.SourceGroup{ID: "sg1", Name: "sg1", Type: common.SourceGroupTypeCSP},
		&model.ConnectionInfo{ID: "c1", Name: "c1", SourceGroupID: "sg1",
			ResourceType: common.ResourceTypeObjectStorage, ResourceID: "bucket-a"})

	orig := refreshCSP
	t.Cleanup(func() { refreshCSP = orig })
	refreshCSP = func(_ *model.SourceGroup, ci *model.ConnectionInfo, _ importInfraOptions) error {
		return upsertSavedData(ci.ID, bucketToData("", &spider.S3BucketDetail{Name: "bucket-a"}), nil)
	}

	got, err := doImportInfra("c1", importInfraOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != model.ConnectionInfoStatusSuccess {
		t.Errorf("status = %q, want %q", got.Status, model.ConnectionInfoStatusSuccess)
	}
}

// The bucket stored by an object-storage import reads back through the refined
// path as the cm-beetle source property, and the totals are left out when the
// objects were not counted.
func TestDoGetRefinedBucket(t *testing.T) {
	openTestDB(t)
	createConnection(t,
		&model.SourceGroup{ID: "sg1", Name: "sg1", Type: common.SourceGroupTypeCSP},
		&model.ConnectionInfo{ID: "c1", Name: "c1", SourceGroupID: "sg1"})

	d := &spider.S3BucketDetail{
		Name:              "bucket-a",
		VersioningEnabled: true,
		CORSRules:         []spider.S3CORSRule{{AllowedOrigin: []string{"*"}, AllowedMethod: []string{"GET"}}},
		EncryptionEnabled: true,
		TotalSizeBytes:    10,
		ObjectCount:       2,
		ObjectsCounted:    false,
	}
	if err := upsertSavedData("c1", bucketToData("us-east-1", d), bucketToSourceProperty(d)); err != nil {
		t.Fatal(err)
	}

	p, err := doGetRefinedBucket("c1")
	if err != nil {
		t.Fatal(err)
	}
	if p.BucketName != "bucket-a" || !p.VersioningEnabled || !p.CORSEnabled || len(p.CORSRule) != 1 || !p.EncryptionEnabled || p.IsPublic {
		t.Errorf("unexpected property %#v", p)
	}
	if p.ObjectCount != 0 || p.TotalSizeBytes != 0 {
		t.Errorf("uncounted totals must stay zero, got %d objects %d bytes", p.ObjectCount, p.TotalSizeBytes)
	}

	if _, err := doGetRefinedBucket("no-such"); err != errBucketNotCollected {
		t.Errorf("missing record: got %v, want errBucketNotCollected", err)
	}
}
