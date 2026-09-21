package amd

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"sync"
)

// ErrAMDSMINotAvailable reports that no amd-smi binary was found.
var ErrAMDSMINotAvailable = errors.New("AMD: amd-smi command is not available")

// amdSMIExtraPaths are the locations ROCm installs amd-smi into but which are
// not on PATH unless the ROCm profile script has been sourced.
var amdSMIExtraPaths = []string{
	"/opt/rocm/bin/amd-smi",
	"/usr/local/bin/amd-smi",
	"/usr/bin/amd-smi",
}

// amdSMIStaticArgs asks for the blocks that describe the card rather than its
// current load. The live readings live behind "amd-smi metric", which is not
// collected here: this agent inventories a source host, and a device it can
// name and size is what a migration needs from it.
var amdSMIStaticArgs = []string{
	"static",
	"--asic",
	"--bus",
	"--vram",
	"--driver",
	"--json",
}

var (
	amdSMIPathOnce sync.Once
	amdSMIPath     string
)

// amdSMIBinary resolves amd-smi, falling back to the standard ROCm install
// locations when it is not on PATH.
func amdSMIBinary() (string, error) {
	amdSMIPathOnce.Do(func() {
		if path, err := exec.LookPath("amd-smi"); err == nil {
			amdSMIPath = path

			return
		}

		for _, path := range amdSMIExtraPaths {
			info, err := os.Stat(path)
			if err != nil || info.IsDir() {
				continue
			}
			amdSMIPath = path

			return
		}
	})

	if amdSMIPath == "" {
		return "", ErrAMDSMINotAvailable
	}

	return amdSMIPath, nil
}

// runAMDSMI executes amd-smi under a timeout and returns stdout on its own,
// since amd-smi prints warnings to stderr that would otherwise corrupt the
// JSON document.
func runAMDSMI(args ...string) ([]byte, error) {
	path, err := amdSMIBinary()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()

	var stdout, stderr bytes.Buffer

	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.WaitDelay = waitDelay
	cmd.Env = append(os.Environ(), "LC_ALL=C", "LANG=C")

	err = cmd.Run()
	if err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = strings.TrimSpace(stdout.String())
		}
		if message == "" {
			message = err.Error()
		}

		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, errors.New("AMD-SMI: timed out after " +
				queryTimeout.String() + ": " + oneLine(message))
		}

		return nil, errors.New("AMD-SMI: " + oneLine(message))
	}

	return stdout.Bytes(), nil
}
