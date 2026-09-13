package nvidia

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jollaman999/utils/logger"
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

// gpuLine and migLine match the two shapes `nvidia-smi -L` prints. The MIG
// line is indented under the GPU it belongs to, which is what separates them.
var (
	gpuLine = regexp.MustCompile(`^GPU (\d+):.*\(UUID: (GPU-[^)]+)\)`)
	migLine = regexp.MustCompile(`^\s+MIG \S+\s+Device\s+(\d+):.*\(UUID: (MIG-[^)]+)\)`)
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

// migUUIDsByDevice reads the MIG instance UUIDs out of `nvidia-smi -L`, keyed
// by the GPU index and then the MIG device index.
//
// The query output is the primary source for MIG instances, but some drivers
// leave the <uuid> element out of <mig_device> even though the schema carries
// it. Measured on a MIG-backed vGPU host running 595.71.03: the XML held no MIG
// UUID at all while `-L` reported them at the same moment. Without this the
// field stays empty and the caller has no way to name an instance.
func migUUIDsByDevice() map[int]map[int]string {
	output, err := runNVIDIASmi(versionTimeout, "-L")
	if err != nil {
		logger.Println(logger.DEBUG, false, "NVIDIA: cannot list MIG devices: "+err.Error())

		return nil
	}

	return parseMIGListing(output)
}

// parseMIGListing pulls the MIG UUIDs out of an `nvidia-smi -L` listing, which
// indents each MIG line under the GPU it belongs to:
//
//	GPU 0: NVIDIA RTX PRO 6000 Blackwell Server Edition (UUID: GPU-e821...)
//	  MIG 2g.48gb     Device  0: (UUID: MIG-e085...)
//
// A host with no MIG instances yields an empty map, as does output that is not
// a listing at all.
func parseMIGListing(output []byte) map[int]map[int]string {
	uuids := make(map[int]map[int]string)
	gpuIndex := -1

	for _, line := range strings.Split(string(output), "\n") {
		if m := gpuLine.FindStringSubmatch(line); m != nil {
			n, err := strconv.Atoi(m[1])
			if err != nil {
				gpuIndex = -1

				continue
			}
			gpuIndex = n

			continue
		}

		m := migLine.FindStringSubmatch(line)
		if m == nil || gpuIndex < 0 {
			continue
		}

		device, err := strconv.Atoi(m[1])
		if err != nil {
			continue
		}

		if uuids[gpuIndex] == nil {
			uuids[gpuIndex] = make(map[int]string)
		}
		uuids[gpuIndex][device] = m[2]
	}

	return uuids
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
