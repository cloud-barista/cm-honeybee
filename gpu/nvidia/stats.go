package nvidia

import (
	"encoding/xml"
	"errors"
	"github.com/jollaman999/utils/logger"
	"strconv"
	"strings"
)

type DeviceAttributes struct {
	GPUUUID             string `json:"gpu_uuid"`
	DriverVersion       string `json:"driver_version"`
	CUDAVersion         string `json:"cuda_version"`
	ProductName         string `json:"product_name"`
	ProductBrand        string `json:"product_brand"`
	ProductArchitecture string `json:"product_architecture"`
	SerialNumber        string `json:"serial_number"`
}

type Performance struct {
	GPUUsage        string `json:"gpu_usage"`         // percent
	FBMemoryUsed    string `json:"fb_memory_used"`    // mb
	FBMemoryTotal   string `json:"fb_memory_total"`   // mb
	FBMemoryUsage   string `json:"fb_memory_usage"`   // percent
	Bar1MemoryUsed  string `json:"bar1_memory_used"`  // mb
	Bar1MemoryTotal string `json:"bar1_memory_total"` // mb
	Bar1MemoryUsage string `json:"bar1_memory_usage"` // percent
}

type NVIDIA struct {
	DeviceAttributes DeviceAttributes `json:"device_attributes"`
	Performance      Performance      `json:"performance"`
}

func QueryGPU() ([]NVIDIA, error) {
	if !isNVIDIASmiAvailable() {
		errMsg := "NVIDIA: nvidia-smi command is not available"
		logger.Println(logger.DEBUG, false, errMsg)

		return []NVIDIA{}, errors.New(errMsg)
	}

	var args []string

	args = append(args, "-q")
	args = append(args, "-x")

	output, err := runNVIDIASmi(args)
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
		fbMemoryUsed, _ := strconv.Atoi(strings.Replace(strings.Replace(strings.ToLower(gpu.FbMemoryUsage.Used),
			"mib", "", -1), " ", "", -1))
		fbMemoryTotal, _ := strconv.Atoi(strings.Replace(strings.Replace(strings.ToLower(gpu.FbMemoryUsage.Total),
			"mib", "", -1), " ", "", -1))
		fBMemoryUsage := strconv.Itoa(int(float32(fbMemoryUsed)/float32(fbMemoryTotal)*100)) + " %"

		bar1MemoryUsed, _ := strconv.Atoi(strings.Replace(strings.Replace(strings.ToLower(gpu.Bar1MemoryUsage.Used),
			"mib", "", -1), " ", "", -1))
		bar1MemoryTotal, _ := strconv.Atoi(strings.Replace(strings.Replace(strings.ToLower(gpu.Bar1MemoryUsage.Total),
			"mib", "", -1), " ", "", -1))
		bar1MemoryUsage := strconv.Itoa(int(float32(bar1MemoryUsed)/float32(bar1MemoryTotal)*100)) + " %"

		nv := NVIDIA{
			DeviceAttributes: DeviceAttributes{
				GPUUUID:             gpu.UUID,
				DriverVersion:       nvidiaSMILog.DriverVersion,
				CUDAVersion:         nvidiaSMILog.CudaVersion,
				ProductName:         gpu.ProductName,
				ProductBrand:        gpu.ProductBrand,
				ProductArchitecture: gpu.ProductArchitecture,
				SerialNumber:        gpu.Serial,
			},
			Performance: Performance{
				GPUUsage:        gpu.Utilization.GpuUtil,
				FBMemoryUsed:    gpu.FbMemoryUsage.Used,
				FBMemoryTotal:   gpu.FbMemoryUsage.Total,
				FBMemoryUsage:   fBMemoryUsage,
				Bar1MemoryUsed:  gpu.Bar1MemoryUsage.Used,
				Bar1MemoryTotal: gpu.Bar1MemoryUsage.Total,
				Bar1MemoryUsage: bar1MemoryUsage,
			},
		}

		nvidia = append(nvidia, nv)
	}

	return nvidia, nil
}
