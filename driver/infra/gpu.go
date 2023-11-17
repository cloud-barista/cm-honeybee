package infra

import (
	"fmt"
	"log"

	"github.com/NVIDIA/go-nvml/pkg/nvml"
)

func GetGpuInfo() {
	ret := nvml.Init()
	if ret != nvml.SUCCESS {
		log.Fatalf("Unable to initialize NVML: %v", nvml.ErrorString(ret))
	}
	defer func() {
		ret := nvml.Shutdown()
		if ret != nvml.SUCCESS {
			log.Fatalf("Unable to shutdown NVML: %v", nvml.ErrorString(ret))
		}
	}()

	count, ret := nvml.DeviceGetCount()
	if ret != nvml.SUCCESS {
		log.Fatalf("Unable to get device count: %v", nvml.ErrorString(ret))
	}

	for i := 0; i < count; i++ {
		device, ret := nvml.DeviceGetHandleByIndex(i)
		if ret != nvml.SUCCESS {
			log.Fatalf("Unable to get device at index %d: %v", i, nvml.ErrorString(ret))
		}

		uuid, ret := device.GetUUID()
		if ret != nvml.SUCCESS {
			log.Fatalf("Unable to get uuid of device at index %d: %v", i, nvml.ErrorString(ret))
		}
		fmt.Printf("%v\n", uuid)
		name, ret := device.GetName()
		if ret != nvml.SUCCESS {
			log.Fatalf("Unable to get name of device at index %d: %v", i, nvml.ErrorString(ret))
		}
		fmt.Printf("%v\n", name)
		GetMemoryInfo, ret := device.GetMemoryInfo()
		if ret != nvml.SUCCESS {
			log.Fatalf("Unable to get GetMemoryInfo of device at index %d: %v", i, nvml.ErrorString(ret))
		}
		fmt.Printf("GetMemoryInfo %v\n", GetMemoryInfo)
		GetMemoryInfo_v2, ret := device.GetMemoryInfo_v2()
		if ret != nvml.SUCCESS {
			log.Fatalf("Unable to get GetMemoryInfo_v2 of device at index %d: %v", i, nvml.ErrorString(ret))
		}
		fmt.Printf("GetMemoryInfo_v2 %v  \n", GetMemoryInfo_v2)
		GetIndex, ret := device.GetIndex()
		if ret != nvml.SUCCESS {
			log.Fatalf("Unable to get GetIndex of device at index %d: %v", i, nvml.ErrorString(ret))
		}
		fmt.Printf("GetIndex %v\n", GetIndex)
		GetDisplayActive, ret := device.GetDisplayActive()
		if ret != nvml.SUCCESS {
			log.Fatalf("Unable to get GetDisplayActive of device at index %d: %v", i, nvml.ErrorString(ret))
		}
		fmt.Printf("GetDisplayActive %v\n", GetDisplayActive)
		GetBrand, ret := device.GetBrand()
		if ret != nvml.SUCCESS {
			log.Fatalf("Unable to get GetBrand of device at index %d: %v", i, nvml.ErrorString(ret))
		}
		fmt.Printf("GetBrand %v\n", GetBrand)
		GetArchitecture, ret := device.GetArchitecture()
		if ret != nvml.SUCCESS {
			log.Fatalf("Unable to get GetArchitecture of device at index %d: %v", i, nvml.ErrorString(ret))
		}
		fmt.Printf("GetArchitecture %v\n", GetArchitecture)
		GetDriverModel, _, ret := device.GetDriverModel()
		if ret != nvml.SUCCESS {
			log.Fatalf("Unable to get GetDriverModel of device at index %d: %v", i, nvml.ErrorString(ret))
		}
		fmt.Printf("GetDriverModel %v  \n", GetDriverModel)

	}

}
