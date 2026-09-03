package nvidia

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
	// queryTimeout bounds `nvidia-smi -q -x`. A wedged driver makes nvidia-smi
	// block indefinitely, and without a bound that block propagates all the way
	// up to the infra collection request.
	queryTimeout = 30 * time.Second
	// versionTimeout bounds the much cheaper `nvidia-smi --version`.
	versionTimeout = 10 * time.Second
	// waitDelay bounds how long we wait for the output pipes after the process
	// was killed, so a child holding stdout cannot keep us blocked.
	waitDelay = 5 * time.Second
)

// ErrNotAvailable reports that no nvidia-smi binary was found in PATH.
var ErrNotAvailable = errors.New("NVIDIA: nvidia-smi command is not available")

var (
	binPathOnce sync.Once
	binPath     string

	nvmlVersionOnce sync.Once
	nvmlVersion     string
)

// smiPath resolves nvidia-smi in PATH. Presence of the binary is the only
// thing it decides; whether the driver actually answers is decided by running
// the real query, because nvidia-smi still prints its help text on a host
// whose driver is not loaded.
func smiPath() (string, error) {
	binPathOnce.Do(func() {
		path, err := exec.LookPath("nvidia-smi")
		if err != nil {
			return
		}
		binPath = path
	})

	if binPath == "" {
		return "", ErrNotAvailable
	}

	return binPath, nil
}

// runNVIDIASmi executes nvidia-smi with the given arguments under a timeout.
// stdout is returned on its own: nvidia-smi writes warnings to stderr, and
// folding those into stdout would corrupt the XML document we are about to
// parse. The locale is pinned so number formatting does not follow the
// agent's environment.
func runNVIDIASmi(timeout time.Duration, args ...string) ([]byte, error) {
	path, err := smiPath()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
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
			return nil, errors.New("NVIDIA-SMI: timed out after " +
				timeout.String() + ": " + oneLine(message))
		}

		return nil, errors.New("NVIDIA-SMI: " + oneLine(message))
	}

	return stdout.Bytes(), nil
}

// oneLine flattens a multi-line command output so it stays readable as a
// single entry of the collected error list.
func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// getNVMLVersion reads the NVML library version from `nvidia-smi --version`.
// The v13 schema deprecates driver_version in the query output, and NVML's own
// version is not in the query output at all, so it comes from here. It is a
// static value, so it is read once per process. Drivers older than R555 have
// no such line and yield an empty string, which is then simply not reported.
func getNVMLVersion() string {
	nvmlVersionOnce.Do(func() {
		output, err := runNVIDIASmi(versionTimeout, "--version")
		if err != nil {
			return
		}
		nvmlVersion = parseNVMLVersion(string(output))
	})

	return nvmlVersion
}

func parseNVMLVersion(output string) string {
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(strings.ToLower(trimmed), "nvml version") {
			continue
		}

		colon := strings.Index(trimmed, ":")
		if colon < 0 {
			continue
		}

		return strings.TrimSpace(trimmed[colon+1:])
	}

	return ""
}
