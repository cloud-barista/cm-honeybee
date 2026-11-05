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

	b := &spider.S3BucketInfo{Name: "bucket-a", Region: "ap-northeast-2"}
	if err := upsertSavedData("c1", bucketToData(b)); err != nil {
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
