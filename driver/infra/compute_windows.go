// Getting compute information for Windows

//go:build windows

package infra

import (
	"errors"
	"fmt"
	"github.com/cloud-barista/cm-honeybee/model/infra"
	"github.com/jaypipes/ghw"
	"github.com/jollaman999/utils/logger"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/yumaojun03/dmidecode"
	"github.com/yumaojun03/dmidecode/parser/memory"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
	"os"
	"path"
	"strings"
	"time"
	"unsafe"
)

func getWindowsReleaseVersion() (string, error) {
	var h windows.Handle // like HostIDWithContext(), we query the registry using the raw windows.RegOpenKeyEx/RegQueryValueEx
	err := windows.RegOpenKeyEx(windows.HKEY_LOCAL_MACHINE, windows.StringToUTF16Ptr(`SOFTWARE\Microsoft\Windows NT\CurrentVersion`), 0, windows.KEY_READ|windows.KEY_WOW64_64KEY, &h)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = windows.RegCloseKey(h)
	}()
	var bufLen uint32
	var valType uint32
	err = windows.RegQueryValueEx(h, windows.StringToUTF16Ptr(`DisplayVersion`), nil, &valType, nil, &bufLen)
	if err != nil {
		return "", err
	}
	regBuf := make([]uint16, bufLen/2+1)
	err = windows.RegQueryValueEx(h, windows.StringToUTF16Ptr(`DisplayVersion`), nil, &valType, (*byte)(unsafe.Pointer(&regBuf[0])), &bufLen)
	if err != nil {
		return "", err
	}

	return windows.UTF16ToString(regBuf[:]), nil
}

func getKernelLastModifiedDate() (string, error) {
	filestat, err := os.Stat(os.Getenv("windir") + "/system32/ntoskrnl.exe")
	if err != nil {
		return "", err
	}

	return filestat.ModTime().String(), nil
}

const (
	VIRTUAL_MACHINE_TYPE_QEMU      = "qemu"
	VIRTUAL_MACHINE_TYPE_XEN       = "xen"
	VIRTUAL_MACHINE_TYPE_VMWARE    = "vmware"
	VIRTUAL_MACHINE_TYPE_VBOX      = "vbox"
	VIRTUAL_MACHINE_TYPE_PARALLELS = "parallels"
	VIRTUAL_MACHINE_TYPE_VIRTUALPC = "virtual pc"
	VIRTUAL_MACHINE_TYPE_HYPERV    = "hyper-v"
	VIRTUAL_MACHINE_TYPE_UNKNOWN   = "unknown"
)

func extractKeyTypeFrom(registryKey string) (registry.Key, string, error) {
	firstSeparatorIndex := strings.Index(registryKey, string(os.PathSeparator))
	keyTypeStr := registryKey[:firstSeparatorIndex]
	keyPath := registryKey[firstSeparatorIndex+1:]

	var keyType registry.Key
	switch keyTypeStr {
	case "HKLM":
		keyType = registry.LOCAL_MACHINE
		break
	case "HKCR":
		keyType = registry.CLASSES_ROOT
		break
	case "HKCU":
		keyType = registry.CURRENT_USER
		break
	case "HKU":
		keyType = registry.USERS
		break
	case "HKCC":
		keyType = registry.CURRENT_CONFIG
		break
	default:
		return keyType, "", errors.New(fmt.Sprintf("Invalid keytype (%v)", keyTypeStr))
	}

	return keyType, keyPath, nil
}

