package nvidia

import (
	"encoding/xml"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/infra"
	"github.com/jollaman999/utils/logger"
	"strconv"
	"strings"
)

func QueryGPU() ([]infra.NVIDIA, error) {
	if !isNVIDIASmiAvailable() {
		errMsg := "NVIDIA: nvidia-smi command is not available"
		logger.Println(logger.DEBUG, false, errMsg)

		return []infra.NVIDIA{}, nil
	}

	output, err := runNVIDIASmi("-q -x")
	if err != nil {
		return []infra.NVIDIA{}, err
	}

	var nvidiaSMILog SmiLog

	err = xml.Unmarshal([]byte(output), &nvidiaSMILog)
	if err != nil {
		return []infra.NVIDIA{}, err
	}

	var nvidia []infra.NVIDIA

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

		nv := infra.NVIDIA{
			DeviceAttribute: infra.NVIDIADeviceAttribute{
				GPUUUID:             gpu.UUID,
				DriverVersion:       nvidiaSMILog.DriverVersion,
				CUDAVersion:         nvidiaSMILog.CudaVersion,
				ProductName:         gpu.ProductName,
				ProductBrand:        gpu.ProductBrand,
				ProductArchitecture: gpu.ProductArchitecture,
			},
			Performance: infra.NVIDIAPerformance{
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
