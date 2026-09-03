package schema_v11

import (
	"encoding/xml"

	"github.com/cloud-barista/cm-honeybee/agent/gpu/nvidia/common"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/infra"
)

// Parse turns a v10 or v11 `nvidia-smi -q -x` document into the common GPU model.
func Parse(data []byte) ([]infra.NVIDIA, error) {
	var log smiLog

	if err := xml.Unmarshal(data, &log); err != nil {
		return nil, err
	}

	driverVersion := common.Str(log.DriverVersion)
	cudaVersion := common.Str(log.CudaVersion)

	gpus := make([]infra.NVIDIA, 0, len(log.GPU))

	for i, g := range log.GPU {
		fbUsed := common.Uint64(g.FbMemoryUsage.Used)
		fbTotal := common.Uint64(g.FbMemoryUsage.Total)
		bar1Used := common.Uint64(g.Bar1MemoryUsage.Used)
		bar1Total := common.Uint64(g.Bar1MemoryUsage.Total)

		nv := infra.NVIDIA{
			DeviceAttribute: infra.NVIDIADeviceAttribute{
				GPUUUID:             common.Str(g.UUID),
				DriverVersion:       driverVersion,
				CUDAVersion:         cudaVersion,
				ProductName:         common.Str(g.ProductName),
				ProductBrand:        common.Str(g.ProductBrand),
				ProductArchitecture: common.Str(g.ProductArchitecture),
				Index:               i,
				MinorNumber:         common.Str(g.MinorNumber),
				Serial:              common.Str(g.Serial),
				PCIBusID:            common.FirstStr(g.Pci.PciBusID, g.ID),
				VBIOSVersion:        common.Str(g.VbiosVersion),
				ComputeMode:         common.Str(g.ComputeMode),
				PersistenceMode:     common.Str(g.PersistenceMode),
				MIGMode:             common.Str(g.MigMode.CurrentMig),
				VirtualizationMode:  common.Str(g.GpuVirtualizationMode.VirtualizationMode),
				HostVGPUMode:        common.Str(g.GpuVirtualizationMode.HostVgpuMode),
				VGPULicenseStatus:   common.Str(g.VgpuSoftwareLicensedProduct.LicenseStatus),
			},
			Performance: infra.NVIDIAPerformance{
				GPUUsage:          common.Uint32(g.Utilization.GpuUtil),
				MemoryUsage:       common.Uint32(g.Utilization.MemoryUtil),
				EncoderUsage:      common.Uint32(g.Utilization.EncoderUtil),
				DecoderUsage:      common.Uint32(g.Utilization.DecoderUtil),
				FBMemoryUsed:      fbUsed,
				FBMemoryTotal:     fbTotal,
				FBMemoryFree:      common.Uint64(g.FbMemoryUsage.Free),
				FBMemoryReserved:  common.Uint64(g.FbMemoryUsage.Reserved),
				FBMemoryUsage:     common.Percent(fbUsed, fbTotal),
				Bar1MemoryUsed:    bar1Used,
				Bar1MemoryTotal:   bar1Total,
				Bar1MemoryFree:    common.Uint64(g.Bar1MemoryUsage.Free),
				Bar1MemoryUsage:   common.Percent(bar1Used, bar1Total),
				PerformanceState:  common.Str(g.PerformanceState),
				FanSpeed:          common.Uint32(g.FanSpeed),
				TemperatureGPU:    common.Int32(g.Temperature.GpuTemp),
				TemperatureMemory: common.Int32(g.Temperature.MemoryTemp),
				PowerDraw:         common.Float64(g.PowerReadings.PowerDraw),
				PowerLimit:        common.Float64(g.PowerReadings.PowerLimit),
				ClockGraphics:     common.Uint32(g.Clocks.GraphicsClock),
				ClockSM:           common.Uint32(g.Clocks.SmClock),
				ClockMemory:       common.Uint32(g.Clocks.MemClock),
				ClockVideo:        common.Uint32(g.Clocks.VideoClock),
				MaxClockGraphics:  common.Uint32(g.MaxClocks.GraphicsClock),
				MaxClockSM:        common.Uint32(g.MaxClocks.SmClock),
				MaxClockMemory:    common.Uint32(g.MaxClocks.MemClock),
				PCIeLinkGen:       common.Uint32(g.Pci.PciGpuLinkInfo.PcieGen.CurrentLinkGen),
				PCIeLinkWidth:     common.LinkWidth(g.Pci.PciGpuLinkInfo.LinkWidths.CurrentLinkWidth),
				PCIeReplayCounter: common.Uint64(g.Pci.ReplayCounter),
				ClocksEventReasons: common.ClocksEventReasons(map[string]string{
					"gpu_idle":                    g.ClocksThrottleReasons.GpuIdle,
					"applications_clocks_setting": g.ClocksThrottleReasons.ApplicationsClocksSetting,
					"sw_power_cap":                g.ClocksThrottleReasons.SwPowerCap,
					"hw_slowdown":                 g.ClocksThrottleReasons.HwSlowdown,
					"hw_thermal_slowdown":         g.ClocksThrottleReasons.HwThermalSlowdown,
					"hw_power_brake_slowdown":     g.ClocksThrottleReasons.HwPowerBrakeSlowdown,
					"sync_boost":                  g.ClocksThrottleReasons.SyncBoost,
					"sw_thermal_slowdown":         g.ClocksThrottleReasons.SwThermalSlowdown,
					"display_clocks_setting":      g.ClocksThrottleReasons.DisplayClocksSetting,
				}),
			},
			ECC: common.ECC(common.Str(g.EccMode.CurrentEcc),
				eccErrors(g.EccErrors.Volatile), eccErrors(g.EccErrors.Aggregate)),
			MIGDevices: migDevices(g.MigDevices.MigDevice),
			Processes:  processes(g.Processes.ProcessInfo),
		}

		gpus = append(gpus, nv)
	}

	return gpus, nil
}

