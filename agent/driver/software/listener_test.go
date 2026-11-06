package software

import (
	"reflect"
	"syscall"
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

func TestParsePackageVersions(t *testing.T) {
	tests := []struct {
		name string
		out  string
		want map[string]string
	}{
		{
			name: "dpkg-query",
			out:  "mysql-server-core-8.0\t8.0.46-0ubuntu0.22.04.4\npostgresql-14\t14.24-0ubuntu0.22.04.1\n",
			want: map[string]string{
				"mysql-server-core-8.0": "8.0.46-0ubuntu0.22.04.4",
				"postgresql-14":         "14.24-0ubuntu0.22.04.1",
			},
		},
		{
			name: "rpm with a missing package",
			out:  "mysql-server\t8.0.36\npackage nginx is not installed\n",
			want: map[string]string{"mysql-server": "8.0.36"},
		},
		{
			name: "multiarch package listed twice keeps the first",
			out:  "libfoo\t1.0\nlibfoo\t1.0\n",
			want: map[string]string{"libfoo": "1.0"},
		},
		{
			name: "empty",
			out:  "",
			want: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parsePackageVersions(tt.out); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parsePackageVersions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestListenSockets(t *testing.T) {
	sock := func(typ uint32, family uint32, status, ip string, port uint32, rport uint32) net.ConnectionStat {
		return net.ConnectionStat{
			Type: typ, Family: family, Status: status,
			Laddr: net.Addr{IP: ip, Port: port}, Raddr: net.Addr{Port: rport},
		}
	}
	const (
		tcp = uint32(syscall.SOCK_STREAM)
		udp = uint32(syscall.SOCK_DGRAM)
		v4  = uint32(syscall.AF_INET)
		v6  = uint32(syscall.AF_INET6)
	)

	conns := []net.ConnectionStat{
		sock(tcp, v4, "LISTEN", "0.0.0.0", 22, 0),
		sock(tcp, v6, "LISTEN", "::", 22, 0),
		sock(tcp, v4, "ESTABLISHED", "192.0.2.5", 22, 51000), // a session, not a listener
		sock(tcp, v4, "LISTEN", "127.0.0.1", 5432, 0),
		sock(tcp, v4, "LISTEN", "127.0.0.1", 5432, 0), // duplicate
		sock(udp, v4, "NONE", "0.0.0.0", 53, 0),
		sock(udp, v4, "NONE", "192.0.2.5", 40000, 53), // connected client socket
		sock(udp, v6, "NONE", "::1", 323, 0),
		sock(tcp, 1, "LISTEN", "/run/app.sock", 0, 0), // unix socket
		sock(udp, v4, "NONE", "0.0.0.0", 0, 0),        // unbound
	}

	want := []software.ListenSocket{
		{Protocol: "tcp", BindAddress: "0.0.0.0", Port: 22},
		{Protocol: "tcp", BindAddress: "::", Port: 22},
		{Protocol: "tcp", BindAddress: "127.0.0.1", Port: 5432},
		{Protocol: "udp", BindAddress: "0.0.0.0", Port: 53},
		{Protocol: "udp", BindAddress: "::1", Port: 323},
	}

	if got := listenSockets(conns); !reflect.DeepEqual(got, want) {
		t.Errorf("listenSockets() = %+v, want %+v", got, want)
	}
	if got := listenSockets(nil); got != nil {
		t.Errorf("listenSockets(nil) = %+v, want nil", got)
	}
}
