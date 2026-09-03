package amd

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const (
	// queryTimeout bounds a rocm-smi run. Like nvidia-smi, rocm-smi can block
	// on a wedged driver, and that block would otherwise propagate up to the
	// infra collection request.
	queryTimeout = 30 * time.Second
	// waitDelay bounds how long we wait for the output pipes after the process
	// was killed.
	waitDelay = 5 * time.Second
)

// ErrNotAvailable reports that no rocm-smi binary was found.
var ErrNotAvailable = errors.New("AMD: rocm-smi command is not available")

// extraPaths are the locations ROCm installs into but which are not on PATH
// unless the ROCm profile script has been sourced.
var extraPaths = []string{
	"/opt/rocm/bin/rocm-smi",
	"/usr/bin/rocm-smi",
}

// queryArgs asks for the inventory-relevant blocks by name. rocm-smi's "-a"
// shorthand does not cover all of them on every release, so they are listed
// explicitly and "-a" is only the fallback for a release that rejects one of
// these flags.
var queryArgs = []string{
	"--showid",
	"--showuniqueid",
	"--showvbios",
	"--showproductname",
	"--showserial",
	"--showbus",
	"--showdriverversion",
	"--showtemp",
	"--showuse",
	"--showmemuse",
	"--showmeminfo", "vram",
	"--showpower",
	"--showmaxpower",
	"--showfan",
	"--showperflevel",
	"--showclocks",
	"--json",
}

var fallbackArgs = []string{"-a", "--json"}

var (
	binPathOnce sync.Once
	binPath     string
)

// smiPath resolves rocm-smi, falling back to the standard ROCm install
// locations when it is not on PATH.
func smiPath() (string, error) {
	binPathOnce.Do(func() {
		if path, err := exec.LookPath("rocm-smi"); err == nil {
			binPath = path

			return
		}

		for _, path := range extraPaths {
			info, err := os.Stat(path)
			if err != nil || info.IsDir() {
				continue
			}
			binPath = path

			return
		}
	})

	if binPath == "" {
		return "", ErrNotAvailable
	}

	return binPath, nil
}

// runROCmSMI executes rocm-smi under a timeout and returns stdout on its own,
// since rocm-smi prints warnings to stderr that would otherwise corrupt the
// JSON document.
func runROCmSMI(args ...string) ([]byte, error) {
	path, err := smiPath()
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
			return nil, errors.New("ROCM-SMI: timed out after " +
				queryTimeout.String() + ": " + oneLine(message))
		}

		return nil, errors.New("ROCM-SMI: " + oneLine(message))
	}

	return stdout.Bytes(), nil
}

func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
