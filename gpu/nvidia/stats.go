package nvidia

type DeviceAttributes struct {
	GPUUUID                string   `json:"gpu_uuid"`
	DriverVersion          string   `json:"driver_version"`
	CUDAVersion            int      `json:"cuda_version"`
	VGPUDriverCapabilities []string `json:"vgpu_driver_capabilities"`
	ProductName            string   `json:"product_name"`
	ProductBrand           string   `json:"product_brand"`
	ProductArchitecture    string   `json:"product_architecture"`
	SerialNumber           string   `json:"serial_number"`
}

type Performance struct {
	GPUUsage        uint32 `json:"gpu_usage"`         // percent
	MemoryUsed      uint64 `json:"memory_used"`       // mb
	MemoryTotal     uint64 `json:"memory_total"`      // mb
	MemoryUsage     uint32 `json:"memory_usage"`      // percent
	Bar1MemoryUsed  uint64 `json:"bar1_memory_used"`  // mb
	Bar1MemoryTotal uint64 `json:"bar1_memory_total"` // mb
	Bar1MemoryUsage uint32 `json:"bar1_memory_usage"` // percent
}

type NVIDIA struct {
	DeviceAttributes DeviceAttributes `json:"device_attributes"`
	Performance      Performance      `json:"performance"`
}
