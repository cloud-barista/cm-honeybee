package infra

import (
	"github.com/cloud-barista/cm-honeybee/model/infra"
	"github.com/zcalusic/sysinfo"
)

func GetComputeInfo() (infra.Compute, error) {
	var compute infra.Compute

	var si sysinfo.SysInfo
	si.GetSysInfo()

	var storage []infra.Storage
	for _, s := range si.Storage {
		storage = append(storage, infra.Storage{
			Name:   s.Name,
			Driver: s.Driver,
			Vendor: s.Vendor,
			Model:  s.Model,
			Serial: s.Serial,
			Size:   s.Size,
		})
	}

	compute = infra.Compute{
		OS: infra.System{
			OS: infra.OS{
				Name:         si.OS.Name,
				Vendor:       si.OS.Vendor,
				Version:      si.OS.Version,
				Release:      si.OS.Release,
				Architecture: si.OS.Architecture,
			},
			Kernel: infra.Kernel{
				Release:      si.Kernel.Release,
				Version:      si.Kernel.Version,
				Architecture: si.Kernel.Architecture,
			},
			Node: infra.Node{
				Hostname:   si.Node.Hostname,
				Hypervisor: si.Node.Hypervisor,
				Machineid:  si.Node.MachineID,
				Timezone:   si.Node.Timezone,
			},
		},
		ComputeResource: infra.ComputeResource{
			CPU: infra.CPU{
				Vendor:  si.CPU.Vendor,
				Model:   si.CPU.Model,
				Speed:   si.CPU.Speed,
				Cache:   si.CPU.Cache,
				Cpus:    si.CPU.Cpus,
				Cores:   si.CPU.Cores,
				Threads: si.CPU.Threads,
			},
			Memory: infra.Memory{
				Type:  si.Memory.Type,
				Speed: si.Memory.Speed,
				Size:  si.Memory.Size,
			},
			Storage: storage,
		},
	}

	return compute, nil
}
