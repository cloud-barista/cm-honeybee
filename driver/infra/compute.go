package infra

import (
	"github.com/zcalusic/sysinfo"
)

type OS struct {
	Name         string `json:"name"`
	Vendor       string `json:"vendor"`
	Version      string `json:"version"`
	Release      string `json:"release"`
	Architecture string `json:"architecture"`
}

type Kernel struct {
	Release      string `json:"release"`
	Version      string `json:"version"`
	Architecture string `json:"architecture"`
}

type Node struct {
	Hostname   string `json:"hostname"`
	Hypervisor string `json:"hypervisor"`
	Machineid  string `json:"machineid"`
	Timezone   string `json:"timezone"`
}

type _OS struct {
	OS     OS     `json:"os"`
	Kernel Kernel `json:"kernel"`
	Node   Node   `json:"node"`
}

type CPU struct {
	Vendor  string `json:"vendor"`
	Model   string `json:"model"`
	Speed   uint   `json:"speed"`   // MHz
	Cache   uint   `json:"cache"`   // KB
	Cpus    uint   `json:"cpus"`    // ea
	Cores   uint   `json:"cores"`   // ea
	Threads uint   `json:"threads"` // ea
}

type Memory struct {
	Type  string `json:"type"`
	Speed uint   `json:"speed"` // MT/s
	Size  uint   `json:"size"`  // MB
}

type Storage struct {
	Name   string `json:"name"`
	Driver string `json:"driver"`
	Vendor string `json:"vendor"`
	Model  string `json:"model"`
	Serial string `json:"serial"`
	Size   uint   `json:"size"` // GB
}

type ComputeResource struct {
	CPU     CPU       `json:"cpu"`
	Memory  Memory    `json:"memory"`
	Storage []Storage `json:"storage"`
}

type Compute struct {
	OS              _OS             `json:"os"`
	ComputeResource ComputeResource `json:"compute_resource"`
}

func GetComputeInfo() (Compute, error) {
	var compute Compute

	var si sysinfo.SysInfo
	si.GetSysInfo()

	var storage []Storage
	for _, s := range si.Storage {
		storage = append(storage, Storage{
			Name:   s.Name,
			Driver: s.Driver,
			Vendor: s.Vendor,
			Model:  s.Model,
			Serial: s.Serial,
			Size:   s.Size,
		})
	}

	compute = Compute{
		OS: _OS{
			OS: OS{
				Name:         si.OS.Name,
				Vendor:       si.OS.Vendor,
				Version:      si.OS.Version,
				Release:      si.OS.Release,
				Architecture: si.OS.Architecture,
			},
			Kernel: Kernel{
				Release:      si.Kernel.Release,
				Version:      si.Kernel.Version,
				Architecture: si.Kernel.Architecture,
			},
			Node: Node{
				Hostname:   si.Node.Hostname,
				Hypervisor: si.Node.Hypervisor,
				Machineid:  si.Node.MachineID,
				Timezone:   si.Node.Timezone,
			},
		},
		ComputeResource: ComputeResource{
			CPU: CPU{
				Vendor:  si.CPU.Vendor,
				Model:   si.CPU.Model,
				Speed:   si.CPU.Speed,
				Cache:   si.CPU.Cache,
				Cpus:    si.CPU.Cpus,
				Cores:   si.CPU.Cores,
				Threads: si.CPU.Threads,
			},
			Memory: Memory{
				Type:  si.Memory.Type,
				Speed: si.Memory.Speed,
				Size:  si.Memory.Size,
			},
			Storage: storage,
		},
	}

	return compute, nil
}
