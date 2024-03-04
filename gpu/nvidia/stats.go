package nvidia

import (
	"encoding/xml"
	"github.com/jollaman999/utils/logger"
	"strconv"
	"strings"
)

type DeviceAttribute struct {
	GPUUUID             string `json:"gpu_uuid"`
	DriverVersion       string `json:"driver_version"`
	CUDAVersion         string `json:"cuda_version"`
	ProductName         string `json:"product_name"`
	ProductBrand        string `json:"product_brand"`
	ProductArchitecture string `json:"product_architecture"`
}

type Performance struct {
	GPUUsage        uint32 `json:"gpu_usage"`         // percent
	FBMemoryUsed    uint64 `json:"fb_memory_used"`    // mb
	FBMemoryTotal   uint64 `json:"fb_memory_total"`   // mb
	FBMemoryUsage   uint32 `json:"fb_memory_usage"`   // percent
	Bar1MemoryUsed  uint64 `json:"bar1_memory_used"`  // mb
	Bar1MemoryTotal uint64 `json:"bar1_memory_total"` // mb
	Bar1MemoryUsage uint32 `json:"bar1_memory_usage"` // percent
}

type NVIDIA struct {
	DeviceAttribute DeviceAttribute `json:"device_attribute"`
	Performance     Performance     `json:"performance"`
}

func QueryGPU() ([]NVIDIA, error) {
	if !isNVIDIASmiAvailable() {
		errMsg := "NVIDIA: nvidia-smi command is not available"
		logger.Println(logger.DEBUG, false, errMsg)

		return []NVIDIA{}, nil
	}

	output, err := runNVIDIASmi("-q -x")
	if err != nil {
		return []NVIDIA{}, err
	}

	var nvidiaSMILog SmiLog

	err = xml.Unmarshal([]byte(output), &nvidiaSMILog)
	if err != nil {
		return []NVIDIA{}, err
	}

	var nvidia []NVIDIA

	for _, gpu := range nvidiaSMILog.Gpu {
		gpuUsage, _ := strconv.Atoi(strings.Replace(strings.Replace(strings.ToLower(gpu.Utilization.GpuUtil),
			"%", "", -1), " ", "", -1))

		fbMemoryUsed, _ := strconv.Atoi(strings.Replace(strings.Replace(strings.ToLower(gpu.FbMemoryUsage.Used),
			"mib", "", -1), " ", "", -1))
		fbMemoryTotal, _ := strconv.Atoi(strings.Replace(strings.Replace(strings.ToLower(gpu.FbMemoryUsage.Total),
			"mib", "", -1), " ", "", -1))
		fBMemoryUsage := float32(fbMemoryUsed) / float32(fbMemoryTotal) * 100

		bar1MemoryUsed, _ := strconv.Atoi(strings.Replace(strings.Replace(strings.ToLower(gpu.Bar1MemoryUsage.Used),
			"mib", "", -1), " ", "", -1))
		bar1MemoryTotal, _ := strconv.Atoi(strings.Replace(strings.Replace(strings.ToLower(gpu.Bar1MemoryUsage.Total),
			"mib", "", -1), " ", "", -1))
		bar1MemoryUsage := float32(bar1MemoryUsed) / float32(bar1MemoryTotal) * 100

		nv := NVIDIA{
			DeviceAttribute: DeviceAttribute{
				GPUUUID:             gpu.UUID,
				DriverVersion:       nvidiaSMILog.DriverVersion,
				CUDAVersion:         nvidiaSMILog.CudaVersion,
				ProductName:         gpu.ProductName,
				ProductBrand:        gpu.ProductBrand,
				ProductArchitecture: gpu.ProductArchitecture,
			},
			Performance: Performance{
				GPUUsage:        uint32(gpuUsage),
				FBMemoryUsed:    uint64(fbMemoryUsed),
				FBMemoryTotal:   uint64(fbMemoryTotal),
				FBMemoryUsage:   uint32(fBMemoryUsage),
				Bar1MemoryUsed:  uint64(bar1MemoryUsed),
				Bar1MemoryTotal: uint64(bar1MemoryTotal),
				Bar1MemoryUsage: uint32(bar1MemoryUsage),
			},
		}

		nvidia = append(nvidia, nv)
	}

	return nvidia, nil
}
