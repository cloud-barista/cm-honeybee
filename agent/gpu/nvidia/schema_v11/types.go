// Package schema_v11 parses the nvsmi_device_v10.dtd and nvsmi_device_v11.dtd
// flavours of `nvidia-smi -q -x`, and is also the fallback for the older
// drivers that emit no DOCTYPE at all.
//
// This schema names the throttle block clocks_throttle_reasons and the power
// block power_readings; v12 renamed both. Pre-Volta drivers additionally count
// ECC errors by bit width (single_bit/double_bit) instead of by SRAM/DRAM
// location, so both shapes are declared here.
package schema_v11

import "encoding/xml"

type smiLog struct {
	XMLName       xml.Name `xml:"nvidia_smi_log"`
	DriverVersion string   `xml:"driver_version"`
	CudaVersion   string   `xml:"cuda_version"`
	GPU           []gpu    `xml:"gpu"`
}

type memory struct {
	Total    string `xml:"total"`
	Reserved string `xml:"reserved"`
	Used     string `xml:"used"`
	Free     string `xml:"free"`
}

type bitCounts struct {
	Total string `xml:"total"`
}

type eccCounts struct {
	SramCorrectable   string    `xml:"sram_correctable"`
	SramUncorrectable string    `xml:"sram_uncorrectable"`
	DramCorrectable   string    `xml:"dram_correctable"`
	DramUncorrectable string    `xml:"dram_uncorrectable"`
	SingleBit         bitCounts `xml:"single_bit"`
	DoubleBit         bitCounts `xml:"double_bit"`
}

type processInfo struct {
	GpuInstanceID string `xml:"gpu_instance_id"`
	Pid           string `xml:"pid"`
	Type          string `xml:"type"`
	ProcessName   string `xml:"process_name"`
	UsedMemory    string `xml:"used_memory"`
}

type migDevice struct {
	Index             string `xml:"index"`
	GpuInstanceID     string `xml:"gpu_instance_id"`
	ComputeInstanceID string `xml:"compute_instance_id"`
	UUID              string `xml:"uuid"`
	FbMemoryUsage     memory `xml:"fb_memory_usage"`
}

type gpu struct {
	ID                  string `xml:"id,attr"`
	ProductName         string `xml:"product_name"`
	ProductBrand        string `xml:"product_brand"`
	ProductArchitecture string `xml:"product_architecture"`
	Serial              string `xml:"serial"`
	UUID                string `xml:"uuid"`
	MinorNumber         string `xml:"minor_number"`
	VbiosVersion        string `xml:"vbios_version"`
	PersistenceMode     string `xml:"persistence_mode"`
	ComputeMode         string `xml:"compute_mode"`
	PerformanceState    string `xml:"performance_state"`
	FanSpeed            string `xml:"fan_speed"`

	MigMode struct {
		CurrentMig string `xml:"current_mig"`
	} `xml:"mig_mode"`

	MigDevices struct {
		MigDevice []migDevice `xml:"mig_device"`
	} `xml:"mig_devices"`

	GpuVirtualizationMode struct {
		VirtualizationMode string `xml:"virtualization_mode"`
		HostVgpuMode       string `xml:"host_vgpu_mode"`
	} `xml:"gpu_virtualization_mode"`

	VgpuSoftwareLicensedProduct struct {
		LicenseStatus string `xml:"license_status"`
	} `xml:"vgpu_software_licensed_product"`

	Pci struct {
		PciBusID       string `xml:"pci_bus_id"`
		PciGpuLinkInfo struct {
			PcieGen struct {
				CurrentLinkGen string `xml:"current_link_gen"`
			} `xml:"pcie_gen"`
			LinkWidths struct {
				CurrentLinkWidth string `xml:"current_link_width"`
			} `xml:"link_widths"`
		} `xml:"pci_gpu_link_info"`
		ReplayCounter string `xml:"replay_counter"`
	} `xml:"pci"`

	FbMemoryUsage   memory `xml:"fb_memory_usage"`
	Bar1MemoryUsage memory `xml:"bar1_memory_usage"`

	Utilization struct {
		GpuUtil     string `xml:"gpu_util"`
		MemoryUtil  string `xml:"memory_util"`
		EncoderUtil string `xml:"encoder_util"`
		DecoderUtil string `xml:"decoder_util"`
	} `xml:"utilization"`

	Temperature struct {
		GpuTemp    string `xml:"gpu_temp"`
		MemoryTemp string `xml:"memory_temp"`
	} `xml:"temperature"`

	PowerReadings struct {
		PowerDraw  string `xml:"power_draw"`
		PowerLimit string `xml:"power_limit"`
	} `xml:"power_readings"`

	Clocks struct {
		GraphicsClock string `xml:"graphics_clock"`
		SmClock       string `xml:"sm_clock"`
		MemClock      string `xml:"mem_clock"`
		VideoClock    string `xml:"video_clock"`
	} `xml:"clocks"`

	MaxClocks struct {
		GraphicsClock string `xml:"graphics_clock"`
		SmClock       string `xml:"sm_clock"`
		MemClock      string `xml:"mem_clock"`
	} `xml:"max_clocks"`

	ClocksThrottleReasons struct {
		GpuIdle                   string `xml:"clocks_throttle_reason_gpu_idle"`
		ApplicationsClocksSetting string `xml:"clocks_throttle_reason_applications_clocks_setting"`
		SwPowerCap                string `xml:"clocks_throttle_reason_sw_power_cap"`
		HwSlowdown                string `xml:"clocks_throttle_reason_hw_slowdown"`
		HwThermalSlowdown         string `xml:"clocks_throttle_reason_hw_thermal_slowdown"`
		HwPowerBrakeSlowdown      string `xml:"clocks_throttle_reason_hw_power_brake_slowdown"`
		SyncBoost                 string `xml:"clocks_throttle_reason_sync_boost"`
		SwThermalSlowdown         string `xml:"clocks_throttle_reason_sw_thermal_slowdown"`
		DisplayClocksSetting      string `xml:"clocks_throttle_reason_display_clocks_setting"`
	} `xml:"clocks_throttle_reasons"`

	EccMode struct {
		CurrentEcc string `xml:"current_ecc"`
	} `xml:"ecc_mode"`

	EccErrors struct {
		Volatile  eccCounts `xml:"volatile"`
		Aggregate eccCounts `xml:"aggregate"`
	} `xml:"ecc_errors"`

	Processes struct {
		ProcessInfo []processInfo `xml:"process_info"`
	} `xml:"processes"`
}
