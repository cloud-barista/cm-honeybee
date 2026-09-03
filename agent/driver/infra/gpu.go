package infra

import (
	"strings"
	"sync"

	"github.com/cloud-barista/cm-honeybee/agent/gpu/amd"
	"github.com/cloud-barista/cm-honeybee/agent/gpu/drm"
	"github.com/cloud-barista/cm-honeybee/agent/gpu/nvidia"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/infra"
)

// GetGPUInfo collects the GPUs of the host from every vendor tool that is
// present, plus the kernel DRM drivers behind them.
//
// It never fails the surrounding infra collection: a vendor tool that is
// missing or does not answer is reported through the Errors list, so a host
// with no GPU and a host whose driver is broken stay distinguishable.
func GetGPUInfo() (infra.GPU, error) {
	var gpu infra.GPU

	var (
		wg sync.WaitGroup

		nvidiaResult nvidia.Result
		nvidiaErr    error

		amdGPUs []infra.AMD
		amdErr  error

		drmDevices []infra.DRM
		drmErr     error
	)

	// The three collectors are independent and each runs an external command
	// under its own timeout, so running them together bounds the total wait by
	// the slowest one instead of their sum.
	wg.Add(3)

	go func() {
		defer wg.Done()
		nvidiaResult, nvidiaErr = nvidia.QueryGPU()
	}()

	go func() {
		defer wg.Done()
		amdGPUs, amdErr = amd.QueryGPU()
	}()

	go func() {
		defer wg.Done()
		drmDevices, drmErr = drm.GetDRMInfo()
	}()

	wg.Wait()

	gpu.NVIDIA = nvidiaResult.GPUs
	gpu.NVIDIASMISchema = nvidiaResult.Schema
	gpu.AMD = amdGPUs
	gpu.DRM = drmDevices

	for _, err := range []error{nvidiaErr, amdErr, drmErr} {
		if err == nil {
			continue
		}

		gpu.Errors = append(gpu.Errors, oneLine(err.Error()))
	}

	return gpu, nil
}

// oneLine flattens a multi-line message so each collected error stays a single
// entry.
func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
