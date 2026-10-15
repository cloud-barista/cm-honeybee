package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"

	rdbmsmodel "github.com/cloud-barista/cm-beetle/imdl/rdbms-model"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/infra"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/software"
	"github.com/cloud-barista/cm-honeybee/server/dao"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"github.com/jollaman999/utils/logger"
)

// The engine names cm-beetle recommends for.
const (
	rdbmsEngineMySQL      = "mysql"
	rdbmsEngineMariaDB    = "mariadb"
	rdbmsEnginePostgreSQL = "postgresql"
)

// The software list an installed database was found in.
const (
	installedRDBMSSourceDEB    = "deb"
	installedRDBMSSourceRPM    = "rpm"
	installedRDBMSSourceLegacy = "legacy"
	installedRDBMSSourceDocker = "docker"
	installedRDBMSSourcePodman = "podman"
)

// rdbmsDefaultPorts are each engine's default port, the same values cm-beetle
// fills an empty port with. A database listening on several ports is reported
// by its default one when it is among them.
var rdbmsDefaultPorts = map[string]int{
	rdbmsEngineMySQL:      3306,
	rdbmsEngineMariaDB:    3306,
	rdbmsEnginePostgreSQL: 5432,
}

// rdbmsServerPackages match the packages that install a database server's
// binary. They name the server core rather than the meta packages around it
// (mysql-server, postgresql), which carry their own version and may be absent,
// and they leave out the clients, which install no server.
var rdbmsServerPackages = []struct {
	pattern *regexp.Regexp
	engine  string
}{
	// DEB
	{regexp.MustCompile(`^mysql-server-core-\d+\.\d+$`), rdbmsEngineMySQL},
	{regexp.MustCompile(`^mysql-community-server-core$`), rdbmsEngineMySQL},
	{regexp.MustCompile(`^mariadb-server-core(-\d+\.\d+)?$`), rdbmsEngineMariaDB},
	{regexp.MustCompile(`^postgresql-\d+$`), rdbmsEnginePostgreSQL},
	// RPM
	{regexp.MustCompile(`^mysql(-community)?-server$`), rdbmsEngineMySQL},
	{regexp.MustCompile(`^(?i:mariadb)-server$`), rdbmsEngineMariaDB},
	{regexp.MustCompile(`^postgresql\d*-server$`), rdbmsEnginePostgreSQL},
}

// rdbmsServerExecutables are the file names of the database server binaries.
// MariaDB before 10.5 also ships its server as "mysqld"; the version output
// tells the two apart (see rdbmsEngineOfProcess).
var rdbmsServerExecutables = map[string]string{
	"mysqld":     rdbmsEngineMySQL,
	"mariadbd":   rdbmsEngineMariaDB,
	"postgres":   rdbmsEnginePostgreSQL,
	"postmaster": rdbmsEnginePostgreSQL,
}

// rdbmsSyftNames are the package names syft's binary classifier gives the
// database products.
var rdbmsSyftNames = map[string]string{
	"mysql":      rdbmsEngineMySQL,
	"mariadb":    rdbmsEngineMariaDB,
	"postgresql": rdbmsEnginePostgreSQL,
}

// rdbmsImages are the image names, without registry or namespace, of the
// official database images and their common rebuilds (e.g. bitnami/mysql).
var rdbmsImages = map[string]string{
	"mysql":        rdbmsEngineMySQL,
	"mysql-server": rdbmsEngineMySQL,
	"mariadb":      rdbmsEngineMariaDB,
	"postgres":     rdbmsEnginePostgreSQL,
	"postgresql":   rdbmsEnginePostgreSQL,
}

var rdbmsVersionNumber = regexp.MustCompile(`\d+(\.\d+)*`)

// rdbmsVersion reads the upstream version out of a package or image
// version: the epoch ("1:") and the distribution suffix ("-0ubuntu0.22.04.1",
// "+maria~ubu2204", "-alpine") are dropped, and a version with no number in it
// ("latest") gives "".
func rdbmsVersion(version string) string {
	v := strings.TrimSpace(version)
	if i := strings.Index(v, ":"); i >= 0 {
		v = v[i+1:]
	}
	return rdbmsVersionNumber.FindString(v)
}

