// Getting bonding interfaces for Linux

//go:build linux

package network

import (
	"bufio"
	"github.com/cloud-barista/cm-honeybee/pkg/api/rest/model/network"
	"github.com/jollaman999/utils/fileutil"
	"github.com/shirou/gopsutil/v3/net"
	"path/filepath"
	"strings"
)

var bondingProcBase = "/proc/net/bonding"

func getBondingInterfaces() ([]string, error) {
	var interfaces []string

	paths, err := filepath.Glob(bondingProcBase + "/*")
	if err != nil {
		return nil, err
	}
	for _, p := range paths {
		interfaces = append(interfaces, filepath.Base(p))
	}

	return interfaces, nil
}

func parseBondPart(bond *network.Bonding, bondPart string) error {
	scanner := bufio.NewScanner(strings.NewReader(bondPart))

	for scanner.Scan() {
		line := scanner.Text()
		split := strings.Split(line, ":")
		if len(split) < 2 {
			continue
		}
		name := strings.TrimSpace(split[0])
		value := strings.TrimSpace(split[1])
		if strings.Contains(name, "Bonding Mode") {
			bond.BondingMode = value
		} else if strings.Contains(name, "Transmit Hash Policy") {
			bond.TransmitHashPolicy = value
		} else if strings.Contains(name, "Primary Slave") {
			bond.SlavesList.PrimarySlave = value
		} else if strings.Contains(name, "Currently Active Slave") {
			bond.SlavesList.CurrentlyActiveSlave = value
		} else if strings.Contains(name, "MII Status") {
			bond.MIIStatus = value
		} else if strings.Contains(name, "MII Polling Interval") {
			bond.MIIPollingInterval = value
		} else if strings.Contains(bond.BondingMode, "802.3ad") {
			if strings.Contains(name, "LACP active") {
				bond.ADInfo.LACPActive = value
			} else if strings.Contains(name, "LACP rate") {
				bond.ADInfo.LACPRate = value
			} else if strings.Contains(name, "Min links") {
				bond.ADInfo.MinLinks = value
			} else if strings.Contains(name, "Aggregator selection policy") {
				bond.ADInfo.ADSelect = value
			} else if strings.Contains(name, "System priority") {
				bond.ADInfo.SystemPriority = value
			} else if strings.Contains(name, "System MAC address") {
				bond.ADInfo.SystemMACAddress = strings.TrimSpace(strings.Replace(line, name+":", "", -1))
			} else if strings.Contains(bond.BondingMode, "Active Aggregator Info") {
				continue
			} else if strings.Contains(name, "Aggregator ID") {
				bond.ADInfo.ActiveAggregatorInfo.AggregatorID = value
			} else if strings.Contains(name, "Number of ports") {
				bond.ADInfo.ActiveAggregatorInfo.NumberOfPorts = value
			} else if strings.Contains(name, "Actor Key") {
				bond.ADInfo.ActiveAggregatorInfo.ActorKey = value
			} else if strings.Contains(name, "Partner Key") {
				bond.ADInfo.ActiveAggregatorInfo.PartnerKey = value
			} else if strings.Contains(name, "Partner Mac Address") {
				bond.ADInfo.ActiveAggregatorInfo.PartnerMACAddress = strings.TrimSpace(strings.Replace(line, name+":", "", -1))
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

func parseSlavePart(bond *network.Bonding, bondPart string) error {
	var slaveIdx = -1
	var slaveInterfaces []network.SlaveInterface

	scanner := bufio.NewScanner(strings.NewReader(bondPart))

	for scanner.Scan() {
		line := scanner.Text()
		split := strings.Split(line, ":")
		if len(split) < 2 {
			continue
		}
		name := strings.TrimSpace(split[0])
		value := strings.TrimSpace(split[1])
		if strings.Contains(name, "Slave Interface") {
			var slaveInterface network.SlaveInterface

			slaveInterface.Name = value
			slaveInterfaces = append(slaveInterfaces, slaveInterface)
			slaveIdx++
		} else if strings.Contains(name, "MII Status") {
			slaveInterfaces[slaveIdx].MIIStatus = value
		} else if strings.Contains(name, "Speed") {
			slaveInterfaces[slaveIdx].Speed = value
		} else if strings.Contains(name, "Duplex") {
			slaveInterfaces[slaveIdx].Duplex = value
		} else if strings.Contains(name, "Link Failure Count") {
			slaveInterfaces[slaveIdx].LinkFailureCount = value
		} else if strings.Contains(name, "Permanent HW addr") {
			slaveInterfaces[slaveIdx].PermanentHWAddr = strings.TrimSpace(strings.Replace(line, name+":", "", -1))
		} else if strings.Contains(name, "Aggregator ID") {
			slaveInterfaces[slaveIdx].AggregatorID = value
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	bond.SlaveInterface = slaveInterfaces

	return nil
}

func parseBondingInfo(bondIface string) (network.Bonding, error) {
	var bond network.Bonding

	bond.Name = bondIface

	content, err := fileutil.ReadFile(bondingProcBase + "/" + bondIface)
	if err != nil {
		return network.Bonding{}, err
	}
	content = strings.TrimSpace(content)

	splitIndex := strings.Index(content, "Slave Interface:")
	if splitIndex == -1 {
		splitIndex = len(content)
	}
	bondPart := content[:splitIndex]
	slavePart := content[splitIndex:]

	err = parseBondPart(&bond, bondPart)
	if err != nil {
		return bond, err
	}
	err = parseSlavePart(&bond, slavePart)
	if err != nil {
		return bond, err
	}

	var slaves []string
	for _, slave := range bond.SlaveInterface {
		slaves = append(slaves, slave.Name)
	}
	bond.SlavesList.Interfaces = slaves

	interfaces, err := net.Interfaces()
	if err != nil {
		return bond, err
	}
	for _, iface := range interfaces {
		if iface.Name == bond.Name {
			var addrs []string

			for _, addr := range iface.Addrs {
				addrs = append(addrs, addr.Addr)
			}
			bond.AddrList = addrs

			break
		}
	}

	return bond, nil
}

func GetBondingInfo() ([]network.Bonding, error) {
	var bonds []network.Bonding

	bondIfaces, err := getBondingInterfaces()
	if err != nil {
		return bonds, err
	}

	for _, bondIface := range bondIfaces {
		bond, err := parseBondingInfo(bondIface)
		if err != nil {
			return bonds, err
		}

		bonds = append(bonds, bond)
	}

	return bonds, nil
}
