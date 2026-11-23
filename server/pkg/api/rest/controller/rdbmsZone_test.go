package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	rdbmsmodel "github.com/cloud-barista/cm-beetle/imdl/rdbms-model"
	"github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/cloud-barista/cm-honeybee/server/lib/spider"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
)

// The zone keys below are the ones each driver was seen to put in an
// instance's KeyValueList.
func TestRDBMSZone(t *testing.T) {
	tests := []struct {
		name string
		kvs  []spider.KeyValue
		want string
	}{
		{"aws", []spider.KeyValue{{Key: "AvailabilityZone", Value: "ap-northeast-1d"}}, "ap-northeast-1d"},
		{"alibaba", []spider.KeyValue{{Key: "RegionId", Value: "ap-northeast-2"}, {Key: "ZoneId", Value: "ap-northeast-2a"}}, "ap-northeast-2a"},
		{"gcp", []spider.KeyValue{{Key: "Region", Value: "asia-northeast3"}, {Key: "GceZone", Value: "asia-northeast3-c"}}, "asia-northeast3-c"},
		// Tencent's ZoneId is a number and comes first; Zone holds the name.
		{"tencent", []spider.KeyValue{{Key: "ZoneId", Value: "180001"}, {Key: "Zone", Value: "ap-seoul-1"}}, "ap-seoul-1"},
		{"tencent numeric id only", []spider.KeyValue{{Key: "ZoneId", Value: "180001"}}, ""},
		{"azure", []spider.KeyValue{{Key: "Location", Value: "koreacentral"}}, ""},
		{"ibm", []spider.KeyValue{{Key: "RegionID", Value: "us-south"}}, ""},
		{"key case ignored", []spider.KeyValue{{Key: "availabilityzone", Value: " ap-northeast-1a "}}, "ap-northeast-1a"},
		{"empty value", []spider.KeyValue{{Key: "AvailabilityZone", Value: ""}, {Key: "Zone", Value: "z-1"}}, "z-1"},
		{"no record", nil, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := rdbmsZone(&spider.RDBMSInfo{KeyValueList: tc.kvs}); got != tc.want {
				t.Errorf("rdbmsZone = %q, want %q", got, tc.want)
			}
		})
	}

	if got := rdbmsZone(nil); got != "" {
		t.Errorf("rdbmsZone(nil) = %q, want empty", got)
	}
}

// stubRDBMSCSP stands in for the managed database calls of the CSP. The listing
// holds db, and the spec is found only in specZone, as alibaba finds it only in
// a zone that offers the engine. The zones the spec was asked in are recorded.
func stubRDBMSCSP(t *testing.T, db spider.RDBMSInfo, specZone string) *[]string {
	t.Helper()

	origList, origSpec := listCSPRDBMS, getCSPDBSpec
	t.Cleanup(func() { listCSPRDBMS, getCSPDBSpec = origList, origSpec })

	asked := &[]string{}
	listCSPRDBMS = func(_ spider.Conn) ([]spider.RDBMSInfo, error) {
		return []spider.RDBMSInfo{db}, nil
	}
	getCSPDBSpec = func(c spider.Conn, _ string, name string) (*spider.DBSpecInfo, error) {
		*asked = append(*asked, c.Zone)
		if c.Zone != specZone {
			return nil, errors.New("DescribeAvailableZones returned no engine versions")
		}
		return &spider.DBSpecInfo{Name: name, VCpu: spider.VCpuInfo{Count: "2"}, MemSizeMiB: "4096"}, nil
	}

	return asked
}

func TestCollectRDBMS_AsksSpecInDBZone(t *testing.T) {
	groupConn := spider.Conn{Provider: "alibaba", Region: "ap-northeast-2", Zone: "ap-northeast-2c"}

	t.Run("database in another zone", func(t *testing.T) {
		db := spider.RDBMSInfo{
			IId:          spider.IID{SystemId: "rm-1"},
			DBEngine:     "MySQL",
			DBSpec:       "mysql.n2.medium.1",
			KeyValueList: []spider.KeyValue{{Key: "ZoneId", Value: "ap-northeast-2a"}},
		}
		asked := stubRDBMSCSP(t, db, "ap-northeast-2a")

		payload, err := collectRDBMS(groupConn, "rm-1")
		if err != nil {
			t.Fatal(err)
		}
		if len(*asked) != 1 || (*asked)[0] != "ap-northeast-2a" {
			t.Errorf("spec asked in zones %v, want [ap-northeast-2a]", *asked)
		}
		if payload.DBSpec == nil || payload.DBSpec.VCpu.Count != "2" || payload.DBSpec.MemSizeMiB != "4096" {
			t.Errorf("DBSpec = %+v, want the spec found in the database's zone", payload.DBSpec)
		}
	})

	// A CSP that reports no zone of the database is asked in the group's zone.
	t.Run("no zone reported", func(t *testing.T) {
		db := spider.RDBMSInfo{
			IId:          spider.IID{SystemId: "db-1"},
			DBEngine:     "mysql",
			DBSpec:       "GP_Gen5_2",
			KeyValueList: []spider.KeyValue{{Key: "Location", Value: "koreacentral"}},
		}
		asked := stubRDBMSCSP(t, db, groupConn.Zone)

		payload, err := collectRDBMS(groupConn, "db-1")
		if err != nil {
			t.Fatal(err)
		}
		if len(*asked) != 1 || (*asked)[0] != groupConn.Zone {
			t.Errorf("spec asked in zones %v, want [%s]", *asked, groupConn.Zone)
		}
		if payload.DBSpec == nil {
			t.Error("DBSpec = nil, want the spec found in the group's zone")
		}
	})
}