func rdbmsEngineOfPackage(name string) string {
	for _, p := range rdbmsServerPackages {
		if p.pattern.MatchString(name) {
			return p.engine
		}
	}
	return ""
}

// rdbmsEngineOfProcess judges a running process from its executable name, its
// version output and syft's classification. MariaDB reports a MySQL-compatible
// version and, before 10.5, runs as mysqld, so its own name in the version
// output takes precedence over the executable name.
func rdbmsEngineOfProcess(exe string, versionRaw string, syft *software.SyftMatch) string {
	engine := rdbmsServerExecutables[path.Base(exe)]
	if engine == "" && syft != nil {
		engine = rdbmsSyftNames[strings.ToLower(syft.Name)]
	}
	if engine == rdbmsEngineMySQL && strings.Contains(strings.ToLower(versionRaw), "mariadb") {
		engine = rdbmsEngineMariaDB
	}
	return engine
}

func rdbmsEngineOfImage(image string) string {
	return rdbmsImages[strings.ToLower(path.Base(getImageName(&image)))]
}

type installedPackage struct {
	source  string
	version string
}

// installedPackages indexes the DEB and RPM packages by name. A DEB whose files
// were removed but whose configuration was kept is not installed, and is left
// out.
func installedPackages(sw *software.Software) map[string]installedPackage {
	packages := make(map[string]installedPackage, len(sw.DEB)+len(sw.RPM))
	for _, deb := range sw.DEB {
		if deb.Status != "" && !strings.HasSuffix(deb.Status, " installed") {
			continue
		}
		packages[deb.Package] = installedPackage{source: installedRDBMSSourceDEB, version: deb.Version}
	}
	for _, rpm := range sw.RPM {
		packages[rpm.Name] = installedPackage{source: installedRDBMSSourceRPM, version: rpm.Version}
	}
	return packages
}

// listenerPackageSource tells which package manager a listener's package came
// from when the package is missing from the lists. The agent asks dpkg first, so
// a host with any DEB is a DEB host.
func listenerPackageSource(sw *software.Software) string {
	if len(sw.DEB) == 0 && len(sw.RPM) > 0 {
		return installedRDBMSSourceRPM
	}
	return installedRDBMSSourceDEB
}

