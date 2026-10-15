package controller

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/infra"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/software"
	"github.com/cloud-barista/cm-honeybee/server/db"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"github.com/moby/moby/api/types/container"
)

func TestRDBMSVersion(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"8.0.39-0ubuntu0.22.04.1", "8.0.39"},
		{"1:10.6.18+maria~ubu2204", "10.6.18"},
		{"14.13-0ubuntu0.22.04.1", "14.13"},
		{"16.4", "16.4"},
		{"16-alpine", "16"},
		{"latest", ""},
		{"", ""},
	}

	for _, tc := range tests {
		if got := rdbmsVersion(tc.in); got != tc.want {
			t.Errorf("rdbmsVersion(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestRDBMSEngineOfPackage(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"mysql-server-core-8.0", rdbmsEngineMySQL},
		{"mysql-community-server-core", rdbmsEngineMySQL},
		{"mariadb-server-core-10.6", rdbmsEngineMariaDB},
		{"mariadb-server-core", rdbmsEngineMariaDB},
		{"postgresql-14", rdbmsEnginePostgreSQL},
		{"mysql-server", rdbmsEngineMySQL},
		{"mysql-community-server", rdbmsEngineMySQL},
		{"MariaDB-server", rdbmsEngineMariaDB},
		{"postgresql16-server", rdbmsEnginePostgreSQL},
		{"postgresql-server", rdbmsEnginePostgreSQL},
		// Clients, meta and helper packages install no server.
		{"mysql-client-core-8.0", ""},
		{"mariadb-client", ""},
		{"postgresql-client-14", ""},
		{"postgresql-common", ""},
		{"postgresql", ""},
		{"postgresql-contrib", ""},
	}

	for _, tc := range tests {
		if got := rdbmsEngineOfPackage(tc.name); got != tc.want {
			t.Errorf("rdbmsEngineOfPackage(%q) = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestRDBMSEngineOfProcess(t *testing.T) {
	tests := []struct {
		name       string
		exe        string
		versionRaw string
		syft       *software.SyftMatch
		want       string
	}{
		{"mysqld", "/opt/mysql/bin/mysqld", "mysqld  Ver 8.0.39 for Linux on x86_64 (MySQL Community Server - GPL)", nil, rdbmsEngineMySQL},
		{"old mariadb as mysqld", "/opt/mariadb/bin/mysqld", "mysqld  Ver 10.3.39-MariaDB for Linux on x86_64", nil, rdbmsEngineMariaDB},
		{"mariadbd", "/opt/mariadb/bin/mariadbd", "", nil, rdbmsEngineMariaDB},
		{"postgres", "/opt/pgsql/bin/postgres", "postgres (PostgreSQL) 16.4", nil, rdbmsEnginePostgreSQL},
		{"renamed but classified", "/opt/db/bin/dbserver", "", &software.SyftMatch{Name: "postgresql", Version: "16.4"}, rdbmsEnginePostgreSQL},
		{"not a database", "/usr/sbin/nginx", "nginx version: nginx/1.24.0", nil, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := rdbmsEngineOfProcess(tc.exe, tc.versionRaw, tc.syft); got != tc.want {
				t.Errorf("rdbmsEngineOfProcess() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRDBMSPort(t *testing.T) {
	tests := []struct {
		name   string
		engine string
		ports  []int
		want   int
	}{
		{"default among several", rdbmsEngineMySQL, []int{3306, 33060}, 3306},
		{"default not first", rdbmsEnginePostgreSQL, []int{5433, 5432}, 5432},
		{"no default gives lowest", rdbmsEngineMySQL, []int{33061, 3307}, 3307},
		{"none", rdbmsEngineMySQL, []int{}, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := rdbmsPort(tc.engine, tc.ports); got != tc.want {
				t.Errorf("rdbmsPort() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestFindInstalledRDBMS(t *testing.T) {
	t.Run("package-owned server, one entry per process", func(t *testing.T) {
		sw := &software.Software{
			DEB: []software.DEB{
				{Package: "postgresql-14", Status: "install ok installed", Version: "14.13-0ubuntu0.22.04.1"},
				{Package: "postgresql-client-14", Status: "install ok installed", Version: "14.13-0ubuntu0.22.04.1"},
			},
			Listeners: []software.Listener{
				{PID: 100, Name: "postgres", ExecutablePath: "/usr/lib/postgresql/14/bin/postgres", Ports: []int{5432}, Package: "postgresql-14"},
				{PID: 200, Name: "postgres", ExecutablePath: "/usr/lib/postgresql/14/bin/postgres", Ports: []int{5433}, Package: "postgresql-14"},
			},
		}

		got := findInstalledRDBMS(sw)
		if len(got) != 2 {
			t.Fatalf("got %d databases, want 2: %+v", len(got), got)
		}
		for i, pid := range []int32{100, 200} {
			if got[i].PID != pid || got[i].Engine != rdbmsEnginePostgreSQL || got[i].Version != "14.13" || got[i].Source != installedRDBMSSourceDEB {
				t.Errorf("got[%d] = %+v", i, got[i])
			}
		}
	})

	t.Run("server package left out of the list as a dependency", func(t *testing.T) {
		sw := &software.Software{
			DEB: []software.DEB{{Package: "postgresql", Status: "install ok installed", Version: "14+238"}},
			Listeners: []software.Listener{{
				PID: 100, Name: "postgres", ExecutablePath: "/usr/lib/postgresql/14/bin/postgres", Ports: []int{5432},
				Package: "postgresql-14", PackageVersion: "14.24-0ubuntu0.22.04.1",
			}},
		}

		got := findInstalledRDBMS(sw)
		if len(got) != 1 || got[0].Engine != rdbmsEnginePostgreSQL || got[0].Version != "14.24" ||
			got[0].Source != installedRDBMSSourceDEB || !reflect.DeepEqual(got[0].Ports, []int{5432}) {
			t.Errorf("got %+v", got)
		}
	})

	t.Run("listener package missing from the list and without a version is skipped", func(t *testing.T) {
		sw := &software.Software{
			Listeners: []software.Listener{{
				PID: 100, Name: "postgres", ExecutablePath: "/usr/lib/postgresql/14/bin/postgres", Ports: []int{5432},
				Package: "postgresql-14",
			}},
		}

		if got := findInstalledRDBMS(sw); len(got) != 0 {
			t.Errorf("got %+v, want none", got)
		}
	})

	t.Run("stopped package reported once without ports", func(t *testing.T) {
		sw := &software.Software{
			RPM: []software.RPM{{Name: "mysql-server", Version: "8.0.36"}},
		}

		got := findInstalledRDBMS(sw)
		want := []model.InstalledRDBMSInfo{{
			Engine: rdbmsEngineMySQL, Version: "8.0.36", Source: installedRDBMSSourceRPM, Ports: []int{}, Package: "mysql-server",
		}}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})

	t.Run("removed deb with kept config is not a database", func(t *testing.T) {
		sw := &software.Software{
			DEB: []software.DEB{{Package: "mysql-server-core-8.0", Status: "deinstall ok config-files", Version: "8.0.39"}},
		}

		if got := findInstalledRDBMS(sw); len(got) != 0 {
			t.Errorf("got %+v, want none", got)
		}
	})

	t.Run("unpackaged mariadb running as mysqld", func(t *testing.T) {
		sw := &software.Software{
			Legacy: []software.Binary{{
				PID: 300, Name: "mysqld", Version: "10.3.39", ExecutablePath: "/opt/mariadb/bin/mysqld",
				VersionRaw: "mysqld  Ver 10.3.39-MariaDB for Linux on x86_64",
			}},
			Listeners: []software.Listener{{PID: 300, Name: "mysqld", ExecutablePath: "/opt/mariadb/bin/mysqld", Ports: []int{3306}}},
		}

		got := findInstalledRDBMS(sw)
		if len(got) != 1 || got[0].Engine != rdbmsEngineMariaDB || got[0].Version != "10.3.39" ||
			got[0].Source != installedRDBMSSourceLegacy || !reflect.DeepEqual(got[0].Ports, []int{3306}) {
			t.Errorf("got %+v", got)
		}
	})

	t.Run("software collected before listeners", func(t *testing.T) {
		sw := &software.Software{
			DEB: []software.DEB{{Package: "mysql-server-core-8.0", Status: "install ok installed", Version: "8.0.39-0ubuntu0.22.04.1"}},
			Legacy: []software.Binary{{
				PID: 400, Name: "postgres", Version: "16.4", ExecutablePath: "/opt/pgsql/bin/postgres",
			}},
		}

		got := findInstalledRDBMS(sw)
		if len(got) != 2 {
			t.Fatalf("got %d databases, want 2: %+v", len(got), got)
		}
		if got[0].Engine != rdbmsEnginePostgreSQL || got[0].PID != 400 || len(got[0].Ports) != 0 {
			t.Errorf("legacy = %+v", got[0])
		}
		if got[1].Engine != rdbmsEngineMySQL || got[1].Package != "mysql-server-core-8.0" || len(got[1].Ports) != 0 {
			t.Errorf("package = %+v", got[1])
		}
	})

	t.Run("container by image", func(t *testing.T) {
		sw := &software.Software{
			Docker: []software.Container{
				{ContainerSummary: container.Summary{
					Names: []string{"/billing-db"},
					Image: "docker.io/bitnami/mysql:8.0.39",
					Ports: []container.PortSummary{
						{PrivatePort: 33060, Type: "tcp"},
						{PrivatePort: 3306, PublicPort: 13306, Type: "tcp"},
						{PrivatePort: 3306, PublicPort: 13306, Type: "tcp"},
					},
				}},
				{ContainerSummary: container.Summary{Names: []string{"/web"}, Image: "nginx:1.27"}},
			},
		}

		got := findInstalledRDBMS(sw)
		if len(got) != 1 {
			t.Fatalf("got %d databases, want 1: %+v", len(got), got)
		}
		if got[0].Engine != rdbmsEngineMySQL || got[0].Version != "8.0.39" || got[0].Container != "billing-db" ||
			got[0].Source != installedRDBMSSourceDocker || !reflect.DeepEqual(got[0].Ports, []int{3306, 33060}) {
			t.Errorf("got %+v", got[0])
		}
	})

	t.Run("no database gives an empty list", func(t *testing.T) {
		sw := &software.Software{
			Listeners: []software.Listener{{PID: 1, Name: "sshd", ExecutablePath: "/usr/sbin/sshd", Ports: []int{22}, Package: "openssh-server"}},
			DEB:       []software.DEB{{Package: "openssh-server", Status: "install ok installed", Version: "1:8.9p1"}},
		}

		got := findInstalledRDBMS(sw)
		if got == nil || len(got) != 0 {
			t.Errorf("got %#v, want an empty non-nil list", got)
		}
	})
}

func saveJSON(t *testing.T, rec interface{}) {
	t.Helper()
	if err := db.DB.Create(rec).Error; err != nil {
		t.Fatal(err)
	}
}

func mustMarshal(t *testing.T, v interface{}) string {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// The refined result joins the databases found in the software with the host
// described by the infra; a VM with no infra still reports its databases.
func TestRefinedInstalledRDBMS(t *testing.T) {
	openTestDB(t)

	sw := software.Software{
		DEB: []software.DEB{{Package: "mysql-server-core-8.0", Status: "install ok installed", Version: "8.0.39-0ubuntu0.22.04.1"}},
		Listeners: []software.Listener{{
			PID: 10, Name: "mysqld", ExecutablePath: "/usr/sbin/mysqld", Ports: []int{3306, 33060}, Package: "mysql-server-core-8.0",
		}},
	}

	var inf infra.Infra
	inf.Compute.OS.Node.Hostname = "db-01"
	inf.Compute.ComputeResource.CPU.Cpus = 1
	inf.Compute.ComputeResource.CPU.Cores = 4
	inf.Compute.ComputeResource.CPU.Threads = 8
	inf.Compute.ComputeResource.Memory.Size = 16384 // MiB

	saveJSON(t, &model.SavedSoftwareInfo{ConnectionID: "with-infra", SoftwareData: mustMarshal(t, sw)})
	saveJSON(t, &model.SavedInfraInfo{ConnectionID: "with-infra", InfraData: mustMarshal(t, inf)})
	saveJSON(t, &model.SavedSoftwareInfo{ConnectionID: "no-infra", SoftwareData: mustMarshal(t, sw)})
	saveJSON(t, &model.SavedSoftwareInfo{ConnectionID: "no-db", SoftwareData: mustMarshal(t, software.Software{})})

	got, err := doGetRefinedInstalledRDBMS("with-infra")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d instances, want 1", len(got))
	}
	if e := got[0].DBEngine; e.Engine != rdbmsEngineMySQL || e.EngineVersion != "8.0.39" || e.Port != 3306 || e.Role != "" {
		t.Errorf("DBEngine = %+v", e)
	}
	if n := got[0].DBNode; n.Hostname != "db-01" || n.CPU.Threads != 8 || n.Memory.TotalSize != 16 {
		t.Errorf("DBNode = %+v", n)
	}
	if got[0].DisplayName != "mysql-server-core-8.0" {
		t.Errorf("DisplayName = %q", got[0].DisplayName)
	}

	got, err = doGetRefinedInstalledRDBMS("no-infra")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].DBNode.CPU.Threads != 0 || got[0].DBEngine.Engine != rdbmsEngineMySQL {
		t.Errorf("no infra: got %+v", got)
	}

	got, err = doGetRefinedInstalledRDBMS("no-db")
	if err != nil || got == nil || len(got) != 0 {
		t.Errorf("no database: got %#v, err %v; want an empty non-nil list", got, err)
	}

	if _, err := doGetRefinedInstalledRDBMS("never-imported"); !errors.Is(err, errRDBMSNotCollected) {
		t.Errorf("never imported: err = %v, want errRDBMSNotCollected", err)
	}

	raw, err := doGetInstalledRDBMSInfo("with-infra")
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) != 1 || raw[0].Installed == nil || raw[0].RDBMS != nil || raw[0].Installed.PID != 10 {
		t.Errorf("raw = %+v", raw)
	}
}
