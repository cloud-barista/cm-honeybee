//go:build linux
// +build linux

package nvidia

import (
	"errors"
	"fmt"
	"github.com/NVIDIA/go-nvml/pkg/nvml"
	"github.com/jollaman999/utils/logger"
	"strings"
)

type NVReader struct {
}

func (r *NVReader) Init() error {
	result := nvml.Init()
	if result != nvml.SUCCESS {
		err := fmt.Sprintf("NVIDIA: Unable to initialize NVML: %v", nvml.ErrorString(result))
		logger.Println(logger.ERROR, true, err)
		return errors.New(err)
	}

	return nil
}

func (r *NVReader) GPUStats() ([]NVIDIA, error) {
	result := nvml.Init()
	if result != nvml.SUCCESS {
		err := fmt.Sprintf("NVIDIA: Unable to initialize NVML: %v", nvml.ErrorString(result))
		logger.Println(logger.ERROR, true, err)
		return nil, errors.New(err)
	}

	count, result := nvml.DeviceGetCount()
	if result != nvml.SUCCESS {
		err := fmt.Sprintf("NVIDIA: Unable to get device count: %v", nvml.ErrorString(result))
		logger.Println(logger.ERROR, true, err)
		return nil, errors.New(err)
	}

	logger.Println(logger.DEBUG, false, fmt.Sprintf("NVIDIA: Found %v GPU device(s)", count))

	var nvStats []NVIDIA

	driverVersion, result := nvml.SystemGetDriverVersion()
	if result != nvml.SUCCESS {
		err := "NVIDIA: Unable to get driver version"
		logger.Println(logger.ERROR, true, err)
		return nil, errors.New(err)
	}

	cudaVersion, result := nvml.SystemGetCudaDriverVersion()
	if result != nvml.SUCCESS {
		err := "NVIDIA: Unable to get CUDA version"
		logger.Println(logger.ERROR, true, err)
		return nil, errors.New(err)
	}

	for i := 0; i < count; i++ {
		device, result := nvml.DeviceGetHandleByIndex(i)
		if result != nvml.SUCCESS {
			err := fmt.Sprintf("NVIDIA: Unable to get device at index %d: %v", i, nvml.ErrorString(result))
			logger.Println(logger.ERROR, true, err)
			return nil, errors.New(err)
		}

		gpuUUID, result := device.GetUUID()
		if result != nvml.SUCCESS {
			err := fmt.Sprintf("NVIDIA: Unable to get uuid of device at index %d: %v", i, nvml.ErrorString(result))
			logger.Println(logger.ERROR, true, err)
			return nil, errors.New(err)
		}
		gpuUUID = strings.Replace(gpuUUID, "GPU-", "", -1)
		logger.Println(logger.DEBUG, false, fmt.Sprintf("NVIDIA: GPU no %v - %v", i, gpuUUID))

		var vGPUDriverCapabilities []string
		capOK, _ := device.GetVgpuCapabilities(nvml.DEVICE_VGPU_CAP_FRACTIONAL_MULTI_VGPU)
		if capOK {
			vGPUDriverCapabilities = append(vGPUDriverCapabilities, "DEVICE_VGPU_CAP_FRACTIONAL_MULTI_VGPU")
		}
		capOK, _ = device.GetVgpuCapabilities(nvml.DEVICE_VGPU_CAP_HETEROGENEOUS_TIMESLICE_PROFILES)
		if capOK {
			vGPUDriverCapabilities = append(vGPUDriverCapabilities, "DEVICE_VGPU_CAP_HETEROGENEOUS_TIMESLICE_PROFILES")
		}
		capOK, _ = device.GetVgpuCapabilities(nvml.DEVICE_VGPU_CAP_HETEROGENEOUS_TIMESLICE_SIZES)
		if capOK {
			vGPUDriverCapabilities = append(vGPUDriverCapabilities, "DEVICE_VGPU_CAP_HETEROGENEOUS_TIMESLICE_SIZES")
		}

		productName, result := device.GetName()
		if result != nvml.SUCCESS {
			err := fmt.Sprintf("NVIDIA: Unable to get GPU name of device %v: %v", gpuUUID, nvml.ErrorString(result))
			logger.Println(logger.ERROR, true, err)
			return nil, errors.New(err)
		}

		productBrandCode, result := device.GetBrand()
		if result != nvml.SUCCESS {
			err := fmt.Sprintf("NVIDIA: Unable to get GPU brand of device %v: %v", gpuUUID, nvml.ErrorString(result))
			logger.Println(logger.ERROR, true, err)
			return nil, errors.New(err)
		}
		var productBrand string
		switch productBrandCode {
		case nvml.BRAND_UNKNOWN:
			productBrand = "Unknown"
		case nvml.BRAND_QUADRO:
			productBrand = "Quadro"
		case nvml.BRAND_TESLA:
			productBrand = "Tesla"
		case nvml.BRAND_NVS:
			productBrand = "NVS"
		case nvml.BRAND_GRID:
			productBrand = "GRID"
		case nvml.BRAND_GEFORCE:
			productBrand = "GeForce"
		case nvml.BRAND_TITAN:
			productBrand = "TITAN"
		case nvml.BRAND_NVIDIA_VAPPS:
			productBrand = "vApps"
		case nvml.BRAND_NVIDIA_VPC:
			productBrand = "vPC"
		case nvml.BRAND_NVIDIA_VCS:
			productBrand = "vCS"
		case nvml.BRAND_NVIDIA_VWS:
			productBrand = "vWS"
		case nvml.BRAND_NVIDIA_VGAMING: // Same with BRAND_NVIDIA_CLOUD_GAMING
			productBrand = "vGaming"
		case nvml.BRAND_QUADRO_RTX:
			productBrand = "Quadro RTX"
		case nvml.BRAND_NVIDIA_RTX:
			productBrand = "NVIDIA RTX"
		case nvml.BRAND_NVIDIA:
			productBrand = "NVIDIA"
		case nvml.BRAND_GEFORCE_RTX:
			productBrand = "GeForce RTX"
		case nvml.BRAND_TITAN_RTX:
			productBrand = "TITAN RTX"
		}

		productArchitectureCode, result := device.GetArchitecture()
		if result != nvml.SUCCESS {
			err := fmt.Sprintf("NVIDIA: Unable to get GPU architecture of device %v: %v", gpuUUID, nvml.ErrorString(result))
			logger.Println(logger.ERROR, true, err)
			return nil, errors.New(err)
		}
		var productArchitecture string
		switch productArchitectureCode {
		case nvml.DEVICE_ARCH_UNKNOWN:
			productArchitecture = "Unknown"
		case nvml.DEVICE_ARCH_KEPLER:
			productArchitecture = "Kepler"
		case nvml.DEVICE_ARCH_MAXWELL:
			productArchitecture = "Maxwell"
		case nvml.DEVICE_ARCH_PASCAL:
			productArchitecture = "Pascal"
		case nvml.DEVICE_ARCH_VOLTA:
			productArchitecture = "Volta"
		case nvml.DEVICE_ARCH_TURING:
			productArchitecture = "Turing"
		case nvml.DEVICE_ARCH_AMPERE:
			productArchitecture = "Ampere"
		case nvml.DEVICE_ARCH_ADA:
			productArchitecture = "Ada"
		case nvml.DEVICE_ARCH_HOPPER:
			productArchitecture = "Hopper"
		}

		serialNumber, _ := device.GetSerial()

		utilization, result := device.GetUtilizationRates()
		if result != nvml.SUCCESS {
			err := fmt.Sprintf("NVIDIA: Unable to get GPU utilization of device %v: %v", gpuUUID, nvml.ErrorString(result))
			logger.Println(logger.ERROR, true, err)
			return nil, errors.New(err)
		}
		gpuUsage := utilization.Gpu

		memory, result := device.GetMemoryInfo()
		if result != nvml.SUCCESS {
			err := fmt.Sprintf("NVIDIA: Unable to get memory of device %v: %v", gpuUUID, nvml.ErrorString(result))
			logger.Println(logger.ERROR, true, err)
			return nil, errors.New(err)
		}
		memoryUsed := memory.Used / 1024 / 1024
		memoryTotal := memory.Total / 1024 / 1024
		memoryUsage := float32(memoryUsed) / float32(memoryTotal) * 100

		bar1memory, result := device.GetBAR1MemoryInfo()
		if result != nvml.SUCCESS {
			err := fmt.Sprintf("NVIDIA: Unable to get memory of device %v: %v", gpuUUID, nvml.ErrorString(result))
			logger.Println(logger.ERROR, true, err)
			return nil, errors.New(err)
		}
		bar1memoryUsed := bar1memory.Bar1Used / 1024 / 1024
		bar1memoryTotal := bar1memory.Bar1Total / 1024 / 1024
		bar1memoryUsage := float32(bar1memoryUsed) / float32(bar1memoryTotal) * 100

		stat := NVIDIA{
			DeviceAttributes: DeviceAttributes{
				GPUUUID:                gpuUUID,
				DriverVersion:          driverVersion,
				CUDAVersion:            cudaVersion,
				VGPUDriverCapabilities: vGPUDriverCapabilities,
				ProductName:            productName,
				ProductBrand:           productBrand,
				ProductArchitecture:    productArchitecture,
				SerialNumber:           serialNumber,
			},
			Performance: Performance{
				GPUUsage:        gpuUsage,
				MemoryUsed:      memoryUsed,
				MemoryTotal:     memoryTotal,
				MemoryUsage:     uint32(memoryUsage),
				Bar1MemoryUsed:  bar1memoryUsed,
				Bar1MemoryTotal: bar1memoryTotal,
				Bar1MemoryUsage: uint32(bar1memoryUsage),
			},
		}

		nvStats = append(nvStats, stat)
	}

	return nvStats, nil
}

func (r *NVReader) GetGPUCount() (int, error) {
	count, result := nvml.DeviceGetCount()
	if result != nvml.SUCCESS {
		err := fmt.Sprintf("NVIDIA: Unable to get device count: %v", nvml.ErrorString(result))
		logger.Println(logger.ERROR, true, err)
		return 0, errors.New(err)
	}

	return count, nil
}

func (r *NVReader) Shutdown() error {
	result := nvml.Shutdown()
	if result != nvml.SUCCESS {
		err := fmt.Sprintf("NVIDIA: Unable to shutdown NVML: %v", nvml.ErrorString(result))
		logger.Println(logger.ERROR, true, err)
		return errors.New(err)
	}

	return nil
}
