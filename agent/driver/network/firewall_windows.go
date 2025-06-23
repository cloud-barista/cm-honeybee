//go:build windows

package network

import (
	"fmt"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/network"
	"github.com/kumako/go-win64api"
	"math/big"
	"net"
	"os/exec"
	"strings"
)

func getPreferredInterface() (*net.IPNet, error) {
	cmd := exec.Command("powershell", "-Command",
		"(Get-NetIPInterface -AddressFamily IPv4 | Where-Object { $_.ConnectionState -eq 'Connected' } | Sort-Object InterfaceMetric | Select-Object -First 1 InterfaceAlias).InterfaceAlias")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	interfaceName := strings.TrimSpace(string(output))
	if interfaceName == "" {
		cmd = exec.Command("powershell", "-Command",
			"(Get-NetIPInterface -AddressFamily IPv6 | Where-Object { $_.ConnectionState -eq 'Connected' } | Sort-Object InterfaceMetric | Select-Object -First 1 InterfaceAlias).InterfaceAlias")
		output, err = cmd.Output()
		if err != nil {
			return nil, err
		}
		interfaceName = strings.TrimSpace(string(output))
		if interfaceName == "" {
			return nil, fmt.Errorf("no active interface found")
		}
	}

	iface, err := net.InterfaceByName(interfaceName)
	if err != nil {
		return nil, err
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return nil, err
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok {
			return ipnet, nil
		}
	}

	return nil, fmt.Errorf("no IP address found for interface")
}

func getLocalSubnetCIDR() string {
	ipNet, err := getPreferredInterface()
	if err != nil {
		return "*"
	}
	return ipNet.String()
}

const protocolUnknown = "unknown"

func parseProtocol(fwRuleProtocol int32) string {
	switch fwRuleProtocol {
	case winapi.NET_FW_IP_PROTOCOL_ANY:
		return "*"
	case winapi.NET_FW_IP_PROTOCOL_TCP:
		return "tcp"
	case winapi.NET_FW_IP_PROTOCOL_UDP:
		return "udp"
	case winapi.NET_FW_IP_PROTOCOL_ICMPv4:
		return "icmp"
	case winapi.NET_FW_IP_PROTOCOL_ICMPv6:
		return "icmpv6"
	default:
		return protocolUnknown
	}
}

func ipToInt(ip net.IP) uint32 {
	ip = ip.To4()
	return uint32(ip[0])<<24 + uint32(ip[1])<<16 + uint32(ip[2])<<8 + uint32(ip[3])
}

func intToIP(n uint32) net.IP {
	return net.IPv4(byte(n>>24), byte(n>>16), byte(n>>8), byte(n))
}

func ipv4RangeToCIDRs(start, end net.IP) []string {
	var cidrs []string

	startInt := ipToInt(start)
	endInt := ipToInt(end)

	if startInt == endInt {
		return []string{start.String() + "/32"}
	}

	for startInt <= endInt {
		maxSize := 32
		for i := 0; i < 32; i++ {
			mask := uint32(1) << i
			if (startInt & (mask - 1)) != 0 {
				break
			}
			if startInt+mask-1 > endInt {
				break
			}
			maxSize = 32 - i - 1
		}

		cidr := fmt.Sprintf("%s/%d", intToIP(startInt).String(), maxSize)
		cidrs = append(cidrs, cidr)

		blockSize := uint32(1) << (32 - maxSize)
		startInt += blockSize
	}

	return cidrs
}

func ipv6ToInt(ip net.IP) *big.Int {
	ipInt := big.NewInt(0)
	ipInt.SetBytes(ip.To16())
	return ipInt
}

func intToIPv6(ipInt *big.Int) net.IP {
	bytes := ipInt.Bytes()
	if len(bytes) < 16 {
		padded := make([]byte, 16)
		copy(padded[16-len(bytes):], bytes)
		bytes = padded
	}
	return net.IP(bytes)
}

func ipv6RangeToCIDRs(start, end net.IP) []string {
	var cidrs []string

	startInt := ipv6ToInt(start)
	endInt := ipv6ToInt(end)

	if startInt.Cmp(endInt) == 0 {
		return []string{start.String() + "/128"}
	}

	for startInt.Cmp(endInt) <= 0 {
		maxSize := 128
		for i := 0; i < 128; i++ {
			mask := new(big.Int).Lsh(big.NewInt(1), uint(i))
			mask.Sub(mask, big.NewInt(1))

			if new(big.Int).And(startInt, mask).Cmp(big.NewInt(0)) != 0 {
				break
			}

			blockEnd := new(big.Int).Add(startInt, mask)
			if blockEnd.Cmp(endInt) > 0 {
				break
			}
			maxSize = 128 - i - 1
		}

		cidr := fmt.Sprintf("%s/%d", intToIPv6(startInt).String(), maxSize)
		cidrs = append(cidrs, cidr)

		blockSize := new(big.Int).Lsh(big.NewInt(1), uint(128-maxSize))
		startInt.Add(startInt, blockSize)
	}

	return cidrs
}

func rangeToCIDRs(startIP, endIP string) []string {
	start := net.ParseIP(startIP)
	end := net.ParseIP(endIP)

	if start == nil || end == nil {
		if strings.Contains(startIP, ":") {
			return []string{startIP + "/128", endIP + "/128"}
		}
		return []string{startIP + "/32", endIP + "/32"}
	}

	if start.To4() != nil && end.To4() != nil {
		return ipv4RangeToCIDRs(start.To4(), end.To4())
	}

	return ipv6RangeToCIDRs(start, end)
}

