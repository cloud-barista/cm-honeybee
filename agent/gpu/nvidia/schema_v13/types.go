// Package schema_v13 parses the nvsmi_device_v13.dtd flavour of
// `nvidia-smi -q -x`, introduced with the R560 driver series.
//
// Only the elements cm-honeybee reports are declared. Elements the v13 schema
// renamed away from v12 are kept alongside their new name where a driver in
// the same generation may still emit either one.
package schema_v13

import "encoding/xml"

type smiLog struct {
	XMLName xml.Name `xml:"nvidia_smi_log"`
	// DriverVersion and CudaVersion are deprecated in v13 and scheduled for
	// removal in CUDA 14.0; KmdVersion and CudaUmdVersion replace them.
	DriverVersion  string `xml:"driver_version"`
	CudaVersion    string `xml:"cuda_version"`
	KmdVersion     string `xml:"kmd_version"`
	CudaUmdVersion string `xml:"cuda_umd_version"`
	GPU            []gpu  `xml:"gpu"`
}

type memory struct {
	Total    string `xml:"total"`
	Reserved string `xml:"reserved"`
	Used     string `xml:"used"`
	Free     string `xml:"free"`
}

type eccCounts struct {
	SramCorrectable string `xml:"sram_correctable"`
	// v13 splits the uncorrectable SRAM counter by error kind.
	SramUncorrectable       string `xml:"sram_uncorrectable"`
	SramUncorrectableParity string `xml:"sram_uncorrectable_parity"`
	SramUncorrectableSecded string `xml:"sram_uncorrectable_secded"`
	DramCorrectable         string `xml:"dram_correctable"`
	DramUncorrectable       string `xml:"dram_uncorrectable"`
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

	// v13 replaces power_draw with average_power_draw plus instant_power_draw.
	GpuPowerReadings struct {
		PowerDraw         string `xml:"power_draw"`
		AveragePowerDraw  string `xml:"average_power_draw"`
		InstantPowerDraw  string `xml:"instant_power_draw"`
		CurrentPowerLimit string `xml:"current_power_limit"`
		PowerLimit        string `xml:"power_limit"`
	} `xml:"gpu_power_readings"`

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

	ClocksEventReasons struct {
		GpuIdle                   string `xml:"clocks_event_reason_gpu_idle"`
		ApplicationsClocksSetting string `xml:"clocks_event_reason_applications_clocks_setting"`
		SwPowerCap                string `xml:"clocks_event_reason_sw_power_cap"`
		HwSlowdown                string `xml:"clocks_event_reason_hw_slowdown"`
		HwThermalSlowdown         string `xml:"clocks_event_reason_hw_thermal_slowdown"`
		HwPowerBrakeSlowdown      string `xml:"clocks_event_reason_hw_power_brake_slowdown"`
		SyncBoost                 string `xml:"clocks_event_reason_sync_boost"`
		SwThermalSlowdown         string `xml:"clocks_event_reason_sw_thermal_slowdown"`
		DisplayClocksSetting      string `xml:"clocks_event_reason_display_clocks_setting"`
	} `xml:"clocks_event_reasons"`

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
