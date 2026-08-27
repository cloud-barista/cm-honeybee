package software

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/cloud-barista/cm-honeybee/agent/common"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/software"
	"github.com/jollaman999/utils/logger"
	"github.com/shirou/gopsutil/v3/process"
)

type BinaryInfo struct {
	Static       bool
	Libraries    []string
	LibraryPaths []string
	MappedLibs   []string // all shared objects loaded into the process (transitive closure)
}

// launchProvenance describes how a process was started on the host. It is filled
// by the platform-specific getLaunchProvenance (see provenance_linux.go /
// provenance_windows.go).
type launchProvenance struct {
	LaunchType       string // "systemd" | "command" | "unknown"
	SystemdUnitName  string
	SystemdUnitPath  string
	SystemdEnabled   bool
	WorkingDirectory string
	ServiceType      string // systemd Type= ("simple"|"forking"|...); "simple" for command-started
	PIDFile          string
}

func GetLegacySWs() ([]software.Binary, error) {
	total := time.Now()

	procs, err := process.Processes()

	if err != nil {
		return []software.Binary{}, err
	}

	// Per-process work is accumulated rather than logged per iteration: the cost
	// here is spread over hundreds of processes, so only the totals say which
	// phase is worth attention.
	var elapsedListen, elapsedAnalyze, elapsedVersion time.Duration
	var elapsedPackages, elapsedProvenance, elapsedOwned, elapsedDeps time.Duration
	defer func() {
		common.LogElapsed("legacy", "total", total,
			fmt.Sprintf("%d procs; listen %s, owned %s, analyze %s, deps %s, version %s, packages %s, provenance %s",
				len(procs),
				elapsedListen.Round(time.Millisecond),
				elapsedOwned.Round(time.Millisecond),
				elapsedAnalyze.Round(time.Millisecond),
				elapsedDeps.Round(time.Millisecond),
				elapsedVersion.Round(time.Millisecond),
				elapsedPackages.Round(time.Millisecond),
				elapsedProvenance.Round(time.Millisecond)))
	}()

	var results []software.Binary

	// Kept index-aligned with results so required packages can be resolved for
	// all of them in one pass once the scan is done.
	var mappedLibsByResult [][]string

	// Everything the cheap first pass gathered, held until package ownership can
	// be resolved for all of them together.
	var candidates []legacyCandidate

	// ppidByPID records the parent PID of every kept process so multi-process
	// services (e.g. an Apache master plus its prefork workers, which all inherit
	// and report the same listening socket) can be collapsed to a single entry.
	ppidByPID := map[int32]int32{}

	// The agent itself is a listening, non-package process, so it would otherwise
	// be collected as a migration candidate. Skip its own process.
	selfPID := int32(os.Getpid())

	var unavailablePID = map[int32]bool{}
	var unavailableDetails []string
	var unavailableCount = map[string]int{}

	markUnavailable := func(pid int32, op string, err error) {
		if err == nil {
			return
		}

		unavailableCount[op]++
		msg := fmt.Sprintf("pid %d: %s unavailable: %v", pid, op, err)
		unavailableDetails = append(unavailableDetails, msg)
		unavailablePID[pid] = true
	}

	for _, p := range procs {
		if p.Pid == selfPID {
			continue
		}

		name, err := p.Name()
		if err != nil || name == "" {
			markUnavailable(p.Pid, "Name", err)
			continue
		}

		// A process running inside a container is migrated by the container path,
		// not as a host legacy binary. Its files live in the container image, so
		// collecting it here would yield an unreproducible binary entry (e.g. the
		// official tomcat/mariadb images expose /usr/local/tomcat, /usr/sbin/mariadbd
		// that do not exist on the host).
		if isContainerizedProcess(p.Pid) {
			logger.Println(logger.DEBUG, true,
				fmt.Sprintf("LegacySW: skipping containerized process: %s (pid %d)", name, p.Pid))
			continue
		}

		listenStart := time.Now()
		hasListen, connectionStatus := getListenStatus(p)
		elapsedListen += time.Since(listenStart)
		if !hasListen {
			continue
		}

		ppid, _ := p.Ppid()

		uids, err := p.Uids()
		if err != nil {
			markUnavailable(p.Pid, "UIDs", err)
			continue
		}

		gids, err := p.Gids()
		if err != nil {
			markUnavailable(p.Pid, "GIDs", err)
			continue
		}

		cmdline, err := p.Cmdline()
		if err != nil {
			markUnavailable(p.Pid, "Cmdline", err)
		}

		cmdSlice, err := p.CmdlineSlice()
		if err != nil {
			markUnavailable(p.Pid, "CmdlineSlice", err)
		}

		exe, err := p.Exe()
		if err != nil {
			markUnavailable(p.Pid, "Exe", err)
		}

		envs, err := p.Environ()
		if err != nil {
			markUnavailable(p.Pid, "Environ", err)
		}

		name = normalizeProcessName(name, cmdSlice)
		if name == "" {
			continue
		}

		isWine, winePrefix := detectWine(cmdSlice, envs, exe)

		candidates = append(candidates, legacyCandidate{
			proc:             p,
			name:             name,
			connectionStatus: connectionStatus,
			ppid:             ppid,
			uids:             uids,
			gids:             gids,
			cmdline:          cmdline,
			cmdSlice:         cmdSlice,
			exe:              exe,
			envs:             envs,
			isWine:           isWine,
			winePrefix:       winePrefix,
			installPath:      representativeInstallPath(exe, cmdSlice, isWine, winePrefix),
		})
	}

	// Package-managed services are migrated as packages, not as legacy binaries,
	// so a candidate whose representative install path is owned by an OS package
	// (the app dir for JVM/Wine apps, otherwise the executable) is dropped here.
	// Ownership is resolved for every candidate at once because each lookup
	// searches the whole package database.
	ownedStart := time.Now()
	installOwners := packagesOwningPaths(resolvedInstallPaths(candidates))
	elapsedOwned = time.Since(ownedStart)

	for _, c := range candidates {
		p := c.proc
		name := c.name
		connectionStatus := c.connectionStatus
		ppid := c.ppid
		uids, gids := c.uids, c.gids
		cmdline, cmdSlice := c.cmdline, c.cmdSlice
		exe, envs := c.exe, c.envs
		isWine, winePrefix := c.isWine, c.winePrefix

		if isPathOwned(c.installPath, installOwners) {
			logger.Println(logger.DEBUG, true,
				fmt.Sprintf("LegacySW: skipping package-managed process: %s (pid %d)", name, p.Pid))
			continue
		}

		analyzeStart := time.Now()
		binInfo, err := analyzeBinary(p)
		elapsedAnalyze += time.Since(analyzeStart)
		if err != nil {
			markUnavailable(p.Pid, "AnalyzeBinary", err)
		}

		isStatic := binInfo != nil && binInfo.Static
		var libs []string
		var libPaths []string
		var mappedLibs []string

		if binInfo != nil {
			libs = binInfo.Libraries
			libPaths = binInfo.LibraryPaths
			mappedLibs = binInfo.MappedLibs
		}

		openFiles, err := extractOpenFilePaths(p)

		if err != nil {
			markUnavailable(p.Pid, "OpenFiles error: %v", err)
		}

		configFiles := extractConfigFiles(cmdSlice, openFiles)
		dataDirs := detectDataDirs(openFiles)
		depsStart := time.Now()
		dependencies := collectDependencies(libPaths, envs, exe)
		elapsedDeps += time.Since(depsStart)

		provenanceStart := time.Now()
		prov := getLaunchProvenance(p.Pid)
		elapsedProvenance += time.Since(provenanceStart)

		versionStart := time.Now()
		version := detectBinaryVersion(exe, cmdSlice, envs)
		elapsedVersion += time.Since(versionStart)

		results = append(results, software.Binary{
			PID:              p.Pid,
			Name:             name,
			Version:          version,
			ConnectionStatus: connectionStatus,
			CmdlineSlice:     cmdSlice,
			Cmdline:          cmdline,
			ExecutablePath:   exe,
			Environ:          envs,
			UIDs:             uniqueInt32(uids),
			GIDs:             uniqueInt32(gids),
			Static:           isStatic,
			Libraries:        libs,
			LibraryPaths:     libPaths,
			Dependencies:     dependencies,
			OpenFilePaths:    openFiles,
			ConfigFiles:      configFiles,
			DataDirs:         dataDirs,
			IsWine:           isWine,
			WinePrefix:       winePrefix,
			LaunchType:       prov.LaunchType,
			SystemdUnitName:  prov.SystemdUnitName,
			SystemdUnitPath:  prov.SystemdUnitPath,
			SystemdEnabled:   prov.SystemdEnabled,
			WorkingDirectory: prov.WorkingDirectory,
			ServiceType:      prov.ServiceType,
			PIDFile:          prov.PIDFile,
		})
		ppidByPID[p.Pid] = ppid
		mappedLibsByResult = append(mappedLibsByResult, mappedLibs)
	}

	// Resolved after the loop, in one batch: every lookup searches the whole
	// package database, so asking per library costs tens of seconds on a process
	// that maps a hundred of them.
	packagesStart := time.Now()
	resolveRequiredPackages(results, mappedLibsByResult)
	elapsedPackages = time.Since(packagesStart)

	results = dedupeServiceWorkers(results, ppidByPID)

	logger.Println(logger.DEBUG, true, fmt.Sprintf("LegacySW : Total process (%d)", len(results)))

	for op, c := range unavailableCount {

		logger.Println(logger.DEBUG, true, fmt.Sprintf("LegacySW : %s unavailable (%d)", op, c))
	}

	return reportResults(unavailablePID, results)
}

