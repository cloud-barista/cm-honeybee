//go:build linux && !android

package software

import (
	"debug/elf"
	"fmt"
	"path/filepath"
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
	var mappedLibs []string
	if mmaps != nil {
		for _, m := range *mmaps {
			if !strings.HasPrefix(m.Path, "/") {
				continue
			}
			mmFiles = append(mmFiles, m.Path)
			// The memory maps are the full set of shared objects actually loaded
			// into the process, i.e. the transitive runtime closure (a copied
			// non-package lib still pulls its own package-provided deps). This is
			// what determines which OS packages the target must have installed.
			if strings.Contains(filepath.Base(m.Path), ".so") {
				mappedLibs = append(mappedLibs, m.Path)
			}
		}
	}

	libPaths := filterPathsByNeeded(neededLibs, mmFiles)

	sort.Strings(neededLibs)
	sort.Strings(libPaths)
	sort.Strings(mappedLibs)

	return &BinaryInfo{
		Static:       isStatic,
		Libraries:    uniqueStr(neededLibs),
		LibraryPaths: uniqueStr(libPaths),
		MappedLibs:   uniqueStr(mappedLibs),
	}, nil
}