func normalizeAddresses(addr string, localSubnetCIDR string) []string {
	if addr == "" || addr == "*" {
		return []string{"0.0.0.0/0", "::/0"}
	}

	addresses := strings.Split(addr, ",")
	var result []string

	for _, address := range addresses {
		address = strings.TrimSpace(address)
		if address == "" {
			continue
		}

		if address == "LocalSubnet" {
			result = append(result, localSubnetCIDR)
			continue
		}

		if strings.Contains(address, "-") {
			parts := strings.Split(address, "-")
			if len(parts) == 2 {
				startAddr := strings.TrimSpace(parts[0])
				endAddr := strings.TrimSpace(parts[1])

				cidrs := rangeToCIDRs(startAddr, endAddr)
				result = append(result, cidrs...)
				continue
			}
		}

		if !strings.Contains(address, "/") {
			if strings.Contains(address, ":") {
				address += "/128"
			} else {
				address += "/32"
			}
		}

		result = append(result, address)
	}

	if len(result) == 0 {
		return []string{"0.0.0.0/0", "::/0"}
	}

	return result
}

func removeDuplicatedRules(fw *[]network.FirewallRule) {
	seen := make(map[string]bool)
	uniqueFw := make([]network.FirewallRule, 0)

	for _, rule := range *fw {
		key := fmt.Sprintf("%s-%s-%s-%s-%s-%s-%s",
			rule.Src, rule.Dst, rule.SrcPorts, rule.DstPorts,
			rule.Protocol, rule.Direction, rule.Action)

		if !seen[key] {
			seen[key] = true
			uniqueFw = append(uniqueFw, rule)
		}
	}

	for i := range uniqueFw {
		uniqueFw[i].Priority = uint(i + 1)
	}

	*fw = uniqueFw
}

func GetFirewallRules() ([]network.FirewallRule, error) {
	var fwRules = make([]network.FirewallRule, 0)
	rules, err := winapi.FirewallRulesGet()
	if err != nil {
		return nil, err
	}

	priority := 0
	localSubnetCIDR := getLocalSubnetCIDR()

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		protocol := parseProtocol(rule.Protocol)
		if protocol == protocolUnknown {
			continue
		}

		localAddresses := normalizeAddresses(rule.LocalAddresses, localSubnetCIDR)
		remoteAddresses := normalizeAddresses(rule.RemoteAddresses, localSubnetCIDR)

		for _, localAddr := range localAddresses {
			for _, remoteAddr := range remoteAddresses {
				// Skip invalid protocol mismatch
				if (localAddr == "0.0.0.0/0" && remoteAddr == "::/0") ||
					(localAddr == "::/0" && remoteAddr == "0.0.0.0/0") {
					continue
				}

				// Skip all local inbound rules
				if rule.Direction == winapi.NET_FW_RULE_DIR_IN {
					if remoteAddr == "LocalSubnet" ||
						strings.HasPrefix(remoteAddr, "fe80:") ||
						strings.Contains(remoteAddr, localSubnetCIDR) {
						continue
					}
				}

				// Skip all local outbound rules
				if rule.Direction == winapi.NET_FW_RULE_DIR_OUT {
					if remoteAddr == "LocalSubnet" ||
						strings.HasPrefix(remoteAddr, "fe80:") ||
						strings.Contains(remoteAddr, localSubnetCIDR) {
						continue
					}
				}

				// Skip all of between any/local/all-nodes/all-routers
				if (localAddr == "*" || localAddr == "LocalSubnet" ||
					strings.HasPrefix(localAddr, "fe80:") ||
					localAddr == "ff02::1/128" ||
					localAddr == "ff02::2/128" ||
					strings.Contains(localAddr, localSubnetCIDR)) &&
					(remoteAddr == "*" || remoteAddr == "LocalSubnet" ||
						strings.HasPrefix(remoteAddr, "fe80:") ||
						remoteAddr == "ff02::1/128" ||
						remoteAddr == "ff02::2/128" ||
						strings.Contains(remoteAddr, localSubnetCIDR)) &&
					(rule.LocalPorts == "*" || rule.LocalPorts == "LocalSubnet" || rule.LocalPorts == "") &&
					(rule.RemotePorts == "*" || rule.RemotePorts == "LocalSubnet" || rule.RemotePorts == "") {
					continue
				}

				var fwRule network.FirewallRule
				fwRule.Protocol = protocol

				if rule.Direction == winapi.NET_FW_RULE_DIR_IN {
					fwRule.Direction = "inbound"
					fwRule.Src = remoteAddr
					fwRule.SrcPorts = rule.RemotePorts
					fwRule.Dst = localAddr
					fwRule.DstPorts = rule.LocalPorts
				} else if rule.Direction == winapi.NET_FW_RULE_DIR_OUT {
					fwRule.Direction = "outbound"
					fwRule.Src = localAddr
					fwRule.SrcPorts = rule.LocalPorts
					fwRule.Dst = remoteAddr
					fwRule.DstPorts = rule.RemotePorts
				} else {
					continue
				}

				if rule.Action == winapi.NET_FW_ACTION_ALLOW {
					fwRule.Action = "allow"
				} else if rule.Action == winapi.NET_FW_ACTION_BLOCK {
					fwRule.Action = "deny"
				}

				priority++
				fwRule.Priority = uint(priority)
				fwRules = append(fwRules, fwRule)
			}
		}
	}

	removeDuplicatedRules(&fwRules)

	return fwRules, nil
}
