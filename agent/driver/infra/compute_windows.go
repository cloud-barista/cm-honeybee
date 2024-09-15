// Getting compute information for Windows

//go:build windows

package infra

import (
	"errors"
	"fmt"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/infra"
	"github.com/jaypipes/ghw"
	"github.com/jollaman999/utils/logger"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/yumaojun03/dmidecode"
	"github.com/yumaojun03/dmidecode/parser/memory"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
	"io"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func readRegistryKey(baseKey registry.Key, path string, keyName string) (string, error) {
	key, err := registry.OpenKey(baseKey, path, registry.QUERY_VALUE)
	if err != nil {
		return "", err
	}
	defer key.Close()

	val, _, err := key.GetStringValue(keyName)
	if err != nil {
		return "", err
	}

	return val, nil
}

func getWindowsVersion(productName string, buildNumber int) string {
	if strings.Contains(productName, "Windows 10") && buildNumber >= 22000 {
		productName = strings.Replace(productName, "Windows 10", "Windows 11", 1)
	}

	return productName
}

func getOSProperties() (infra.OS, error) {
	productName, err := readRegistryKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, "ProductName")
	if err != nil {
		return infra.OS{}, err
	}

	buildNumberStr, err := readRegistryKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, "CurrentBuild")
	if err != nil {
		return infra.OS{}, err
	}

	buildNumber, err := strconv.Atoi(buildNumberStr)
	if err != nil {
		return infra.OS{}, err
	}

	prettyName := getWindowsVersion(productName, buildNumber)

	versionID, err := readRegistryKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, "ReleaseId")
	if err != nil {
		return infra.OS{}, err
	}

	version, err := readRegistryKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, "CurrentVersion")
	if err != nil {
		return infra.OS{}, err
	}

	versionCodename, err := readRegistryKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, "DisplayVersion")
	if err != nil {
		versionCodename = "N/A"
	}

	osInfo := infra.OS{
		PrettyName:      prettyName,
		Name:            "Windows",
		VersionID:       versionID,
		Version:         version,
		VersionCodename: versionCodename,
		ID:              "windows",
		IDLike:          "windows",
	}

	return osInfo, nil
}

func getKernelLastModifiedDate() (string, error) {
	filestat, err := os.Stat(os.Getenv("windir") + "/system32/ntoskrnl.exe")
	if err != nil {
		return "", err
	}

	return filestat.ModTime().String(), nil
}

const (
	VirtualMachineTypeQemu      = "qemu"
	VirtualMachineTypeXen       = "xen"
	VirtualMachineTypeVmware    = "vmware"
	VirtualMachineTypeVbox      = "vbox"
	VirtualMachineTypeParallels = "parallels"
	VirtualMachineTypeVirtualpc = "virtual pc"
	VirtualMachineTypeHyperv    = "hyper-v"
	VirtualMachineTypeUnknown   = "unknown"
)

func lastPathSeparator(s string) int {
	i := len(s) - 1
	for i >= 0 && s[i] != os.PathSeparator {
		i--
	}
	return i
}

func pathSplit(path string) (dir, file string) {
	i := lastPathSeparator(path)
	return path[:i+1], path[i+1:]
}

func extractKeyTypeFrom(registryKey string) (registry.Key, string, error) {
	firstSeparatorIndex := strings.Index(registryKey, string(os.PathSeparator))
	keyTypeStr := registryKey[:firstSeparatorIndex]
	keyPath := registryKey[firstSeparatorIndex+1:]

	var keyType registry.Key
	switch keyTypeStr {
	case "HKLM":
		keyType = registry.LOCAL_MACHINE
	case "HKCR":
		keyType = registry.CLASSES_ROOT
	case "HKCU":
		keyType = registry.CURRENT_USER
	case "HKU":
		keyType = registry.USERS
	case "HKCC":
		keyType = registry.CURRENT_CONFIG
	default:
		return keyType, "", fmt.Errorf("Invalid keytype (%v)", keyTypeStr)
	}

	return keyType, keyPath, nil
}