// dedupeServiceWorkers collapses processes that belong to the same service into a
// single entry. A service like Apache (prefork/worker MPM) runs one master plus
// several worker processes that all share the same executable and inherit the
// master's listening socket, so each worker is otherwise reported as a duplicate.
// Entries are grouped by executable path; within a group the "master" (a process
// whose parent is not itself part of the group) is kept and the workers dropped.
// Processes without a resolved executable path are never merged.
func dedupeServiceWorkers(results []software.Binary, ppidByPID map[int32]int32) []software.Binary {
	type group struct {
		members []software.Binary
		pids    map[int32]bool
	}

	groups := map[string]*group{}
	var order []string

	for _, b := range results {
		key := b.ExecutablePath
		if key == "" {
			key = fmt.Sprintf("\x00pid:%d", b.PID) // keep exe-less processes distinct
		}
		g, ok := groups[key]
		if !ok {
			g = &group{pids: map[int32]bool{}}
			groups[key] = g
			order = append(order, key)
		}
		g.members = append(g.members, b)
		g.pids[b.PID] = true
	}

	var deduped []software.Binary
	for _, key := range order {
		g := groups[key]
		if len(g.members) == 1 {
			deduped = append(deduped, g.members[0])
			continue
		}

		var kept []software.Binary
		for _, b := range g.members {
			if !g.pids[ppidByPID[b.PID]] {
				kept = append(kept, b)
			}
		}

		// No in-group parent found (unexpected): fall back to the lowest PID.
		if len(kept) == 0 {
			lowest := g.members[0]
			for _, b := range g.members[1:] {
				if b.PID < lowest.PID {
					lowest = b
				}
			}
			kept = []software.Binary{lowest}
		}

		deduped = append(deduped, kept...)
	}

	return deduped
}

