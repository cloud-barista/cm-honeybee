//go:build linux

package software

import "testing"

func TestContainerIDFromCgroup(t *testing.T) {
	const id = "84a6f37067792b065439bd86a852ad3cb56dfe3c4a941045c8dbf68325ace310"

	tests := []struct {
		name   string
		cgroup string
		want   string
	}{
		{"docker, systemd driver", "0::/system.slice/docker-" + id + ".scope\n", id},
		{"docker, cgroupfs driver (v1)", "12:pids:/docker/" + id + "\n11:memory:/docker/" + id + "\n", id},
		{"podman, systemd driver", "0::/machine.slice/libpod-" + id + ".scope/container\n", id},
		{"podman conmon is not the container", "0::/machine.slice/libpod-conmon-" + id + ".scope\n", ""},
		{"host service", "0::/system.slice/mysql.service\n", ""},
		{"docker daemon itself", "0::/system.slice/docker.service\n", ""},
		{"short id", "0::/system.slice/docker-84a6f3706779.scope\n", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := containerIDFromCgroup(tt.cgroup); got != tt.want {
				t.Errorf("containerIDFromCgroup(%q) = %q, want %q", tt.cgroup, got, tt.want)
			}
		})
	}
}
