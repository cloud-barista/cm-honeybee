//go:build linux && !android

package software

import (
	"debug/elf"
	"fmt"
	"sort"
	"strings"

	"github.com/jollaman999/utils/logger"
	"github.com/shirou/gopsutil/v3/process"
)

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