// findInstalledRDBMS judges which of a VM's collected software is a database
// server. One running server process is one database, so a host running two
// instances of the same package reports both; a database package with no
// running process is reported once, without ports, as a stopped database.
//
// A process is matched to its software by what the agent recorded for it: the
// package owning its executable, or its PID among the processes no package
// owns. Software collected before the agent reported listeners has none, and
// its databases are reported from the package and process lists without ports.
func findInstalledRDBMS(sw *software.Software) []model.InstalledRDBMSInfo {
	packages := installedPackages(sw)
	legacyByPID := make(map[int32]software.Binary, len(sw.Legacy))
	for _, b := range sw.Legacy {
		legacyByPID[b.PID] = b
	}

	found := []model.InstalledRDBMSInfo{}
	runningPackages := make(map[string]bool)
	matchedPIDs := make(map[int32]bool)

	for _, l := range sw.Listeners {
		db := model.InstalledRDBMSInfo{
			Ports:          append([]int{}, l.Ports...),
			Package:        l.Package,
			PID:            l.PID,
			ExecutablePath: l.ExecutablePath,
			Syft:           l.Syft,
		}

		if l.Package != "" {
			pkg, ok := packages[l.Package]
			if !ok {
				// The package lists leave out packages other packages
				// depend on, such as mysql-server-core-8.0 under
				// mysql-server; the listener carries its version instead.
				if l.PackageVersion == "" {
					continue
				}
				pkg = installedPackage{source: listenerPackageSource(sw), version: l.PackageVersion}
			}
			db.Source = pkg.source
			db.Version = rdbmsVersion(pkg.version)
			db.Engine = rdbmsEngineOfPackage(l.Package)
			if db.Engine == "" {
				// A server built into a package these rules do not know
				// (e.g. a vendor fork) is still recognized by its binary.
				db.Engine = rdbmsEngineOfProcess(l.ExecutablePath, "", l.Syft)
			}
		} else {
			b := legacyByPID[l.PID]
			db.Source = installedRDBMSSourceLegacy
			db.Version = b.Version
			db.VersionRaw = b.VersionRaw
			db.Engine = rdbmsEngineOfProcess(l.ExecutablePath, b.VersionRaw, l.Syft)
		}
		if db.Engine == "" {
			continue
		}
		if db.Version == "" && l.Syft != nil {
			db.Version = l.Syft.Version
		}

		runningPackages[l.Package] = true
		matchedPIDs[l.PID] = true
		found = append(found, db)
	}

	for _, b := range sw.Legacy {
		if matchedPIDs[b.PID] {
			continue
		}
		engine := rdbmsEngineOfProcess(b.ExecutablePath, b.VersionRaw, nil)
		if engine == "" {
			continue
		}
		found = append(found, model.InstalledRDBMSInfo{
			Engine:         engine,
			Version:        b.Version,
			Source:         installedRDBMSSourceLegacy,
			Ports:          []int{},
			PID:            b.PID,
			ExecutablePath: b.ExecutablePath,
			VersionRaw:     b.VersionRaw,
		})
	}

	names := make([]string, 0, len(packages))
	for name := range packages {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		engine := rdbmsEngineOfPackage(name)
		if engine == "" || runningPackages[name] {
			continue
		}
		pkg := packages[name]
		found = append(found, model.InstalledRDBMSInfo{
			Engine:  engine,
			Version: rdbmsVersion(pkg.version),
			Source:  pkg.source,
			Ports:   []int{},
			Package: name,
		})
	}

	found = append(found, containerRDBMS(sw.Docker, installedRDBMSSourceDocker)...)
	found = append(found, containerRDBMS(sw.Podman, installedRDBMSSourcePodman)...)

	return found
}

// containerRDBMS judges containers by their image. The port reported is the
// one the database listens on inside the container, which is its configured
// port; the host port it is published on is a property of the container.
func containerRDBMS(containers []software.Container, source string) []model.InstalledRDBMSInfo {
	var found []model.InstalledRDBMSInfo

	for _, c := range containers {
		engine := rdbmsEngineOfImage(c.ContainerSummary.Image)
		if engine == "" {
			continue
		}

		seen := make(map[int]bool)
		ports := []int{}
		for _, p := range c.ContainerSummary.Ports {
			port := int(p.PrivatePort)
			if port <= 0 || seen[port] || (p.Type != "" && p.Type != "tcp") {
				continue
			}
			seen[port] = true
			ports = append(ports, port)
		}
		sort.Ints(ports)

		found = append(found, model.InstalledRDBMSInfo{
			Engine:    engine,
			Version:   rdbmsVersion(getImageTag(&c.ContainerSummary.Image)),
			Source:    source,
			Ports:     ports,
			Container: getContainerName(&c.ContainerSummary),
			Image:     c.ContainerSummary.Image,
		})
	}

	return found
}

// rdbmsPort picks the one port the source model holds: the engine's default
// port when the database listens on it, otherwise the lowest one. A database
// with no known port gives 0, which cm-beetle fills with the default.
func rdbmsPort(engine string, ports []int) int {
	if len(ports) == 0 {
		return 0
	}

	lowest := ports[0]
	for _, p := range ports {
		if p == rdbmsDefaultPorts[engine] {
			return p
		}
		if p < lowest {
			lowest = p
		}
	}
	return lowest
}

// installedRDBMSName names a database by what it was found as, so that several
// databases on one host stay distinguishable.
func installedRDBMSName(db model.InstalledRDBMSInfo) string {
	switch {
	case db.Container != "":
		return db.Container
	case db.Package != "":
		return db.Package
	default:
		return path.Base(db.ExecutablePath)
	}
}

