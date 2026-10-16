package software

import (
	"reflect"
	"testing"

	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/software"
)

// TestDedupeServiceWorkers pins the legacy binary collapsing behavior that
// dedupeByService now provides generically.
func TestDedupeServiceWorkers(t *testing.T) {
	tests := []struct {
		name      string
		results   []software.Binary
		ppidByPID map[int32]int32
		wantPIDs  []int32
	}{
		{
			name:      "empty",
			results:   nil,
			ppidByPID: map[int32]int32{},
			wantPIDs:  nil,
		},
		{
			name: "master kept, workers dropped, order preserved",
			results: []software.Binary{
				{PID: 20, ExecutablePath: "/opt/nginx/sbin/nginx"},
				{PID: 10, ExecutablePath: "/usr/local/apache2/bin/httpd"},
				{PID: 11, ExecutablePath: "/usr/local/apache2/bin/httpd"},
				{PID: 21, ExecutablePath: "/opt/nginx/sbin/nginx"},
			},
			ppidByPID: map[int32]int32{10: 1, 11: 10, 20: 1, 21: 20},
			wantPIDs:  []int32{20, 10},
		},
		{
			name: "no in-group parent falls back to lowest pid",
			results: []software.Binary{
				{PID: 32, ExecutablePath: "/opt/app/app"},
				{PID: 31, ExecutablePath: "/opt/app/app"},
			},
			// A parent cycle inside the group: every member has an in-group parent.
			ppidByPID: map[int32]int32{31: 32, 32: 31},
			wantPIDs:  []int32{31},
		},
		{
			name: "separately launched instances both kept",
			results: []software.Binary{
				{PID: 40, ExecutablePath: "/opt/app/app"},
				{PID: 41, ExecutablePath: "/opt/app/app"},
			},
			ppidByPID: map[int32]int32{40: 1, 41: 1},
			wantPIDs:  []int32{40, 41},
		},
		{
			name: "missing executable path kept distinct",
			results: []software.Binary{
				{PID: 50},
				{PID: 51},
			},
			ppidByPID: map[int32]int32{50: 1, 51: 50},
			wantPIDs:  []int32{50, 51},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dedupeServiceWorkers(tt.results, tt.ppidByPID)
			var pids []int32
			for _, b := range got {
				pids = append(pids, b.PID)
			}
			if !reflect.DeepEqual(pids, tt.wantPIDs) {
				t.Errorf("dedupeServiceWorkers() PIDs = %v, want %v", pids, tt.wantPIDs)
			}
		})
	}
}
