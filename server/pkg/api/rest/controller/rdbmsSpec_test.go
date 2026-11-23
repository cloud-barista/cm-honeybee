package controller

import (
	"testing"

	"github.com/cloud-barista/cm-honeybee/server/lib/spider"
)

// alibabaRDSRecord is part of the KeyValueList the alibaba driver was seen to
// report for an RDS instance. The TempUpgradeRecovery values are not its size.
func alibabaRDSRecord() []spider.KeyValue {
	return []spider.KeyValue{
		{Key: "ZoneId", Value: "ap-northeast-2a"},
		{Key: "DBInstanceClass", Value: "mysql.n2e.small.1"},
		{Key: "DBInstanceCPU", Value: "1"},
		{Key: "DBInstanceMemory", Value: "2048"},
		{Key: "TempUpgradeRecoveryCpu", Value: "0"},
		{Key: "TempUpgradeRecoveryMemory", Value: "0"},
	}
}

func TestRDBMSSpecFromRecord(t *testing.T) {
	tests := []struct {
		name    string
		kvs     []spider.KeyValue
		wantCPU string
		wantMiB string
	}{
		{"alibaba", alibabaRDSRecord(), "1", "2048"},
		{"key case ignored", []spider.KeyValue{{Key: "dbinstancecpu", Value: " 4 "}, {Key: "DBINSTANCEMEMORY", Value: "8192"}}, "4", "8192"},
		{"zero value skipped for a later one", []spider.KeyValue{{Key: "DBInstanceCPU", Value: "0"}, {Key: "DBInstanceCPU", Value: "2"}, {Key: "DBInstanceMemory", Value: "4096"}}, "2", "4096"},
		{"memory missing", []spider.KeyValue{{Key: "DBInstanceCPU", Value: "1"}}, "", ""},
		{"cpu missing", []spider.KeyValue{{Key: "DBInstanceMemory", Value: "2048"}}, "", ""},
		{"cpu zero", []spider.KeyValue{{Key: "DBInstanceCPU", Value: "0"}, {Key: "DBInstanceMemory", Value: "2048"}}, "", ""},
		{"memory not a number", []spider.KeyValue{{Key: "DBInstanceCPU", Value: "1"}, {Key: "DBInstanceMemory", Value: "NA"}}, "", ""},
		{"cpu negative", []spider.KeyValue{{Key: "DBInstanceCPU", Value: "-1"}, {Key: "DBInstanceMemory", Value: "2048"}}, "", ""},
		{"tencent keys not read", []spider.KeyValue{{Key: "Cpu", Value: "1"}, {Key: "Memory", Value: "1000"}}, "", ""},
		{"no record", nil, "", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db := &spider.RDBMSInfo{DBEngine: "MySQL", DBSpec: "mysql.n2e.small.1", KeyValueList: tc.kvs}
			spec := rdbmsSpecFromRecord(db)
			if tc.wantCPU == "" {
				if spec != nil {
					t.Errorf("spec = %+v, want nil", spec)
				}
				return
			}
			if spec == nil {
				t.Fatal("spec = nil")
			}
			if spec.VCpu.Count != tc.wantCPU || spec.MemSizeMiB != tc.wantMiB ||
				spec.Name != "mysql.n2e.small.1" || spec.DBEngine != "MySQL" {
				t.Errorf("spec = %+v, want %s vCPU, %s MiB, named after the instance's spec", spec, tc.wantCPU, tc.wantMiB)
			}
		})
	}

	if spec := rdbmsSpecFromRecord(nil); spec != nil {
		t.Errorf("rdbmsSpecFromRecord(nil) = %+v, want nil", spec)
	}
}

// stubRDBMSSpecCalls stands in for the managed database calls of the CSP with a
// listing that holds db and a spec lookup that answers 2 vCPU and 4096 MiB. It
// returns the number of spec lookups made.
func stubRDBMSSpecCalls(t *testing.T, db spider.RDBMSInfo) *int {
	t.Helper()

	origList, origSpec := listCSPRDBMS, getCSPDBSpec
	t.Cleanup(func() { listCSPRDBMS, getCSPDBSpec = origList, origSpec })

	calls := new(int)
	listCSPRDBMS = func(_ spider.Conn) ([]spider.RDBMSInfo, error) {
		return []spider.RDBMSInfo{db}, nil
	}
	getCSPDBSpec = func(_ spider.Conn, _ string, name string) (*spider.DBSpecInfo, error) {
		*calls++
		return &spider.DBSpecInfo{Name: name, VCpu: spider.VCpuInfo{Count: "2"}, MemSizeMiB: "4096"}, nil
	}

	return calls
}

func TestCollectRDBMS_SizeFromRecord(t *testing.T) {
	conn := spider.Conn{Provider: "alibaba", Region: "ap-northeast-2", Zone: "ap-northeast-2a"}

	t.Run("record carries the size", func(t *testing.T) {
		db := spider.RDBMSInfo{IId: spider.IID{SystemId: "rm-1"}, DBEngine: "MySQL", DBSpec: "mysql.n2e.small.1",
			KeyValueList: alibabaRDSRecord()}
		calls := stubRDBMSSpecCalls(t, db)

		payload, err := collectRDBMS(conn, "rm-1")
		if err != nil {
			t.Fatal(err)
		}
		if *calls != 0 {
			t.Errorf("spec asked %d times, want none", *calls)
		}
		if payload.DBSpec == nil || payload.DBSpec.VCpu.Count != "1" || payload.DBSpec.MemSizeMiB != "2048" ||
			payload.DBSpec.Name != "mysql.n2e.small.1" {
			t.Fatalf("DBSpec = %+v, want 1 vCPU and 2048 MiB from the record", payload.DBSpec)
		}

		node := refineManagedRDBMS(&payload).DBNode
		if node.CPU.Cpus != 1 || node.CPU.Cores != 1 || node.CPU.Threads != 1 || node.Memory.TotalSize != 2 {
			t.Errorf("refined dbNode cpu %+v memory %d GiB, want 1 vCPU and 2 GiB", node.CPU, node.Memory.TotalSize)
		}
	})

	t.Run("record lacks the memory", func(t *testing.T) {
		db := spider.RDBMSInfo{IId: spider.IID{SystemId: "rm-2"}, DBEngine: "MySQL", DBSpec: "mysql.n2e.small.1",
			KeyValueList: []spider.KeyValue{{Key: "DBInstanceCPU", Value: "1"}, {Key: "DBInstanceMemory", Value: "0"}}}
		calls := stubRDBMSSpecCalls(t, db)

		payload, err := collectRDBMS(conn, "rm-2")
		if err != nil {
			t.Fatal(err)
		}
		if *calls != 1 {
			t.Errorf("spec asked %d times, want once", *calls)
		}
		if payload.DBSpec == nil || payload.DBSpec.VCpu.Count != "2" || payload.DBSpec.MemSizeMiB != "4096" {
			t.Errorf("DBSpec = %+v, want the looked-up spec", payload.DBSpec)
		}
	})
}