func refineInstalledRDBMS(db model.InstalledRDBMSInfo, node rdbmsmodel.DBNodeProperty) rdbmsmodel.SourceRDBMSProperty {
	return rdbmsmodel.SourceRDBMSProperty{
		DisplayName: installedRDBMSName(db),
		DBNode:      node,
		DBEngine: rdbmsmodel.DBEngineProperty{
			Engine:        db.Engine,
			EngineVersion: db.Version,
			Port:          rdbmsPort(db.Engine, db.Ports),
		},
	}
}

// installedRDBMSNode describes the host the databases run on from its saved
// infra information. A VM whose infra has not been imported gives an empty
// node, which cm-beetle fills with its defaults: the databases are still worth
// reporting without it.
func installedRDBMSNode(connID string) rdbmsmodel.DBNodeProperty {
	savedInfraInfo, err := dao.SavedInfraInfoGet(connID)
	if err != nil || strings.TrimSpace(savedInfraInfo.InfraData) == "" {
		return rdbmsmodel.DBNodeProperty{}
	}

	var infraInfo infra.Infra
	if err := json.Unmarshal([]byte(savedInfraInfo.InfraData), &infraInfo); err != nil {
		logger.Println(logger.WARN, false, "Error occurred while parsing infra information."+
			" (ConnectionID = "+connID+")")
		return rdbmsmodel.DBNodeProperty{}
	}

	refined, err := doGetRefinedInfraInfo(&infraInfo)
	if err != nil {
		return rdbmsmodel.DBNodeProperty{}
	}

	node := rdbmsmodel.DBNodeProperty{
		Hostname:  refined.Hostname,
		MachineId: refined.MachineId,
		CPU: rdbmsmodel.CpuProperty{
			Architecture: refined.CPU.Architecture,
			Cpus:         refined.CPU.Cpus,
			Cores:        refined.CPU.Cores,
			Threads:      refined.CPU.Threads,
			MaxSpeed:     refined.CPU.MaxSpeed,
			Vendor:       refined.CPU.Vendor,
			Model:        refined.CPU.Model,
		},
		Memory: rdbmsmodel.MemoryProperty{
			Type:      refined.Memory.Type,
			TotalSize: refined.Memory.TotalSize,
			Available: refined.Memory.Available,
			Used:      refined.Memory.Used,
		},
		RootDisk: rdbmsmodel.DiskProperty{
			Label:     refined.RootDisk.Label,
			Type:      refined.RootDisk.Type,
			TotalSize: refined.RootDisk.TotalSize,
			Available: refined.RootDisk.Available,
			Used:      refined.RootDisk.Used,
		},
	}
	for _, d := range refined.DataDisks {
		node.DataDisks = append(node.DataDisks, rdbmsmodel.DiskProperty{
			Label:     d.Label,
			Type:      d.Type,
			TotalSize: d.TotalSize,
			Available: d.Available,
			Used:      d.Used,
		})
	}

	return node
}

// doGetInstalledSoftwareForRDBMS reads a VM's saved software. A VM whose
// software has not been imported is reported as not collected, so source group
// queries skip it.
func doGetInstalledSoftwareForRDBMS(connID string) (*software.Software, error) {
	savedSoftwareInfo, err := dao.SavedSoftwareInfoGet(connID)
	if err != nil {
		errMsg := "Failed to get information of the software." +
			" (ConnectionID = " + connID + ")"
		logger.Println(logger.ERROR, false, errMsg)
		return nil, fmt.Errorf("%s: %w", errMsg, errRDBMSNotCollected)
	}

	var softwareInfo software.Software
	err = json.Unmarshal([]byte(savedSoftwareInfo.SoftwareData), &softwareInfo)
	if err != nil {
		errMsg := "Error occurred while parsing software information." +
			" (ConnectionID = " + connID + ")"
		logger.Println(logger.ERROR, false, errMsg)
		return nil, errors.New(errMsg)
	}

	return &softwareInfo, nil
}