func doesRegistryKeyExist(registryKeys []string) (bool, error) {
	for _, key := range registryKeys {
		subkeyPrefix := ""

		// Handle trailing wildcard
		if key[len(key)-1:] == "*" {
			key, subkeyPrefix = pathSplit(key)
			subkeyPrefix = subkeyPrefix[:len(subkeyPrefix)-1] // remove *
		}

		keyType, keyPath, err := extractKeyTypeFrom(key)
		if err != nil {
			return false, err
		}

		keyHandle, err := registry.OpenKey(keyType, keyPath, registry.QUERY_VALUE|registry.ENUMERATE_SUB_KEYS)
		if err != nil {
			return false, fmt.Errorf("can't open %v : %v", key, err)
		}
		defer func() {
			_ = keyHandle.Close()
		}()

		// The registryKey we were looking for has been found
		if subkeyPrefix == "" {
			break
		}

		// If a wildcard has been specified,
		// we look for sub-keys to see if one exists
		subKeys, err := keyHandle.ReadSubKeyNames(0xFFFF)
		if err != nil {
			if err == io.EOF {
				return false, nil
			}
			return false, err
		}

		for _, subKeyName := range subKeys {
			if strings.HasPrefix(subKeyName, subkeyPrefix) {
				return true, nil
			}
		}
	}

	return false, nil
}

// checkVirtualMachineRegistry() referenced from https://github.com/ShellCode33/VM-Detection
func checkVirtualMachineRegistry() string {
	hyperVKeys := []string{
		`HKLM\SOFTWARE\Microsoft\Hyper-V`,
		`HKLM\SOFTWARE\Microsoft\VirtualMachine`,
		`HKLM\SOFTWARE\Microsoft\Virtual Machine\Guest\Parameters`,
		`HKLM\SYSTEM\ControlSet001\Services\vmicheartbeat`,
		`HKLM\SYSTEM\ControlSet001\Services\vmicvss`,
		`HKLM\SYSTEM\ControlSet001\Services\vmicshutdown`,
		`HKLM\SYSTEM\ControlSet001\Services\vmicexchange`,
	}

	parallelsKeys := []string{
		`HKLM\SYSTEM\CurrentControlSet\Enum\PCI\VEN_1AB8*`,
	}

	virtualPCKeys := []string{
		`HKLM\SYSTEM\CurrentControlSet\Enum\PCI\VEN_5333*`,
		`HKLM\SYSTEM\ControlSet001\Services\vpcbus`,
		`HKLM\SYSTEM\ControlSet001\Services\vpc-s3`,
		`HKLM\SYSTEM\ControlSet001\Services\vpcuhub`,
		`HKLM\SYSTEM\ControlSet001\Services\msvmmouf`,
	}

	xenKeys := []string{
		`HKLM\HARDWARE\ACPI\DSDT\xen`,
		`HKLM\HARDWARE\ACPI\FADT\xen`,
		`HKLM\HARDWARE\ACPI\RSDT\xen`,
		`HKLM\SYSTEM\ControlSet001\Services\xenevtchn`,
		`HKLM\SYSTEM\ControlSet001\Services\xennet`,
		`HKLM\SYSTEM\ControlSet001\Services\xennet6`,
		`HKLM\SYSTEM\ControlSet001\Services\xensvc`,
		`HKLM\SYSTEM\ControlSet001\Services\xenvdb`,
	}

	exist, err := doesRegistryKeyExist(hyperVKeys)
	if err != nil {
		logger.Println(logger.DEBUG, true, "COMPUTE: "+err.Error())
	}
	if exist {
		return VirtualMachineTypeHyperv
	}

	exist, err = doesRegistryKeyExist(parallelsKeys)
	if err != nil {
		logger.Println(logger.DEBUG, true, "COMPUTE: "+err.Error())
	}
	if exist {
		return VirtualMachineTypeParallels
	}

	exist, err = doesRegistryKeyExist(virtualPCKeys)
	if err != nil {
		logger.Println(logger.DEBUG, true, "COMPUTE: "+err.Error())
	}
	if exist {
		return VirtualMachineTypeVirtualpc
	}

	exist, err = doesRegistryKeyExist(xenKeys)
	if err != nil {
		logger.Println(logger.DEBUG, true, "COMPUTE: "+err.Error())
	}
	if exist {
		return VirtualMachineTypeXen
	}

	return ""
}

func checkVirtualMachineTypeString(input string) string {
	if strings.Contains(input, "qemu") {
		return VirtualMachineTypeQemu
	} else if strings.Contains(input, "vbox") || strings.Contains(input, "virtualbox") {
		return VirtualMachineTypeVbox
	} else if strings.Contains(input, "vmware") {
		return VirtualMachineTypeVmware
	}

	return VirtualMachineTypeUnknown
}

