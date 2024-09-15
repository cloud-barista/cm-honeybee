// Getting compute information for Linux

//go:build linux

package infra

import (
	"bufio"
	"errors"
	"github.com/shirou/gopsutil/v3/disk"
	"regexp"
	"strings"
	"time"

	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/infra"
	"github.com/jaypipes/ghw"
	"github.com/jollaman999/utils/cmd"
	"github.com/jollaman999/utils/fileutil"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/yumaojun03/dmidecode"
	"github.com/yumaojun03/dmidecode/parser/memory"
)

func getOSProperties() (infra.OS, error) {
	var OS infra.OS

	content, err := fileutil.ReadFile("/etc/os-release")
	if err != nil {
		return OS, errors.New("failed to read os-release file")
	}
	content = strings.TrimSpace(content)
	scanner := bufio.NewScanner(strings.NewReader(content))

	for scanner.Scan() {
		line := scanner.Text()
		split := strings.Split(line, "=")
		if len(split) < 2 {
			continue
		}
		name := strings.TrimSpace(split[0])
		value := strings.Replace(strings.TrimSpace(split[1]), "\"", "", -1)
		if name == "PRETTY_NAME" {
			OS.PrettyName = value
		} else if name == "NAME" {
			OS.Name = value
		} else if name == "VERSION_ID" {
			OS.VersionID = value
		} else if name == "VERSION" {
			OS.Version = value
		} else if name == "VERSION_CODENAME" {
			OS.VersionCodename = value
		} else if name == "ID" {
			OS.ID = value
		} else if name == "ID_LIKE" {
			OS.IDLike = value
		}
	}

	if OS.VersionCodename == "" {
		re := regexp.MustCompile(`\(([^)]+)\)`)
		matches := re.FindStringSubmatch(OS.Version)
		if len(matches) > 1 {
			OS.VersionCodename = matches[1]
		}
	}

	return OS, nil
}
func getKernelVersion() (string, error) {
	output, err := cmd.RunCMD("uname -v")
	if err != nil {
		return "", err
	}

	output = strings.TrimSuffix(output, "\n")

	return output, nil
}

func GetComputeInfo() (infra.Compute, error) {
	var compute infra.Compute

	// OS information
	os, err := getOSProperties()
	if err != nil {
		return compute, err
	}

	// host information
	h, err := host.Info()
	if err != nil {
		return compute, err
	}
	virtualizationSystem := h.VirtualizationSystem
	if h.VirtualizationRole != "guest" {
		virtualizationSystem = ""
	}

	// Get kernel version
	kernelVersion, err := getKernelVersion()
	if err != nil {
		return compute, err
	}

	// Get DMI
	dmi, err := dmidecode.New()
	if err != nil {
		return compute, err
	}

	// CPU information
	c, err := cpu.Info()
	if err != nil {
		return compute, err
	}

	pro, err := dmi.Processor()
	if err != nil {
		return compute, err
	}

	cpus := uint(len(pro))
	var cores uint
	var threads uint
	if cpus > 0 {
		cores = uint(pro[0].CoreCount)
		threads = uint(pro[0].ThreadCount)
	} else {
		return compute, errors.New("failed to get information of processors")
	}

	// timezone information
	t := time.Now()
	tz, _ := t.Zone()

	memoryDevice, err := dmi.MemoryDevice()
	if err != nil {
		return compute, err
	}

	var memType = memory.MemoryDeviceTypeUnknown
	var memSpeed = uint(0)
	var memSize = uint(0)

	for _, m := range memoryDevice {
		memSize += uint(m.Size)
		if m.Type != memory.MemoryDeviceTypeUnknown {
			memType = m.Type
		}
		if m.Speed > 0 {
			memSpeed = uint(m.Speed)
		}
	}

	v, err := mem.VirtualMemory()
	if err != nil {
		return compute, err
	}

	memUsed := uint(v.Used / 1024 / 1024)
	memAvailable := memSize - memUsed

	// storage information
	block, err := ghw.Block()
	if err != nil {
		return compute, err
	}

	rootDisk := infra.Disk{}
	dataDisk := []infra.Disk{}
	for _, d := range block.Disks {
		if !strings.Contains(d.Name, "loop") {
			for _, part := range d.Partitions {
				if strings.EqualFold(part.MountPoint, "/") {
					dUsage, err := disk.Usage(part.MountPoint)
					if err != nil {
						return compute, err
					}
					rootDisk = infra.Disk{
						Name:      part.Name,
						Label:     part.Label,
						Type:      d.DriveType.String(),
						Size:      uint(dUsage.Total / 1024 / 1024 / 1024),
						Used:      uint(dUsage.Used / 1024 / 1024 / 1024),
						Available: uint(dUsage.Free / 1024 / 1024 / 1024),
					}
				} else {
					if !strings.Contains(strings.ToUpper(part.MountPoint), "EFI") && !strings.Contains(strings.ToUpper(part.Label), "EFI") {
						dUsage, err := disk.Usage(part.MountPoint)
						if err != nil {
							continue
						}
						dataDisk = append(dataDisk, infra.Disk{
							Name:      part.Name,
							Label:     part.Label,
							Type:      d.DriveType.String(),
							Size:      uint(dUsage.Total / 1024 / 1024 / 1024),
							Used:      uint(dUsage.Used / 1024 / 1024 / 1024),
							Available: uint(dUsage.Free / 1024 / 1024 / 1024),
						})
					}
				}
			}
		}
	}

	// All of compute information
	compute = infra.Compute{
		OS: infra.System{
			OS: os,
			Kernel: infra.Kernel{
				Release:      h.KernelVersion,
				Version:      kernelVersion,
				Architecture: h.KernelArch,
			},
			Node: infra.Node{
				Hostname:   h.Hostname,
				Hypervisor: virtualizationSystem,
				Machineid:  h.HostID,
				Timezone:   tz,
			},
		},
		ComputeResource: infra.ComputeResource{
			CPU: infra.CPU{
				Vendor:   c[0].VendorID,
				Model:    c[0].ModelName,
				MaxSpeed: uint(c[0].Mhz),
				Cache:    uint(c[0].CacheSize),
				Cpus:     cpus,
				Cores:    cores,
				Threads:  threads,
			},
			Memory: infra.Memory{
				Type:      memType.String(),
				Speed:     memSpeed,
				Size:      memSize,
				Used:      memUsed,
				Available: memAvailable,
			},
			RootDisk: rootDisk,
			DataDisk: dataDisk,
		},
	}

	return compute, nil
}