// zonedGroup is a csp source group registered with zone ap-northeast-2c.
var zonedGroup = &model.SourceGroup{ID: "sg-zoned", Name: "sg-zoned", Type: common.SourceGroupTypeCSP,
	ProviderName: "alibaba", RegionName: "ap-northeast-2/ap-northeast-2c"}

func saveZonedRDBMS(t *testing.T, connID string, kvs []spider.KeyValue) {
	t.Helper()

	db := &spider.RDBMSInfo{IId: spider.IID{SystemId: connID}, DBEngine: "mysql", KeyValueList: kvs}
	if err := upsertSavedRDBMS(connID, model.RDBMSInfo{RDBMS: db}); err != nil {
		t.Fatal(err)
	}
}

func refinedRDBMSCloud(t *testing.T, body []byte) *rdbmsmodel.CloudProperty {
	t.Helper()

	var m rdbmsmodel.SourceRDBMSModel
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("%s: %v", body, err)
	}
	if m.SourceRDBMSModel.SourceCloud == nil {
		t.Fatalf("sourceCloud = nil in %s", body)
	}
	return m.SourceRDBMSModel.SourceCloud
}

// The refined sourceCloud names the zone the database runs in. The group result
// does so only when every database reported the same zone, and otherwise names
// the group's zone, since the model has one zone for all of its databases.
func TestGetRDBMSInfoRefined_DBZone(t *testing.T) {
	openTestDB(t)
	createGroup(t, zonedGroup, []model.ConnectionInfo{
		testConn("db-a", zonedGroup, common.ResourceTypeRDBMS, ""),
		testConn("db-a2", zonedGroup, common.ResourceTypeRDBMS, ""),
		testConn("db-nozone", zonedGroup, common.ResourceTypeRDBMS, ""),
	})
	zoneA := []spider.KeyValue{{Key: "ZoneId", Value: "ap-northeast-2a"}}
	saveZonedRDBMS(t, "db-a", zoneA)
	saveZonedRDBMS(t, "db-a2", zoneA)
	saveZonedRDBMS(t, "db-nozone", []spider.KeyValue{{Key: "RegionId", Value: "ap-northeast-2"}})

	tests := []struct {
		connID string
		want   string
	}{
		{"db-a", "ap-northeast-2a"},
		{"db-nozone", "ap-northeast-2c"},
	}
	for _, tc := range tests {
		rec := connRequest(t, http.MethodGet, zonedGroup.ID, tc.connID, GetRDBMSInfoRefined)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: %d %s", tc.connID, rec.Code, rec.Body.String())
		}
		sc := refinedRDBMSCloud(t, rec.Body.Bytes())
		if sc.Zone != tc.want || sc.Region != "ap-northeast-2" || sc.Csp != "alibaba" {
			t.Errorf("%s: sourceCloud = %+v, want zone %s in ap-northeast-2", tc.connID, sc, tc.want)
		}
	}

	rec := groupRequest(t, zonedGroup.ID, GetRDBMSInfoSourceGroupRefined)
	if sc := refinedRDBMSCloud(t, rec.Body.Bytes()); sc.Zone != "ap-northeast-2c" {
		t.Errorf("group with a database of no zone: zone = %q, want the group's ap-northeast-2c", sc.Zone)
	}

	saveZonedRDBMS(t, "db-nozone", zoneA)
	rec = groupRequest(t, zonedGroup.ID, GetRDBMSInfoSourceGroupRefined)
	if sc := refinedRDBMSCloud(t, rec.Body.Bytes()); sc.Zone != "ap-northeast-2a" {
		t.Errorf("group of databases in one zone: zone = %q, want ap-northeast-2a", sc.Zone)
	}

	saveZonedRDBMS(t, "db-a2", []spider.KeyValue{{Key: "ZoneId", Value: "ap-northeast-2b"}})
	rec = groupRequest(t, zonedGroup.ID, GetRDBMSInfoSourceGroupRefined)
	if sc := refinedRDBMSCloud(t, rec.Body.Bytes()); sc.Zone != "ap-northeast-2c" {
		t.Errorf("group of databases in two zones: zone = %q, want the group's ap-northeast-2c", sc.Zone)
	}
}

func TestCommonZone(t *testing.T) {
	tests := []struct {
		zones []string
		want  string
	}{
		{nil, ""},
		{[]string{"a"}, "a"},
		{[]string{"a", "a"}, "a"},
		{[]string{"a", "b"}, ""},
		{[]string{"a", ""}, ""},
		{[]string{""}, ""},
	}
	for _, tc := range tests {
		if got := commonZone(tc.zones); got != tc.want {
			t.Errorf("commonZone(%q) = %q, want %q", tc.zones, got, tc.want)
		}
	}
}