func getVirtualMachineType(dmidecode *dmidecode.Decoder) (string, error) {
	pro, err := dmidecode.Processor()
	if err != nil {
		return "", err
	}

	if len(pro) > 0 {
		manufacturer := strings.ToLower(pro[0].Manufacturer)
		typeByCPU := checkVirtualMachineTypeString(manufacturer)
		if typeByCPU != VirtualMachineTypeUnknown {
			return typeByCPU, nil
		}
	}

	bios, err := dmidecode.BIOS()
	if err != nil {
		return "", err
	}

	for _, b := range bios {
		typeByVendor := checkVirtualMachineTypeString(b.Vendor)
		if typeByVendor != VirtualMachineTypeUnknown {
			return typeByVendor, nil
		}
		typeByVersion := checkVirtualMachineTypeString(b.BIOSVersion)
		if typeByVersion != VirtualMachineTypeUnknown {
			return typeByVersion, nil
		}
	}

	return checkVirtualMachineRegistry(), nil
}

func getVolumeLabel(drive string) (string, error) {
	volumeName := make([]uint16, syscall.MAX_PATH+1)

	err := windows.GetVolumeInformation(
		windows.StringToUTF16Ptr(drive+"\\"), // Drive path
		&volumeName[0],                       // Volume name buffer
		uint32(len(volumeName)),              // Size of the volume name buffer
		nil,                                  // Volume serial number (not used)
		nil,                                  // Maximum component length (not used)
		nil,                                  // File system flags (not used)
		nil,                                  // File system name (not used)
		0,                                    // Size of the file system name (not used)
	)
	if err != nil {
		return "", err
	}

	// Convert from Unicode to string
	return syscall.UTF16ToString(volumeName), nil
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
		return infra.Compute{}, err
	}

	// Get DMI
	dmi, err := dmidecode.New()
	if err != nil {
		return infra.Compute{}, err
	}

	// Get virtual machine type
	virtualizationSystem, err := getVirtualMachineType(dmi)
	if err != nil {
		return infra.Compute{}, err
	}

	// Get kernel version
	kernelLastModifiedDate, err := getKernelLastModifiedDate()
	if err != nil {
		return infra.Compute{}, err
	}

	// CPU information
	c, err := cpu.Info()
	if err != nil {
		return infra.Compute{}, err
	}

	pro, err := dmi.Processor()
	if err != nil {
		return infra.Compute{}, err
	}

	cpus := uint(len(pro))
	var cores uint
	var threads uint
	if cpus > 0 {
		cores = uint(pro[0].CoreCount)
		threads = uint(pro[0].ThreadCount)
	} else {
		return infra.Compute{}, errors.New("failed to get information of processors")
	}

	// timezone information
	t := time.Now()
	tz, _ := t.Zone()

	memoryDevice, err := dmi.MemoryDevice()
	if err != nil {
		return infra.Compute{}, err
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
		return infra.Compute{}, err
	}

	rootDisk := infra.Disk{}
	dataDisk := []infra.Disk{}
	for _, d := range block.Disks {
		for _, part := range d.Partitions {
			if strings.EqualFold(part.Type, "Installable File System") {
				dUsage, err := disk.Usage(part.MountPoint)
				if err != nil {
					return compute, err
				}
				label, err := getVolumeLabel(part.Name)
				if err != nil {
					return compute, err
				}
				rootDisk = infra.Disk{
					Name:      part.Name,
					Label:     label,
					Type:      d.DriveType.String(),
					Size:      uint(dUsage.Total / 1024 / 1024 / 1024),
					Used:      uint(dUsage.Used / 1024 / 1024 / 1024),
					Available: uint(dUsage.Free / 1024 / 1024 / 1024),
				}
			} else {
				dUsage, err := disk.Usage(part.MountPoint)
				if err != nil {
					continue
				}
				label, err := getVolumeLabel(part.Name)
				if err != nil {
					return compute, err
				}
				dataDisk = append(dataDisk, infra.Disk{
					Name:      part.Name,
					Label:     label,
					Type:      d.DriveType.String(),
					Size:      uint(dUsage.Total / 1024 / 1024 / 1024),
					Used:      uint(dUsage.Used / 1024 / 1024 / 1024),
					Available: uint(dUsage.Free / 1024 / 1024 / 1024),
				})
			}
		}
	}

	// All of compute information
	compute = infra.Compute{
		OS: infra.System{
			OS: os,
			Kernel: infra.Kernel{
				Release:      h.KernelVersion,
				Version:      kernelLastModifiedDate,
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
