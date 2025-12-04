package software

import (
	"debug/elf"
	"fmt"
	"os"
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
}

func GetLegacySWs() ([]software.Binary, error) {
	procs, err := process.Processes()

	if err != nil {
		return []software.Binary{}, err
	}

	var results []software.Binary

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
		name, err := p.Name()
		if err != nil || name == "" {
			markUnavailable(p.Pid, "Name", err)
			continue
		}

		hasListen, connectionStatus := getListenStatus(p)
		if !hasListen {
			continue
		}

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

		binInfo, err := analyzeBinary(p)
		if err != nil {
			markUnavailable(p.Pid, "AnalyzeBinary", err)
		}

		isStatic := binInfo != nil && binInfo.Static
		var libs []string
		var libPaths []string

		if binInfo != nil {
			libs = binInfo.Libraries
			libPaths = binInfo.LibraryPaths
		}

		openFiles, err := extractOpenFilePaths(p)

		if err != nil {
			markUnavailable(p.Pid, "OpenFiles error: %v", err)
		}

		configFiles := extractConfigFiles(cmdSlice, openFiles)
		dataDirs := detectDataDirs(openFiles)
		isWine, winePrefix := detectWine(envs)

		results = append(results, software.Binary{
			PID:              p.Pid,
			Name:             name,
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
			OpenFilePaths:    openFiles,
			ConfigFiles:      configFiles,
			DataDirs:         dataDirs,
			IsWine:           isWine,
			WinePrefix:       winePrefix,
		})
	}

	logger.Println(logger.DEBUG, true, fmt.Sprintf("LegacySW : Total process (%d)", len(results)))

	for op, c := range unavailableCount {

		logger.Println(logger.DEBUG, true, fmt.Sprintf("LegacySW : %s unavailable (%d)", op, c))
	}

	return reportResults(unavailablePID, results)
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

func getListenStatus(p *process.Process) (bool, string) {
	conns, _ := p.Connections()
	for _, c := range conns {
		if c.Status == "LISTEN" {
			return true, c.Status
		}
	}
	return true, ""
}

func detectWine(envs []string) (bool, string) {
	for _, e := range envs {
		if strings.HasPrefix(e, "WINEPREFIX=") {
			return true, strings.TrimPrefix(e, "WINEPREFIX=")
		}
	}
	return false, ""
}

func analyzeBinary(p *process.Process) (*BinaryInfo, error) {
	exe, err := p.Exe()
	if err != nil {
		return nil, err
	}

	f, err := elf.Open(exe)
	if err != nil {
		return nil, err
	}

	defer func() {
		if cerr := f.Close(); cerr != nil {
			logger.Println(logger.DEBUG, true,
				fmt.Sprintf("pid %d: ELF Close warning: %v", p.Pid, cerr))
		}
	}()

	isStatic := true
	for _, ph := range f.Progs {
		if ph.Type == elf.PT_INTERP {
			isStatic = false
			break
		}
	}

	if isStatic {
		return &BinaryInfo{Static: true}, nil
	}

	neededLibs, _ := f.ImportedLibraries()
	neededLibs = filterLibNames(neededLibs)

	mmaps, _ := p.MemoryMaps(false)
	var mmFiles []string
	if mmaps != nil {
		for _, m := range *mmaps {
			if strings.HasPrefix(m.Path, "/") {
				mmFiles = append(mmFiles, m.Path)
			}
		}
	}

	libPaths := filterPathsByNeeded(neededLibs, mmFiles)

	sort.Strings(neededLibs)
	sort.Strings(libPaths)

	return &BinaryInfo{
		Static:       isStatic,
		Libraries:    uniqueStr(neededLibs),
		LibraryPaths: uniqueStr(libPaths),
	}, nil
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
