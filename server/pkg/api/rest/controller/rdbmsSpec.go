package controller

import (
	"strconv"
	"strings"

	"github.com/cloud-barista/cm-honeybee/server/lib/spider"
)

// rdbmsSizeKeys are the pairs of keys of a database's flattened CSP record
// that give its vCPU count and its memory in MiB, in the order they are tried:
//   - Alibaba: DBInstanceCPU, DBInstanceMemory. Its GetDBSpec was seen to take
//     over a minute and run into the driver call timeout, which left the CPU
//     and memory empty.
//
// A pair is used only when both of its values are there, so that the CPU and
// the memory always come from the same CSP's fields.
var rdbmsSizeKeys = []struct {
	cpu       string
	memoryMiB string
}{
	{"DBInstanceCPU", "DBInstanceMemory"},
}

// rdbmsSpecFromRecord builds the spec size of a managed database out of its
// KeyValueList, or gives nil when the CSP's record does not carry it, in which
// case the spec has to be asked for. Keys are matched ignoring case, as
// rdbmsZone does. A value that is not a positive whole number counts as absent:
// drivers report an unknown number as "0", "-1" or "NA".
func rdbmsSpecFromRecord(db *spider.RDBMSInfo) *spider.DBSpecInfo {
	if db == nil {
		return nil
	}

	for _, keys := range rdbmsSizeKeys {
		cpu, okCPU := rdbmsPositiveValue(db.KeyValueList, keys.cpu)
		mem, okMem := rdbmsPositiveValue(db.KeyValueList, keys.memoryMiB)
		if !okCPU || !okMem {
			continue
		}
		return &spider.DBSpecInfo{
			Name:       db.DBSpec,
			DBEngine:   db.DBEngine,
			VCpu:       spider.VCpuInfo{Count: strconv.FormatUint(cpu, 10)},
			MemSizeMiB: strconv.FormatUint(mem, 10),
		}
	}

	return nil
}

// rdbmsPositiveValue reads the first value under key that is a positive whole
// number.
func rdbmsPositiveValue(kvs []spider.KeyValue, key string) (uint64, bool) {
	for _, kv := range kvs {
		if !strings.EqualFold(strings.TrimSpace(kv.Key), key) {
			continue
		}
		if v, err := strconv.ParseUint(strings.TrimSpace(kv.Value), 10, 64); err == nil && v > 0 {
			return v, true
		}
	}

	return 0, false
}
