package software

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

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
	procs, err := process.Processes()

	if err != nil {
		return []software.Binary{}, err
	}

	var results []software.Binary

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

		hasListen, connectionStatus := getListenStatus(p)
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

		// Package-managed services are migrated as packages, not as legacy binaries.
		// Skip a candidate whose representative install path is owned by an OS package
		// (the app dir for JVM/Wine apps, otherwise the executable).
		if isPackageOwned(representativeInstallPath(exe, cmdSlice, isWine, winePrefix)) {
			logger.Println(logger.DEBUG, true,
				fmt.Sprintf("LegacySW: skipping package-managed process: %s (pid %d)", name, p.Pid))
			continue
		}

		binInfo, err := analyzeBinary(p)
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
		dependencies := collectDependencies(libPaths, envs, exe)
		requiredPackages := collectRequiredPackages(mappedLibs)
		prov := getLaunchProvenance(p.Pid)
		version := detectBinaryVersion(exe, cmdSlice, envs)

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
			RequiredPackages: requiredPackages,
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
	}

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

// collectRequiredPackages returns the OS packages that provide the package-owned
// shared objects loaded by the process (e.g. libpcre2-8 for a source-built
// Apache, plus transitive deps such as libexpat pulled in by a copied non-package
// lib). These are NOT copied (collectDependencies excludes package-owned paths);
// instead the target must install them via its package manager, otherwise the
// migrated binary would be missing its runtime libraries. Non-package libs map to
// no package and are skipped here (they are copied instead).
func collectRequiredPackages(mappedLibs []string) []string {
	seen := map[string]bool{}
	var pkgs []string

	for _, lp := range mappedLibs {
		if lp == "" {
			continue
		}
		name := packageNameOwning(lp)
		if name != "" && !seen[name] {
			seen[name] = true
			pkgs = append(pkgs, name)
		}
	}

	sort.Strings(pkgs)
	return pkgs
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
