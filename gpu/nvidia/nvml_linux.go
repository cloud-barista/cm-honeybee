//go:build linux
// +build linux

package nvidia

import (
	"fmt"
	"github.com/NVIDIA/go-nvml/pkg/nvml"
	"github.com/jollaman999/utils/logger"
)

type NVReader struct {
}

func (r *NVReader) Init() error {
	result := nvml.Init()

	if result != nvml.SUCCESS {
		return fmt.Errorf("Unable to initialize NVML: %v", nvml.ErrorString(result))
	}

	return nil
}

func (r *NVReader) GPUStats() (AllGPUStats, error) {
	result := nvml.Init()

	if result != nvml.SUCCESS {
		return nil, fmt.Errorf("Unable to initialize NVML: %v", nvml.ErrorString(result))
	}

	count, result := nvml.DeviceGetCount()

	if result != nvml.SUCCESS {
		return nil, fmt.Errorf("Unable to get device count: %v", nvml.ErrorString(result))
	}

	logger.Println(logger.DEBUG, false, fmt.Sprintf("Found %v GPU device(s)", count))

	stats := make(map[string]GPUStats, count)

	for i := 0; i < count; i++ {
		device, result := nvml.DeviceGetHandleByIndex(i)

		if result != nvml.SUCCESS {
			return nil, fmt.Errorf("Unable to get device at index %d: %v", i, nvml.ErrorString(result))
		}

		uuid, result := device.GetUUID()

		if result != nvml.SUCCESS {
			return nil, fmt.Errorf("Unable to get uuid of device at index %d: %v", i, nvml.ErrorString(result))
		}

		logger.Println(logger.DEBUG, false, fmt.Sprintf("GPU no %v - %v", i, uuid))

		utilization, result := device.GetUtilizationRates()

		if result != nvml.SUCCESS {
			return nil, fmt.Errorf("Unable to get GPU utilization of device %v: %v", uuid, nvml.ErrorString(result))
		}

		memory, result := device.GetMemoryInfo()

		if result != nvml.SUCCESS {
			return nil, fmt.Errorf("Unable to get memory of device %v: %v", uuid, nvml.ErrorString(result))
		}

		stats[uuid] = GPUStats{
			UsagePercentage:    utilization.Gpu,
			MemoryUsedInBytes:  memory.Used,
			TotalMemoryInBytes: memory.Total,
		}
	}

	return stats, nil
}

func (r *NVReader) GetGPUCount() (int, error) {
	count, result := nvml.DeviceGetCount()

	if result != nvml.SUCCESS {
		return 0, fmt.Errorf("Unable to get device count: %v", nvml.ErrorString(result))
	}

	return count, nil
}

func (r *NVReader) Shutdown() error {
	result := nvml.Shutdown()

	if result != nvml.SUCCESS {
		return fmt.Errorf("Unable to shutdown NVML: %v", nvml.ErrorString(result))
	}

	return nil
}