func normalizeProcessName(name string, cmd []string) string {
	if strings.HasPrefix(name, "[") {
		return name
	}

	if name == "sudo" && len(cmd) > 1 {
		for _, tok := range cmd[1:] {
			if !strings.HasPrefix(tok, "-") {
				return filepath.Base(tok)
			}
		}
	}
	return name
}

var flagPatterns = []struct {
	Prefix string
	Next   bool // -c next arg
}{
	{"--config=", false},
	{"--conf=", false},
	{"-c", true},
	{"--config-file=", false},
	{"--defaults-file=", false},
}

func extractConfigFiles(cmd []string, openFiles []string) []software.ConfigFile {
	seen := map[string]software.ConfigFile{}
	extractFromFlags(cmd, seen)
	extractFromOpenFD(openFiles, seen)
	var out []software.ConfigFile
	for _, v := range seen {
		out = append(out, v)
	}
	return out
}

func extractFromFlags(cmd []string, seen map[string]software.ConfigFile) {
	for i, arg := range cmd {
		for _, pat := range flagPatterns {
			// -c <path>
			if pat.Next && arg == pat.Prefix && i+1 < len(cmd) {
				seen[cmd[i+1]] = software.ConfigFile{Path: cmd[i+1], Source: "flag"}
			}
			// --config=/path/file
			if !pat.Next && strings.HasPrefix(arg, pat.Prefix) {
				path := strings.TrimPrefix(arg, pat.Prefix)
				seen[path] = software.ConfigFile{Path: path, Source: "flag"}
			}
		}
	}
}

