package infra

// NVIDIADeviceAttribute holds the static identity of a single NVIDIA GPU.
// Optional strings are empty when nvidia-smi reported them as absent, "N/A"
// or "Not Supported", so a missing reading is never mistaken for a real value.
type NVIDIADeviceAttribute struct {
	GPUUUID             string `json:"gpu_uuid"`
	DriverVersion       string `json:"driver_version"`
	CUDAVersion         string `json:"cuda_version"`
	ProductName         string `json:"product_name"`
	ProductBrand        string `json:"product_brand"`
	ProductArchitecture string `json:"product_architecture"`
	NVMLVersion         string `json:"nvml_version,omitempty"`
	Index               int    `json:"index"`
	MinorNumber         string `json:"minor_number,omitempty"`
	Serial              string `json:"serial,omitempty"`
	PCIBusID            string `json:"pci_bus_id,omitempty"`
	VBIOSVersion        string `json:"vbios_version,omitempty"`
	ComputeMode         string `json:"compute_mode,omitempty"`
	PersistenceMode     string `json:"persistence_mode,omitempty"`
	MIGMode             string `json:"mig_mode,omitempty"`
	VirtualizationMode  string `json:"virtualization_mode,omitempty"`
	HostVGPUMode        string `json:"host_vgpu_mode,omitempty"`
	VGPULicenseStatus   string `json:"vgpu_license_status,omitempty"`
}

// NVIDIAPerformance holds the live readings of a single NVIDIA GPU.
// Every numeric member is a pointer: nil means nvidia-smi did not report the
// value on this GPU, which is a different statement from the value being zero.
type NVIDIAPerformance struct {
	GPUUsage          *uint32  `json:"gpu_usage,omitempty"`          // percent
	MemoryUsage       *uint32  `json:"memory_usage,omitempty"`       // percent
	EncoderUsage      *uint32  `json:"encoder_usage,omitempty"`      // percent
	DecoderUsage      *uint32  `json:"decoder_usage,omitempty"`      // percent
	FBMemoryUsed      *uint64  `json:"fb_memory_used,omitempty"`     // MiB
	FBMemoryTotal     *uint64  `json:"fb_memory_total,omitempty"`    // MiB
	FBMemoryFree      *uint64  `json:"fb_memory_free,omitempty"`     // MiB
	FBMemoryReserved  *uint64  `json:"fb_memory_reserved,omitempty"` // MiB
	FBMemoryUsage     *uint32  `json:"fb_memory_usage,omitempty"`    // percent
	Bar1MemoryUsed    *uint64  `json:"bar1_memory_used,omitempty"`   // MiB
	Bar1MemoryTotal   *uint64  `json:"bar1_memory_total,omitempty"`  // MiB
	Bar1MemoryFree    *uint64  `json:"bar1_memory_free,omitempty"`   // MiB
	Bar1MemoryUsage   *uint32  `json:"bar1_memory_usage,omitempty"`  // percent
	PerformanceState  string   `json:"performance_state,omitempty"`
	FanSpeed          *uint32  `json:"fan_speed,omitempty"`          // percent
	TemperatureGPU    *int32   `json:"temperature_gpu,omitempty"`    // celsius
	TemperatureMemory *int32   `json:"temperature_memory,omitempty"` // celsius
	PowerDraw         *float64 `json:"power_draw,omitempty"`         // watt
	PowerLimit        *float64 `json:"power_limit,omitempty"`        // watt
	ClockGraphics     *uint32  `json:"clock_graphics,omitempty"`     // MHz
	ClockSM           *uint32  `json:"clock_sm,omitempty"`           // MHz
	ClockMemory       *uint32  `json:"clock_memory,omitempty"`       // MHz
	ClockVideo        *uint32  `json:"clock_video,omitempty"`        // MHz
	MaxClockGraphics  *uint32  `json:"max_clock_graphics,omitempty"` // MHz
	MaxClockSM        *uint32  `json:"max_clock_sm,omitempty"`       // MHz
	MaxClockMemory    *uint32  `json:"max_clock_memory,omitempty"`   // MHz
	PCIeLinkGen       *uint32  `json:"pcie_link_gen_current,omitempty"`
	PCIeLinkWidth     *uint32  `json:"pcie_link_width_current,omitempty"` // lanes
	PCIeReplayCounter *uint64  `json:"pcie_replay_counter,omitempty"`
	// ClocksEventReasons is the NVML clocks-event (throttle) reason bitmask,
	// folded from the per-reason Active/Not Active strings. Zero means "reported,
	// nothing active"; nil means nvidia-smi reported no reason at all.
	ClocksEventReasons *uint64 `json:"clocks_event_reasons,omitempty"`
}

// NVIDIAECCErrors holds one ECC counter block (volatile or aggregate).
// SingleBit and DoubleBit only appear on the pre-Volta schema, which counts
// errors by bit width instead of by SRAM/DRAM location.
type NVIDIAECCErrors struct {
	SRAMCorrectable   *uint64 `json:"sram_correctable,omitempty"`
	SRAMUncorrectable *uint64 `json:"sram_uncorrectable,omitempty"`
	DRAMCorrectable   *uint64 `json:"dram_correctable,omitempty"`
	DRAMUncorrectable *uint64 `json:"dram_uncorrectable,omitempty"`
	SingleBit         *uint64 `json:"single_bit,omitempty"`
	DoubleBit         *uint64 `json:"double_bit,omitempty"`
}

