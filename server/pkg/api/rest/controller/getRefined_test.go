package controller

import (
	"encoding/json"
	"testing"

	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/infra"
)

func TestMibToGiB(t *testing.T) {
	cases := []struct {
		mib      uint64
		rounded  uint64
		capacity uint64
	}{
		{mib: 0, rounded: 0, capacity: 0},
		{mib: 249, rounded: 0, capacity: 1},
		{mib: 362, rounded: 0, capacity: 1},
		{mib: 511, rounded: 0, capacity: 1},
		{mib: 512, rounded: 1, capacity: 1},
		{mib: 1020, rounded: 1, capacity: 1}, // 1 GiB VM as the OS sees it
		{mib: 1024, rounded: 1, capacity: 1},
		{mib: 1536, rounded: 2, capacity: 2},
		{mib: 2000, rounded: 2, capacity: 2},
		{mib: 8192, rounded: 8, capacity: 8},
	}

	for _, c := range cases {
		if got := mibToGiB(c.mib); got != c.rounded {
			t.Errorf("mibToGiB(%d) = %d, want %d", c.mib, got, c.rounded)
		}

		if got := mibToGiBCapacity(c.mib); got != c.capacity {
			t.Errorf("mibToGiBCapacity(%d) = %d, want %d", c.mib, got, c.capacity)
		}
	}
}

// TestDoGetRefinedInfraInfoMemory pins the reading that motivated the rounding:
// a 1 GiB KT Cloud instance whose collected memory truncated to a totalSize of
// 0, which reads downstream as a node with no memory at all.
func TestDoGetRefinedInfraInfoMemory(t *testing.T) {
	infraInfo := &infra.Infra{}
	infraInfo.Compute.ComputeResource.Memory = infra.Memory{
		Type:      "RAM",
		Speed:     0,
		Size:      1020,
		Used:      249,
		Available: 362,
	}

	refined, err := doGetRefinedInfraInfo(infraInfo)
	if err != nil {
		t.Fatalf("doGetRefinedInfraInfo: %v", err)
	}

	if refined.Memory.TotalSize != 1 {
		t.Errorf("Memory.TotalSize = %d, want 1", refined.Memory.TotalSize)
	}

	got, err := json.Marshal(refined.Memory)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	want := `{"type":"RAM","totalSize":1}`
	if string(got) != want {
		t.Errorf("refined memory = %s, want %s", got, want)
	}
}
