package controller

import (
	"encoding/json"
	"errors"
	"testing"

	rdbmsmodel "github.com/cloud-barista/cm-beetle/imdl/rdbms-model"
	"github.com/cloud-barista/cm-honeybee/server/dao"
	"github.com/cloud-barista/cm-honeybee/server/lib/spider"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
)

func TestEndpointPort(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		want     int
	}{
		{"host and port", "db.example.com:3306", 3306},
		{"ip and port", "10.0.0.5:5432", 5432},
		{"ipv6 and port", "[::1]:3307", 3307},
		{"host only", "db.example.com", 0},
		{"not available", "NA", 0},
		{"empty", "", 0},
		{"non-numeric port", "db.example.com:mysql", 0},
		{"port out of range", "db.example.com:70000", 0},
		{"zero port", "db.example.com:0", 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := endpointPort(tc.endpoint); got != tc.want {
				t.Errorf("endpointPort(%q) = %d, want %d", tc.endpoint, got, tc.want)
			}
		})
	}
}

func TestRDBMSRole(t *testing.T) {
	tests := []struct {
		name string
		kvs  []spider.KeyValue
		want string
	}{
		{"no evidence", nil, ""},
		{"aws replica", []spider.KeyValue{{Key: "ReadReplicaSourceDBInstanceIdentifier", Value: "source-db"}}, rdbmsRoleReplica},
		// A primary flattens the nil pointer to an empty value.
		{"aws primary", []spider.KeyValue{{Key: "ReadReplicaSourceDBInstanceIdentifier", Value: ""}}, ""},
		{"gcp replica by type", []spider.KeyValue{{Key: "InstanceType", Value: "READ_REPLICA_INSTANCE"}}, rdbmsRoleReplica},
		{"gcp replica by master", []spider.KeyValue{{Key: "MasterInstanceName", Value: "proj:primary"}}, rdbmsRoleReplica},
		{"gcp primary", []spider.KeyValue{{Key: "InstanceType", Value: "CLOUD_SQL_INSTANCE"}}, ""},
		// DBInstanceType is not evidence: its meaning differs per CSP.
		{"unrelated key", []spider.KeyValue{{Key: "DBInstanceType", Value: "ReadReplica"}}, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := rdbmsRole(tc.kvs); got != tc.want {
				t.Errorf("rdbmsRole = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestNormalizeRDBMSEngine(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"postgres", "postgresql"},
		{"PostgreSQL", "postgresql"},
		{"MySQL", "mysql"},
		{" mariadb ", "mariadb"},
		{"", ""},
	}

	for _, tc := range tests {
		if got := normalizeRDBMSEngine(tc.in); got != tc.want {
			t.Errorf("normalizeRDBMSEngine(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestRefineManagedRDBMS(t *testing.T) {
	base := spider.RDBMSInfo{
		IId:             spider.IID{NameId: "billing-db", SystemId: "db-abc123"},
		DBEngine:        "postgres",
		DBEngineVersion: "15.4",
		DBSpec:          "db.t3.medium",
		StorageType:     "gp3",
		StorageSize:     "100",
		Endpoint:        "billing-db.example.com:5432",
	}

	t.Run("full spec", func(t *testing.T) {
		got := refineManagedRDBMS(&model.RDBMSInfo{
			RDBMS: &base,
			DBSpec: &spider.DBSpecInfo{
				VCpu:       spider.VCpuInfo{Count: "2", ClockGHz: "2.5"},
				MemSizeMiB: "4096",
			},
		})

		if got.DisplayName != "billing-db" {
			t.Errorf("DisplayName = %q", got.DisplayName)
		}
		if got.DBEngine.Engine != "postgresql" || got.DBEngine.EngineVersion != "15.4" {
			t.Errorf("DBEngine = %+v", got.DBEngine)
		}
		if got.DBEngine.Port != 5432 || got.DBEngine.Role != "" {
			t.Errorf("Port/Role = %d/%q", got.DBEngine.Port, got.DBEngine.Role)
		}
		cpu := got.DBNode.CPU
		if cpu.Cpus != 1 || cpu.Cores != 2 || cpu.Threads != 2 || cpu.MaxSpeed != 2.5 {
			t.Errorf("CPU = %+v", cpu)
		}
		if got.DBNode.Memory.TotalSize != 4 {
			t.Errorf("Memory.TotalSize = %d, want 4", got.DBNode.Memory.TotalSize)
		}
		if got.DBNode.RootDisk.TotalSize != 100 || got.DBNode.RootDisk.Type != "gp3" {
			t.Errorf("RootDisk = %+v", got.DBNode.RootDisk)
		}
		if len(got.DBNode.DataDisks) != 0 {
			t.Errorf("DataDisks = %+v, want none", got.DBNode.DataDisks)
		}
	})

	// "-1" is cb-spider's "not exposed"; it must stay zero so cm-beetle fills it.
	t.Run("unknown spec", func(t *testing.T) {
		got := refineManagedRDBMS(&model.RDBMSInfo{
			RDBMS: &base,
			DBSpec: &spider.DBSpecInfo{
				VCpu:       spider.VCpuInfo{Count: "-1", ClockGHz: "-1"},
				MemSizeMiB: "-1",
			},
		})

		if got.DBNode.CPU != (rdbmsmodel.CpuProperty{}) {
			t.Errorf("CPU = %+v, want zero", got.DBNode.CPU)
		}
		if got.DBNode.Memory.TotalSize != 0 {
			t.Errorf("Memory.TotalSize = %d, want 0", got.DBNode.Memory.TotalSize)
		}
	})

	t.Run("no spec and unparseable storage", func(t *testing.T) {
		in := base
		in.StorageSize = "NA"
		in.Endpoint = "NA"

		got := refineManagedRDBMS(&model.RDBMSInfo{RDBMS: &in})

		if got.DBNode.CPU.Threads != 0 || got.DBNode.Memory.TotalSize != 0 {
			t.Errorf("DBNode = %+v, want empty compute", got.DBNode)
		}
		if got.DBNode.RootDisk.TotalSize != 0 || got.DBNode.RootDisk.Type != "" {
			t.Errorf("RootDisk = %+v, want empty", got.DBNode.RootDisk)
		}
		if got.DBEngine.Port != 0 {
			t.Errorf("Port = %d, want 0", got.DBEngine.Port)
		}
	})
}

// The stored payload must round-trip through the table, and a connection that
// was never collected must be reported as such so source group queries skip it.
func TestManagedRDBMSInfoDAO(t *testing.T) {
	openTestDB(t)

	conn := &model.ConnectionInfo{ID: "conn-rdbms"}

	_, err := doGetManagedRDBMSInfo(conn)
	if !errors.Is(err, errRDBMSNotCollected) {
		t.Fatalf("missing record: err = %v, want errRDBMSNotCollected", err)
	}

	payload := model.RDBMSInfo{
		RDBMS:  &spider.RDBMSInfo{IId: spider.IID{SystemId: "db-1"}, DBEngine: "mysql", StorageSize: "20"},
		DBSpec: &spider.DBSpecInfo{Name: "db.t3.micro", MemSizeMiB: "1024"},
	}
	if err := upsertSavedRDBMS(conn.ID, payload); err != nil {
		t.Fatal(err)
	}
	// A second import replaces the record instead of failing on the primary key.
	payload.RDBMS.StorageSize = "40"
	if err := upsertSavedRDBMS(conn.ID, payload); err != nil {
		t.Fatal(err)
	}

	got, err := doGetManagedRDBMSInfo(conn)
	if err != nil {
		t.Fatal(err)
	}
	if got.RDBMS.IId.SystemId != "db-1" || got.RDBMS.StorageSize != "40" {
		t.Errorf("RDBMS = %+v", got.RDBMS)
	}
	if got.DBSpec == nil || got.DBSpec.MemSizeMiB != "1024" {
		t.Errorf("DBSpec = %+v", got.DBSpec)
	}

	saved, err := dao.SavedRDBMSInfoGet(conn.ID)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal([]byte(saved.RDBMSData), &raw); err != nil {
		t.Fatal(err)
	}
	if rdbms, _ := raw["rdbms"].(map[string]any); rdbms == nil || rdbms["MasterUserPassword"] != nil {
		t.Errorf("stored rdbms = %v, want no MasterUserPassword", raw["rdbms"])
	}

	if err := dao.SavedRDBMSInfoDelete(saved); err != nil {
		t.Fatal(err)
	}
	if _, err := dao.SavedRDBMSInfoGet(conn.ID); err == nil {
		t.Error("record still present after delete")
	}
}
