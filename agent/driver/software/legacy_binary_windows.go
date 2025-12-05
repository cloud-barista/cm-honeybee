//go:build windows

package software

import "github.com/shirou/gopsutil/v3/process"

// Windows: ELF/Mmaps 미지원
func analyzeBinary(p *process.Process) (*BinaryInfo, error) {

	return &BinaryInfo{
		Static:       false,
		Libraries:    nil,
		LibraryPaths: nil,
	}, nil
}
