package software

import (
	"reflect"
	"testing"

	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/software"
	"github.com/shirou/gopsutil/v3/net"
)

func conn(status string, port uint32) net.ConnectionStat {
	return net.ConnectionStat{Status: status, Laddr: net.Addr{IP: "0.0.0.0", Port: port}}
}

func TestListenPorts(t *testing.T) {
	tests := []struct {
		name  string
		conns []net.ConnectionStat
		want  []int
	}{
		{name: "no connections", conns: nil, want: nil},
		{
			name:  "only established and udp",
			conns: []net.ConnectionStat{conn("ESTABLISHED", 3306), conn("NONE", 53), conn("", 123)},
			want:  nil,
		},
		{
			name:  "sorted",
			conns: []net.ConnectionStat{conn("LISTEN", 5432), conn("LISTEN", 22), conn("LISTEN", 3306)},
			want:  []int{22, 3306, 5432},
		},
		{
			name:  "ipv4 and ipv6 on the same port collapse",
			conns: []net.ConnectionStat{conn("LISTEN", 3306), conn("LISTEN", 3306), conn("LISTEN", 33060)},
			want:  []int{3306, 33060},
		},
		{
			name:  "port zero ignored",
			conns: []net.ConnectionStat{conn("LISTEN", 0), conn("LISTEN", 80)},
			want:  []int{80},
		},
		{
			name:  "mixed states keep listen only",
			conns: []net.ConnectionStat{conn("TIME_WAIT", 80), conn("LISTEN", 443), conn("ESTABLISHED", 443)},
			want:  []int{443},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := listenPorts(tt.conns); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("listenPorts() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDedupeListenerWorkers(t *testing.T) {
	tests := []struct {
		name      string
		listeners []software.Listener
		ppidByPID map[int32]int32
		wantPIDs  []int32
	}{
		{
			name: "workers collapse into master",
			listeners: []software.Listener{
				{PID: 100, ExecutablePath: "/usr/sbin/httpd", Ports: []int{80}},
				{PID: 101, ExecutablePath: "/usr/sbin/httpd", Ports: []int{80}},
				{PID: 102, ExecutablePath: "/usr/sbin/httpd", Ports: []int{80}},
				{PID: 200, ExecutablePath: "/usr/sbin/mysqld", Ports: []int{3306}},
			},
			ppidByPID: map[int32]int32{100: 1, 101: 100, 102: 100, 200: 1},
			wantPIDs:  []int32{100, 200},
		},
		{
			name: "same exe with unrelated parents kept apart",
			listeners: []software.Listener{
				{PID: 300, ExecutablePath: "/opt/app/bin/server", Ports: []int{8080}},
				{PID: 301, ExecutablePath: "/opt/app/bin/server", Ports: []int{8081}},
			},
			ppidByPID: map[int32]int32{300: 1, 301: 1},
			wantPIDs:  []int32{300, 301},
		},
		{
			name: "exe-less processes never merged",
			listeners: []software.Listener{
				{PID: 400, Ports: []int{9000}},
				{PID: 401, Ports: []int{9000}},
			},
			ppidByPID: map[int32]int32{400: 1, 401: 400},
			wantPIDs:  []int32{400, 401},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dedupeListenerWorkers(tt.listeners, tt.ppidByPID)
			var pids []int32
			for _, l := range got {
				pids = append(pids, l.PID)
			}
			if !reflect.DeepEqual(pids, tt.wantPIDs) {
				t.Errorf("dedupeListenerWorkers() PIDs = %v, want %v", pids, tt.wantPIDs)
			}
		})
	}
}