func extractFromOpenFD(openFiles []string, seen map[string]software.ConfigFile) {
	for _, p := range openFiles {
		if isConfigExt(p) {
			seen[p] = software.ConfigFile{Path: p, Source: "openfd"}
		}
	}
}

func isConfigExt(p string) bool {
	l := strings.ToLower(p)
	return strings.HasSuffix(l, ".conf") ||
		strings.HasSuffix(l, ".ini") ||
		strings.HasSuffix(l, ".yaml") ||
		strings.HasSuffix(l, ".yml") ||
		strings.HasSuffix(l, ".json") ||
		strings.HasSuffix(l, ".properties")
}

var level3Patterns = []string{
	".db", ".sqlite", ".rdb", ".data", "ibdata", ".binlog",
}

var level2Prefixes = []string{
	"/var/lib", "/var/games", "/data", "/storage",
}

var level1Patterns = []string{
	".idx", ".meta", ".jsonl", ".wal", ".journal",
}

var excludeDirKeywords = []string{
	"cache", "tmp", "log", "font", "index",
	".local/share/Trash",
}

func detectDataDirs(openFiles []string) []string {
	candidates := map[string]int{}
	for _, path := range openFiles {
		if !strings.HasPrefix(path, "/") {
			continue
		}
		score := calculateDataEvidenceScore(path)
		if score > 0 {
			dir := filepath.Dir(path)
			candidates[dir] += score
		}
	}
	return filterValidDataDirs(candidates)
}

func calculateDataEvidenceScore(path string) int {
	score := 0
	lp := strings.ToLower(path)
	if hasAny(lp, level3Patterns) {
		score += 6
	}
	if hasAny(lp, level2Prefixes) {
		score += 3
	}
	if hasAny(lp, level1Patterns) {
		score += 1
	}
	return score
}

func filterValidDataDirs(cands map[string]int) []string {
	var result []string
	for dir, score := range cands {
		if score < 6 {
			continue
		}
		if hasAny(dir, excludeDirKeywords) {
			continue
		}

		if _, err := os.Stat(dir); err == nil {
			result = append(result, dir)
		}
	}
	sort.Strings(result)
	return uniqueStr(result)
}

// getListenStatus reports whether the process owns a listening socket and the
// status string to record. Legacy SW migration targets network-serving software,
// so a process with no LISTEN socket is not a migration candidate and is skipped
// by the caller.
func getListenStatus(p *process.Process) (bool, string) {
	conns, _ := p.Connections()
	for _, c := range conns {
		if c.Status == "LISTEN" {
			return true, c.Status
		}
	}
	return false, ""
}

// representativeInstallPath returns the path that best identifies the software for
// package-ownership checks: the app directory for JVM (catalina.home) and Wine
// (WINEPREFIX) apps, otherwise the executable. This avoids misclassifying e.g.
// Tomcat (manually installed in /opt) just because it runs on a packaged JDK.
func representativeInstallPath(exePath string, cmdSlice []string, isWine bool, winePrefix string) string {
	if home := catalinaHomeFrom(cmdSlice); home != "" {
		return home
	}
	if isWine && winePrefix != "" {
		return winePrefix
	}
	return exePath
}

// legacyCandidate is a process that passed the cheap filters, held until package
// ownership can be resolved for every candidate in one batch.
type legacyCandidate struct {
	proc             *process.Process
	name             string
	connectionStatus string
	ppid             int32
	uids             []int32
	gids             []int32
	cmdline          string
	cmdSlice         []string
	exe              string
	envs             []string
	isWine           bool
	winePrefix       string
	installPath      string
}

