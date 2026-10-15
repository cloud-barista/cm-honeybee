package software

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/cloud-barista/cm-honeybee/agent/common"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/software"
	"github.com/jollaman999/utils/logger"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

// GetListeners reports every host process that owns a LISTEN socket, together
// with its listening ports and the OS package owning its executable. Unlike
// GetLegacySWs it keeps package-owned processes: its purpose is to say which
// installed software is serving on which port, whatever installed it.
// Containerized processes are left to the container collectors, except those of
// a container on the host's network, whose ports the runtime does not record;
// workers of a multi-process service are collapsed into the service's master.
//
// Only a failure to list processes is returned as an error; a process whose
// details cannot be read is skipped.
func GetListeners() ([]software.Listener, error) {
	total := time.Now()

	procs, err := process.Processes()
	if err != nil {
		return []software.Listener{}, err
	}

	var elapsedConns, elapsedOwned, elapsedSyft time.Duration
	defer func() {
		common.LogElapsed("listeners", "total", total,
			fmt.Sprintf("%d procs; connections %s, owned %s, syft %s",
				len(procs),
				elapsedConns.Round(time.Millisecond),
				elapsedOwned.Round(time.Millisecond),
				elapsedSyft.Round(time.Millisecond)))
	}()

	// The agent serves its own API, so it would otherwise report itself.
	selfPID := int32(os.Getpid())

	listeners := make([]software.Listener, 0)
	ppidByPID := map[int32]int32{}
	unavailableCount := map[string]int{}

	for _, p := range procs {
		if p.Pid == selfPID {
			continue
		}

		name, err := p.Name()
		if err != nil || name == "" {
			unavailableCount["Name"]++
			continue
		}

		containerID := ""
		if isContainerizedProcess(p.Pid) {
			if !sharesHostNetwork(p.Pid) {
				continue
			}
			containerID = containerIDOfProcess(p.Pid)
			if containerID == "" {
				continue
			}
		}

		connsStart := time.Now()
		conns, err := p.Connections()
		elapsedConns += time.Since(connsStart)
		if err != nil {
			unavailableCount["Connections"]++
			continue
		}

		ports := listenPorts(conns)
		if len(ports) == 0 {
			continue
		}

		// Kept without an executable path, like GetLegacySWs does: the ports
		// alone still tell which process serves where.
		exe, err := p.Exe()
		if err != nil {
			unavailableCount["Exe"]++
		}

		ppid, _ := p.Ppid()
		ppidByPID[p.Pid] = ppid

		listeners = append(listeners, software.Listener{
			PID:            p.Pid,
			Name:           name,
			ExecutablePath: exe,
			Ports:          ports,
			ContainerID:    containerID,
		})
	}

	listeners = dedupeListenerWorkers(listeners, ppidByPID)

	// Resolved in one batch because each lookup searches the whole package
	// database. A container's executable path names a file inside the
	// container, so looking it up on the host would describe whatever host
	// file has the same path.
	ownedStart := time.Now()
	exes := make([]string, 0, len(listeners))
	for _, l := range listeners {
		if l.ContainerID == "" {
			exes = append(exes, l.ExecutablePath)
		}
	}
	owners := packagesOwningPaths(ownershipLookupPaths(exes))
	var ownerNames []string
	for i := range listeners {
		if listeners[i].ContainerID != "" {
			continue
		}
		listeners[i].Package = packageOwningPath(listeners[i].ExecutablePath, owners)
		if listeners[i].Package != "" {
			ownerNames = append(ownerNames, listeners[i].Package)
		}
	}
	versions := packageVersions(ownerNames)
	for i := range listeners {
		listeners[i].PackageVersion = versions[listeners[i].Package]
	}
	elapsedOwned = time.Since(ownedStart)

	syftStart := time.Now()
	attachSyftMatches(listeners)
	elapsedSyft = time.Since(syftStart)

	logger.Println(logger.DEBUG, true, fmt.Sprintf("Listeners: Total process (%d)", len(listeners)))
	for op, c := range unavailableCount {
		logger.Println(logger.DEBUG, true, fmt.Sprintf("Listeners: %s unavailable (%d)", op, c))
	}

	return listeners, nil
}

// listenPorts returns the local ports of the LISTEN sockets among conns, sorted
// and without duplicates (a service listening on both IPv4 and IPv6 reports the
// same port twice). UDP sockets have no LISTEN state and are not included.
func listenPorts(conns []net.ConnectionStat) []int {
	seen := map[int]bool{}
	var ports []int

	for _, c := range conns {
		if c.Status != "LISTEN" || c.Laddr.Port == 0 {
			continue
		}
		port := int(c.Laddr.Port)
		if !seen[port] {
			seen[port] = true
			ports = append(ports, port)
		}
	}

	sort.Ints(ports)
	return ports
}

// packageVersions asks the package manager for the installed version of each
// named package, with one invocation for all of them. The package manager is
// chosen in the same order as packagesOwningPaths, so the versions belong to the
// packages it reported. A package the manager does not know is left out.
func packageVersions(names []string) map[string]string {
	if len(names) == 0 {
		return map[string]string{}
	}

	var cmd *exec.Cmd
	if _, err := exec.LookPath("dpkg-query"); err == nil {
		cmd = exec.Command("dpkg-query", append([]string{"-W", "-f=${Package}\t${Version}\n"}, names...)...)
	} else if _, err := exec.LookPath("rpm"); err == nil {
		cmd = exec.Command("rpm", append([]string{"-q", "--queryformat", "%{NAME}\t%{VERSION}\n"}, names...)...)
	} else {
		return map[string]string{}
	}

	// Both exit non-zero when any name is unknown but still print the others.
	out, err := cmd.Output()
	if err != nil && len(out) == 0 {
		logger.Println(logger.DEBUG, false, "Listeners: Failed to get package versions ("+err.Error()+")")
	}

	return parsePackageVersions(string(out))
}

// parsePackageVersions reads "name\tversion" lines. Other lines, such as rpm's
// "package X is not installed", have no tab and are skipped.
func parsePackageVersions(out string) map[string]string {
	versions := map[string]string{}
	for _, line := range strings.Split(out, "\n") {
		name, version, ok := strings.Cut(line, "\t")
		if !ok || name == "" || version == "" {
			continue
		}
		if _, exists := versions[name]; !exists {
			versions[name] = version
		}
	}

	return versions
}

// dedupeListenerWorkers collapses the workers of a multi-process service into
// its master, the same way dedupeServiceWorkers does for legacy binaries:
// workers inherit the master's listening socket and would each be reported.
func dedupeListenerWorkers(listeners []software.Listener, ppidByPID map[int32]int32) []software.Listener {
	return dedupeByService(listeners, ppidByPID,
		func(l software.Listener) int32 { return l.PID },
		func(l software.Listener) string { return l.ExecutablePath })
}
