package infra

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/moxspec/moxspec/bonding"
)

func printNetAllStruct() {
	// var si sysinfo.SysInfo
	// si.GetSysInfo()
	// data, err := json.MarshalIndent(&si, "", "  ")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Println(string(data))
	var ni bonding.BondInterface
	data, err := json.MarshalIndent(&ni, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))

}

// func print_net_custom_info() {
// 	var si sysinfo.SysInfo

// 	fmt.Println("CPU Info:")
// 	fmt.Printf("  Model: %s\n", si.CPU.Model)
// 	fmt.Printf("  Speed : %d\n", si.CPU.Speed)
// 	fmt.Printf("  Cache: %d\n", si.CPU.Cache)
// 	fmt.Printf("  Cpus: %d\n", si.CPU.Cpus)
// 	fmt.Printf("  Cores: %d\n", si.CPU.Cores)
// 	fmt.Printf("  Threads: %s\n", si.CPU.Threads)

// 	fmt.Println("\nMemory Info:")
// 	fmt.Printf("  Type: %s MB\n", si.Memory.Type)
// 	fmt.Printf("  Speed: %d MB\n", si.Memory.Speed)
// 	fmt.Printf("  Size: %d MB\n", si.Memory.Size)

// 	fmt.Println("\nOS Info:")
// 	fmt.Printf("  Name: %s\n", si.OS.Name)
// 	fmt.Printf("  Version: %s\n", si.OS.Version)

// 	fmt.Println("\nProduct Info:")
// 	fmt.Printf("  Name: %s\n", si.Product.Name)
// 	fmt.Printf("  Vendor: %s\n", si.Product.Vendor)
// 	fmt.Printf("  Version: %s\n", si.Product.Version)

// 	fmt.Println("\nKernel Info:")
// 	fmt.Printf("  Version: %s\n", si.Kernel.Version)
// 	fmt.Printf("  Release: %s\n", si.Kernel.Release)
// 	fmt.Printf("  Architecture: %s\n", si.Kernel.Architecture)

// 	fmt.Println("\nNetwork Devices:")
// 	for _, netDev := range si.Network {
// 		fmt.Printf("  %s:\n", netDev.Name)
// 		fmt.Printf("    MAC Address: %s\n", netDev.MACAddress)
// 		fmt.Printf("    Port: %v\n", netDev.Port)
// 		fmt.Printf("    Driver: %v\n", netDev.Driver)
// 		fmt.Printf("    Speed: %v\n", netDev.Speed)
// 	}

// 	fmt.Println("\nStorage Devices:")
// 	for _, storageDev := range si.Storage {
// 		fmt.Printf("  %s:\n", storageDev.Name)
// 		fmt.Printf("    Driver: %s\n", storageDev.Driver)
// 		fmt.Printf("    Vendor: %s\n", storageDev.Vendor)
// 		fmt.Printf("    Model: %s\n", storageDev.Model)
// 		fmt.Printf("    Size: %d GB\n", storageDev.Size/1024/1024/1024)
// 	}

// 	fmt.Println("\nNode Info:")
// 	fmt.Printf("  Hostname: %s\n", si.Node.Hostname)
// 	fmt.Printf("  Hypervisor: %s\n", si.Node.Hypervisor)
// 	fmt.Printf("  MachineID: %s\n", si.Node.MachineID)
// 	fmt.Printf("  Timezone: %s\n", si.Node.Timezone)

// }

func ExampleNetInfo() {
	fmt.Println("### Print All of sysinfo strcut\n")
	printNetAllStruct()
	fmt.Println("###############\n")
	// fmt.Println("### Print Custom sysinfo\n")
	// print_custom_info()
	// fmt.Println("###############\n")

}