// NVIDIAECC holds the ECC mode and error counters of a single NVIDIA GPU.
type NVIDIAECC struct {
	Mode      string          `json:"mode,omitempty"`
	Volatile  NVIDIAECCErrors `json:"volatile"`
	Aggregate NVIDIAECCErrors `json:"aggregate"`
}

// NVIDIAProcess is one process holding memory on a GPU.
type NVIDIAProcess struct {
	PID           uint32  `json:"pid"`
	Type          string  `json:"type,omitempty"`
	Name          string  `json:"name,omitempty"`
	UsedMemory    *uint64 `json:"used_memory,omitempty"` // MiB
	GPUInstanceID string  `json:"gpu_instance_id,omitempty"`
}

// NVIDIAMIGDevice is one MIG instance carved out of a physical GPU.
type NVIDIAMIGDevice struct {
	Index             string  `json:"index"`
	GPUInstanceID     string  `json:"gpu_instance_id,omitempty"`
	ComputeInstanceID string  `json:"compute_instance_id,omitempty"`
	UUID              string  `json:"uuid,omitempty"`
	FBMemoryTotal     *uint64 `json:"fb_memory_total,omitempty"` // MiB
	FBMemoryUsed      *uint64 `json:"fb_memory_used,omitempty"`  // MiB
	FBMemoryFree      *uint64 `json:"fb_memory_free,omitempty"`  // MiB
}

type NVIDIA struct {
	DeviceAttribute NVIDIADeviceAttribute `json:"device_attribute"`
	Performance     NVIDIAPerformance     `json:"performance"`
	ECC             *NVIDIAECC            `json:"ecc,omitempty"`
	MIGDevices      []NVIDIAMIGDevice     `json:"mig_devices,omitempty"`
	Processes       []NVIDIAProcess       `json:"processes,omitempty"`
}

// AMDDeviceAttribute holds the static identity of a single AMD GPU as
// reported by rocm-smi.
type AMDDeviceAttribute struct {
	Card             string `json:"card"`
	GPUUUID          string `json:"gpu_uuid,omitempty"`
	GPUID            string `json:"gpu_id,omitempty"`
	UniqueID         string `json:"unique_id,omitempty"`
	ProductName      string `json:"product_name,omitempty"`
	SerialNumber     string `json:"serial_number,omitempty"`
	PCIBusID         string `json:"pci_bus_id,omitempty"`
	VBIOSVersion     string `json:"vbios_version,omitempty"`
	DriverVersion    string `json:"driver_version,omitempty"`
	VendorID         string `json:"vendor_id,omitempty"`
	DeviceID         string `json:"device_id,omitempty"`
	ComputePartition string `json:"compute_partition,omitempty"`
	MemoryPartition  string `json:"memory_partition,omitempty"`
}

// AMDPerformance holds the live readings of a single AMD GPU. Pointers carry
// the same "not reported" meaning as in NVIDIAPerformance.
type AMDPerformance struct {
	GPUUsage          *uint32  `json:"gpu_usage,omitempty"`         // percent
	MemoryUsage       *uint32  `json:"memory_usage,omitempty"`      // percent
	VRAMMemoryUsed    *uint64  `json:"vram_memory_used,omitempty"`  // MiB
	VRAMMemoryTotal   *uint64  `json:"vram_memory_total,omitempty"` // MiB
	PerformanceLevel  string   `json:"performance_level,omitempty"`
	FanSpeed          *uint32  `json:"fan_speed,omitempty"`          // percent
	TemperatureGPU    *float64 `json:"temperature_gpu,omitempty"`    // celsius
	TemperatureMemory *float64 `json:"temperature_memory,omitempty"` // celsius
	PowerDraw         *float64 `json:"power_draw,omitempty"`         // watt
	PowerCap          *float64 `json:"power_cap,omitempty"`          // watt
	ClockSM           *uint32  `json:"clock_sm,omitempty"`           // MHz
	ClockMemory       *uint32  `json:"clock_memory,omitempty"`       // MHz
}

type AMD struct {
	DeviceAttribute AMDDeviceAttribute `json:"device_attribute"`
	Performance     AMDPerformance     `json:"performance"`
}

type DRM struct {
	Card              string `json:"card,omitempty"`
	PCIBusID          string `json:"pci_bus_id,omitempty"`
	DriverName        string `json:"driver_name"`
	DriverVersion     string `json:"driver_version"`
	DriverDate        string `json:"driver_date"`
	DriverDescription string `json:"driver_description"`
}

type GPU struct {
	NVIDIA []NVIDIA `json:"nvidia"`
	AMD    []AMD    `json:"amd"`
	DRM    []DRM    `json:"drm"`
	// NVIDIASMISchema is the nvidia-smi XML schema version found in the output
	// (v11, v12, v13 ...). A version with no parser of its own is read with the
	// nearest one and reported as "v9 (read as v11)", so a substituted reading
	// is never presented as an understood one. Empty when nvidia-smi did not run.
	NVIDIASMISchema string   `json:"nvidia_smi_schema,omitempty"`
	Errors          []string `json:"errors"`
}
