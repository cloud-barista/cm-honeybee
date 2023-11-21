//go:build !linux
// +build !linux

package nvidia

import (
	"context"
	"errors"
)

var ERR_GPU_NOT_SUPPORTED = errors.New("GPU monitoring is supported only on Linux (with the NVML toolchain installed)")

type GPUReader struct {
}

func (*GPUReader) Init() error {
	return ERR_GPU_NOT_SUPPORTED
}

func (*GPUReader) GPUStats(ctx context.Context) (*AllGPUStats, error) {
	return nil, ERR_GPU_NOT_SUPPORTED
}

func (*GPUReader) Shutdown() error {
	return ERR_GPU_NOT_SUPPORTED
}