func doesRegistryKeyExist(registryKeys []string) (bool, error) {
	for _, key := range registryKeys {
		subkeyPrefix := ""

		// Handle trailing wildcard
		if key[len(key)-1:] == "*" {
			key, subkeyPrefix = path.Split(key)
			subkeyPrefix = subkeyPrefix[:len(subkeyPrefix)-1] // remove *
		}

		keyType, keyPath, err := extractKeyTypeFrom(key)

		if err != nil {
			return false, err
		}

		keyHandle, err := registry.OpenKey(keyType, keyPath, registry.QUERY_VALUE)
		if err != nil {
			return false, errors.New(fmt.Sprintf("can't open %v : %v", key, err))
		}
		defer func() {
			_ = keyHandle.Close()
		}()

		// If a wildcard has been specified...
		if subkeyPrefix != "" {
			// ... we look for sub-keys to see if one exists
			subKeys, err := keyHandle.ReadSubKeyNames(0xFFFF)
			if err != nil {
				return false, err
			}

			for _, subKeyName := range subKeys {
				if strings.HasPrefix(subKeyName, subkeyPrefix) {
					return true, nil
				}
			}

			return false, nil
		} else {
			// The registryKey we were looking for has been found
			return true, nil
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
		return VIRTUAL_MACHINE_TYPE_HYPERV
	}

	exist, err = doesRegistryKeyExist(parallelsKeys)
	if err != nil {
		logger.Println(logger.DEBUG, true, "COMPUTE: "+err.Error())
	}
	if exist {
		return VIRTUAL_MACHINE_TYPE_PARALLELS
	}

	exist, err = doesRegistryKeyExist(virtualPCKeys)
	if err != nil {
		logger.Println(logger.DEBUG, true, "COMPUTE: "+err.Error())
	}
	if exist {
		return VIRTUAL_MACHINE_TYPE_VIRTUALPC
	}

	exist, err = doesRegistryKeyExist(xenKeys)
	if err != nil {
		logger.Println(logger.DEBUG, true, "COMPUTE: "+err.Error())
	}
	if exist {
		return VIRTUAL_MACHINE_TYPE_XEN
	}

	return ""
}

func checkVirtualMachineTypeString(input string) string {
	if strings.Contains(input, "qemu") {
		return VIRTUAL_MACHINE_TYPE_QEMU
	} else if strings.Contains(input, "vbox") || strings.Contains(input, "virtualbox") {
		return VIRTUAL_MACHINE_TYPE_VBOX
	} else if strings.Contains(input, "vmware") {
		return VIRTUAL_MACHINE_TYPE_VMWARE
	}

	return VIRTUAL_MACHINE_TYPE_UNKNOWN
}

func getVirtualMachineType(dmidecode *dmidecode.Decoder) (string, error) {
	pro, err := dmidecode.Processor()
	if err != nil {
		return "", err
	}

	if len(pro) > 0 {
		manufacturer := strings.ToLower(pro[0].Manufacturer)
		return checkVirtualMachineTypeString(manufacturer), nil
	}

	bios, err := dmidecode.BIOS()
	for _, b := range bios {
		typeByVendor := checkVirtualMachineTypeString(b.Vendor)
		if typeByVendor != VIRTUAL_MACHINE_TYPE_UNKNOWN {
			return typeByVendor, nil
		}
		typeByVersion := checkVirtualMachineTypeString(b.BIOSVersion)
		if typeByVersion != VIRTUAL_MACHINE_TYPE_UNKNOWN {
			return typeByVersion, nil
		}
	}

	return checkVirtualMachineRegistry(), nil
}

func GetComputeInfo() (infra.Compute, error) {
	var compute infra.Compute

	// OS information
	releaseVersion, err := getWindowsReleaseVersion()
	if err != nil {
		return infra.Compute{}, err
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

	mem, err := dmi.MemoryDevice()
	if err != nil {
		return infra.Compute{}, err
	}

	var memType = memory.MemoryDeviceTypeUnknown
	var memSpeed = uint16(0)
	var memSize = uint16(0)

	for _, m := range mem {
		memSize += m.Size
		if m.Type != memory.MemoryDeviceTypeUnknown {
			memType = m.Type
		}
		if m.Speed > 0 {
			memSpeed = m.Speed
		}
	}

	// storage information
	var storage []infra.Storage

	block, err := ghw.Block()
	if err != nil {
		return infra.Compute{}, err
	}

	for _, disk := range block.Disks {
		storage = append(storage, infra.Storage{
			Name:   disk.Name,
			Driver: disk.StorageController.String(),
			Vendor: disk.Vendor,
			Model:  disk.Model,
			Serial: disk.SerialNumber,
			Size:   uint(disk.SizeBytes / 1024 / 1024 / 1024),
		})
	}

	// All of compute information
	compute = infra.Compute{
		OS: infra.System{
			OS: infra.OS{
				Name:         h.OS,
				Vendor:       h.Platform,
				Version:      h.PlatformVersion,
				Release:      releaseVersion,
				Architecture: h.KernelArch,
			},
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
				Vendor:  c[0].VendorID,
				Model:   c[0].ModelName,
				Speed:   uint(c[0].Mhz),
				Cache:   uint(c[0].CacheSize),
				Cpus:    cpus,
				Cores:   cores,
				Threads: threads,
			},
			Memory: infra.Memory{
				Type:  memType.String(),
				Speed: uint(memSpeed),
				Size:  uint(memSize),
			},
			Storage: storage,
		},
	}

	return compute, nil
}