// eccErrors reads whichever counter shape the driver emitted. Pre-Volta
// drivers report single_bit/double_bit totals and leave the SRAM/DRAM
// counters absent; both are surfaced under their own names rather than
// folded together, since they count different things.
func eccErrors(c eccCounts) infra.NVIDIAECCErrors {
	return infra.NVIDIAECCErrors{
		SRAMCorrectable:   common.Uint64(c.SramCorrectable),
		SRAMUncorrectable: common.Uint64(c.SramUncorrectable),
		DRAMCorrectable:   common.Uint64(c.DramCorrectable),
		DRAMUncorrectable: common.Uint64(c.DramUncorrectable),
		SingleBit:         common.Uint64(c.SingleBit.Total),
		DoubleBit:         common.Uint64(c.DoubleBit.Total),
	}
}

func migDevices(devices []migDevice) []infra.NVIDIAMIGDevice {
	if len(devices) == 0 {
		return nil
	}

	migs := make([]infra.NVIDIAMIGDevice, 0, len(devices))
	for _, d := range devices {
		migs = append(migs, infra.NVIDIAMIGDevice{
			Index:             common.Str(d.Index),
			GPUInstanceID:     common.Str(d.GpuInstanceID),
			ComputeInstanceID: common.Str(d.ComputeInstanceID),
			UUID:              common.Str(d.UUID),
			FBMemoryTotal:     common.Uint64(d.FbMemoryUsage.Total),
			FBMemoryUsed:      common.Uint64(d.FbMemoryUsage.Used),
			FBMemoryFree:      common.Uint64(d.FbMemoryUsage.Free),
		})
	}

	return migs
}

func processes(infos []processInfo) []infra.NVIDIAProcess {
	if len(infos) == 0 {
		return nil
	}

	procs := make([]infra.NVIDIAProcess, 0, len(infos))
	for _, p := range infos {
		pid := common.Uint32(p.Pid)
		if pid == nil {
			continue
		}

		procs = append(procs, infra.NVIDIAProcess{
			PID:           *pid,
			Type:          common.Str(p.Type),
			Name:          common.ProcessName(p.ProcessName),
			UsedMemory:    common.Uint64(p.UsedMemory),
			GPUInstanceID: common.Str(p.GpuInstanceID),
		})
	}

	if len(procs) == 0 {
		return nil
	}

	return procs
}