// resolvedInstallPaths returns each candidate's install path along with its
// symlink target, which is what an ownership lookup has to be asked about.
func resolvedInstallPaths(candidates []legacyCandidate) []string {
	seen := map[string]bool{}
	var paths []string

	for _, c := range candidates {
		for _, path := range ownershipCandidates(c.installPath) {
			if !seen[path] {
				seen[path] = true
				paths = append(paths, path)
			}
		}
	}

	return paths
}

// ownershipCandidates returns the paths to ask a package manager about for path:
// the path itself and, when it is a symlink, its target.
func ownershipCandidates(path string) []string {
	if path == "" {
		return nil
	}

	candidates := []string{path}
	if real, err := filepath.EvalSymlinks(path); err == nil && real != path {
		candidates = append(candidates, real)
	}

	return candidates
}

// isPathOwned reports whether path is provided by an installed OS package,
// answering from an ownership map built earlier by packagesOwningPaths.
func isPathOwned(path string, owners map[string]string) bool {
	for _, c := range ownershipCandidates(path) {
		if owners[c] != "" {
			return true
		}
	}

	return false
}

// isPackageOwned reports whether path is provided by an installed OS package
// (dpkg on Debian/Ubuntu, rpm on RHEL/Fedora). Such software is migrated by the
// package path, not as a legacy binary.
func isPackageOwned(path string) bool {
	if path == "" {
		return false
	}

	real := path
	if r, err := filepath.EvalSymlinks(path); err == nil {
		real = r
	}

	if _, err := exec.LookPath("dpkg"); err == nil {
		return exec.Command("dpkg", "-S", real).Run() == nil
	}
	if _, err := exec.LookPath("rpm"); err == nil {
		return exec.Command("rpm", "-qf", real).Run() == nil
	}
	return false
}

// resolveRequiredPackages fills RequiredPackages for every collected binary: the
// OS packages that provide the package-owned shared objects the process loaded
// (e.g. libpcre2-8 for a source-built Apache, plus transitive deps such as
// libexpat pulled in by a copied non-package lib). These are NOT copied
// (collectDependencies excludes package-owned paths); instead the target must
// install them via its package manager, otherwise the migrated binary would be
// missing its runtime libraries. Non-package libs map to no package and are
// skipped (they are copied instead).
//
// mappedLibsByResult is index-aligned with results. Every distinct path across
// all of them is resolved in one batch, because a single lookup searches the
// entire package database.
func resolveRequiredPackages(results []software.Binary, mappedLibsByResult [][]string) {
	// The same library is mapped by many processes, so resolve each distinct
	// path only once.
	candidatesByLib := map[string][]string{}
	seen := map[string]bool{}
	var allPaths []string

	for _, libs := range mappedLibsByResult {
		for _, lib := range libs {
			if lib == "" {
				continue
			}
			if _, done := candidatesByLib[lib]; done {
				continue
			}

			candidates := []string{lib}
			if real, err := filepath.EvalSymlinks(lib); err == nil && real != lib {
				candidates = append(candidates, real)
			}
			candidatesByLib[lib] = candidates

			for _, c := range candidates {
				if !seen[c] {
					seen[c] = true
					allPaths = append(allPaths, c)
				}
			}
		}
	}

	owners := packagesOwningPaths(allPaths)

	for i := range results {
		if i >= len(mappedLibsByResult) {
			break
		}

		found := map[string]bool{}
		var pkgs []string

		for _, lib := range mappedLibsByResult[i] {
			for _, c := range candidatesByLib[lib] {
				name := owners[c]
				if name == "" {
					continue
				}
				if !found[name] {
					found[name] = true
					pkgs = append(pkgs, name)
				}
				break
			}
		}

		sort.Strings(pkgs)
		results[i].RequiredPackages = pkgs
	}
}

// dpkgInfoDir holds one .list file per installed package, naming the files that
// package owns.
const dpkgInfoDir = "/var/lib/dpkg/info"

