package software

import (
	"fmt"
	"os"
	"sort"
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
// Containerized processes are left to the container collectors, and workers of a
// multi-process service are collapsed into the service's master.
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

		if isContainerizedProcess(p.Pid) {
			continue
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
		})
	}

	listeners = dedupeListenerWorkers(listeners, ppidByPID)

	// Resolved in one batch because each lookup searches the whole package
	// database.
	ownedStart := time.Now()
	exes := make([]string, 0, len(listeners))
	for _, l := range listeners {
		exes = append(exes, l.ExecutablePath)
	}
	owners := packagesOwningPaths(ownershipLookupPaths(exes))
	for i := range listeners {
		listeners[i].Package = packageOwningPath(listeners[i].ExecutablePath, owners)
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

// dedupeListenerWorkers collapses the workers of a multi-process service into
// its master, the same way dedupeServiceWorkers does for legacy binaries:
// workers inherit the master's listening socket and would each be reported.
func dedupeListenerWorkers(listeners []software.Listener, ppidByPID map[int32]int32) []software.Listener {
	return dedupeByService(listeners, ppidByPID,
		func(l software.Listener) int32 { return l.PID },
		func(l software.Listener) string { return l.ExecutablePath })
}
