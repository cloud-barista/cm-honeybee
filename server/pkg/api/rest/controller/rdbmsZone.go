package controller

import (
	"strings"

	"github.com/cloud-barista/cm-honeybee/server/lib/spider"
)

// rdbmsZoneKeys are the keys of a database's flattened CSP record that name the
// zone it runs in, in the order they are tried:
//   - AWS: AvailabilityZone
//   - GCP: GceZone
//   - Alibaba: ZoneId
//   - Tencent: Zone. Its ZoneId is a number, not a zone name, and is passed over.
//
// Azure reports only the region (Location) and IBM only the region (RegionID),
// so their databases have no zone of their own.
var rdbmsZoneKeys = []string{"AvailabilityZone", "GceZone", "ZoneId", "Zone"}

// rdbmsZone reads the zone a managed database runs in from its KeyValueList, or
// gives "" when the CSP does not report one. Keys are matched ignoring case, as
// drivers flatten field names as their SDKs spell them. A value made of digits
// only is a zone's numeric ID, which no driver call takes as a zone, so it is
// not a zone.
func rdbmsZone(db *spider.RDBMSInfo) string {
	if db == nil {
		return ""
	}

	for _, key := range rdbmsZoneKeys {
		for _, kv := range db.KeyValueList {
			if !strings.EqualFold(strings.TrimSpace(kv.Key), key) {
				continue
			}
			if v := strings.TrimSpace(kv.Value); v != "" && !digitsOnly(v) {
				return v
			}
		}
	}

	return ""
}

// digitsOnly reports whether s holds nothing but ASCII digits.
func digitsOnly(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