// packagesOwningPaths maps each path to the OS package providing it, omitting
// paths no package owns.
func packagesOwningPaths(paths []string) map[string]string {
	owners := make(map[string]string, len(paths))
	if len(paths) == 0 {
		return owners
	}

	if _, err := exec.LookPath("dpkg"); err == nil {
		if resolveFromDpkgFileLists(paths, owners) {
			return owners
		}
	}

	// An rpm host, or a dpkg host whose file lists could not be read: ask the
	// package manager per path.
	for _, path := range paths {
		if name := packageNameOwning(path); name != "" {
			owners[path] = name
		}
	}

	return owners
}

// resolveFromDpkgFileLists answers every lookup with a single scan of the dpkg
// file lists. `dpkg -S` re-searches the whole database per invocation, which
// costs tens of seconds once a process maps a hundred libraries. Reports whether
// the lists could be read at all; a path missing from them is not package-owned.
func resolveFromDpkgFileLists(paths []string, owners map[string]string) bool {
	entries, err := os.ReadDir(dpkgInfoDir)
	if err != nil {
		return false
	}

	wanted := make(map[string]bool, len(paths))
	for _, path := range paths {
		wanted[path] = true
	}

	for _, entry := range entries {
		fileName := entry.Name()
		if !strings.HasSuffix(fileName, ".list") {
			continue
		}

		pkg := strings.TrimSuffix(fileName, ".list")
		if idx := strings.IndexByte(pkg, ':'); idx >= 0 { // strip multiarch suffix
			pkg = pkg[:idx]
		}

		fd, err := os.Open(filepath.Join(dpkgInfoDir, fileName))
		if err != nil {
			continue
		}

		scanner := bufio.NewScanner(fd)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			if wanted[line] {
				if _, exists := owners[line]; !exists {
					owners[line] = pkg
				}
			}
		}

		_ = fd.Close()
	}

	return true
}

// packageNameOwning returns the name of the OS package that provides path, or ""
// if none (or no package manager). It mirrors isPackageOwned but extracts the
// package name (dpkg on Debian/Ubuntu, rpm on RHEL/Fedora).
func packageNameOwning(path string) string {
	candidates := []string{path}
	if r, err := filepath.EvalSymlinks(path); err == nil && r != path {
		candidates = append(candidates, r)
	}

	if _, err := exec.LookPath("dpkg"); err == nil {
		for _, c := range candidates {
			out, err := exec.Command("dpkg", "-S", c).Output()
			if err != nil {
				continue
			}
			// "pkg:arch: /path" or "pkg: /path"; multiple providers are newline-separated.
			line := strings.SplitN(strings.TrimSpace(string(out)), "\n", 2)[0]
			idx := strings.LastIndex(line, ": ")
			if idx < 0 {
				continue
			}
			pkg := strings.TrimSpace(line[:idx])
			if c := strings.IndexByte(pkg, ':'); c >= 0 { // strip multiarch suffix
				pkg = pkg[:c]
			}
			if pkg != "" {
				return pkg
			}
		}
		return ""
	}

	if _, err := exec.LookPath("rpm"); err == nil {
		for _, c := range candidates {
			out, err := exec.Command("rpm", "-qf", "--queryformat", "%{NAME}", c).Output()
			if err != nil {
				continue
			}
			if name := strings.TrimSpace(string(out)); name != "" {
				return name
			}
		}
	}

	return ""
}

// collectDependencies returns the non-package-owned runtime paths that must be
// copied for a legacy binary: linked libraries outside any OS package, plus a
// manually-installed JDK home. Package-provided runtimes (apt/dnf JDK, system
// libs) are installed by package migration instead, so they are excluded.
func collectDependencies(libPaths []string, environ []string, exePath string) []string {
	var deps []string

	for _, lp := range libPaths {
		if lp != "" && !isPackageOwned(lp) {
			deps = append(deps, lp)
		}
	}

	if jh := javaHomeFrom(environ, exePath); jh != "" && !isPackageOwned(jh) {
		found := false
		for _, d := range deps {
			if d == jh {
				found = true
				break
			}
		}
		if !found {
			deps = append(deps, jh)
		}
	}

	return deps
}

// detectWine reports whether a process runs under Wine and its WINEPREFIX bottle.
// An explicit WINEPREFIX env wins; otherwise it falls back to a heuristic (a Wine
// loader in the executable/command line, or a .exe argument) and defaults the
// bottle to the user's ~/.wine.
func detectWine(cmdSlice []string, envs []string, exePath string) (bool, string) {
	for _, e := range envs {
		if v, ok := strings.CutPrefix(e, "WINEPREFIX="); ok {
			return true, strings.TrimSpace(v)
		}
	}

	if !looksLikeWine(exePath, cmdSlice) {
		return false, ""
	}

	// Default Wine bottle is $HOME/.wine.
	for _, e := range envs {
		if home, ok := strings.CutPrefix(e, "HOME="); ok {
			if home = strings.TrimSpace(home); home != "" {
				return true, filepath.Join(home, ".wine")
			}
		}
	}
	return true, ""
}

var wineLoaders = map[string]bool{
	"wine": true, "wine64": true, "wine-preloader": true,
	"wine64-preloader": true, "wineserver": true,
}

// looksLikeWine detects a Wine process from its executable / argv[0] being a Wine
// loader, or any argument being a Windows .exe.
func looksLikeWine(exePath string, cmdSlice []string) bool {
	if exePath != "" && wineLoaders[filepath.Base(exePath)] {
		return true
	}
	for i, arg := range cmdSlice {
		if i == 0 && wineLoaders[filepath.Base(arg)] {
			return true
		}
		if strings.HasSuffix(strings.ToLower(arg), ".exe") {
			return true
		}
	}
	return false
}

func filterLibNames(libs []string) []string {
	excl := []string{"libc", "ld-linux", "linux-vdso"}
	var out []string
	for _, lib := range libs {
		if !containsAny(lib, excl) {
			out = append(out, lib)
		}
	}
	return out
}

func filterPathsByNeeded(needed []string, paths []string) []string {
	var out []string
	for _, p := range paths {
		base := filepath.Base(p)
		for _, need := range needed {
			if matchSONAME(need, base) {
				out = append(out, p)
				break
			}
		}
	}
	return out
}
func matchSONAME(need, actual string) bool {
	return actual == need || strings.HasPrefix(actual, need+".")
}

func extractOpenFilePaths(p *process.Process) ([]string, error) {
	files, err := p.OpenFiles()
	if err != nil {
		return nil, err
	}
	var out []string
	for _, f := range files {
		p := f.Path
		if p == "" ||
			strings.HasPrefix(p, "socket:") ||
			strings.HasPrefix(p, "pipe:") ||
			strings.Contains(p, "(deleted)") {
			continue
		}
		if strings.HasPrefix(p, "/") {
			out = append(out, p)
		}
	}

	return out, nil
}

func containsAny(s string, patterns []string) bool {
	for _, p := range patterns {
		if strings.Contains(s, p) {
			return true
		}
	}
	return false
}
func hasAny(s string, patterns []string) bool {
	l := strings.ToLower(s)
	for _, p := range patterns {
		if strings.Contains(l, strings.ToLower(p)) {
			return true
		}
	}
	return false
}

func uniqueStr(vals []string) []string {
	m := map[string]bool{}
	out := []string{}
	for _, v := range vals {
		if !m[v] {
			m[v] = true
			out = append(out, v)
		}
	}
	return out
}

func uniqueInt32(vals []int32) []int32 {
	m := make(map[int32]bool)
	var out []int32
	for _, v := range vals {
		if !m[v] {
			m[v] = true
			out = append(out, v)
		}
	}
	return out
}

func reportResults(failedPID map[int32]bool, results []software.Binary) ([]software.Binary, error) {
	if len(failedPID) == 0 {
		return results, nil
	}

	var pids []int32
	for pid := range failedPID {
		pids = append(pids, pid)
	}
	sort.Slice(pids, func(i, j int) bool {
		return pids[i] < pids[j]
	})

	var failedPIDs []string
	for _, pid := range pids {
		failedPIDs = append(failedPIDs, fmt.Sprintf("%d", pid))
	}

	logger.Println(logger.DEBUG, true,
		fmt.Sprintf("LegacySW partial data access in %d processes: [%s]",
			len(failedPIDs),
			strings.Join(failedPIDs, ", "),
		),
	)

	return results, nil
}
