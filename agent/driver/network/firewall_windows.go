//go:build windows

package network

import (
	"fmt"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/network"
	"github.com/kumako/go-win64api"
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

		// Skip all of any-any allows
		if (rule.LocalAddresses == "*" || rule.LocalAddresses == "LocalSubnet") &&
			(rule.RemoteAddresses == "*" || rule.RemoteAddresses == "LocalSubnet") &&
			(rule.LocalPorts == "*" || rule.LocalPorts == "LocalSubnet" || rule.LocalPorts == "") &&
			(rule.RemotePorts == "*" || rule.RemotePorts == "LocalSubnet" || rule.RemotePorts == "") {
			continue
		}

		var fwRule network.FirewallRule
		protocol := parseProtocol(rule.Protocol)
		if protocol == protocolUnknown {
			continue
		}

		localAddr := rule.LocalAddresses
		if localAddr == "LocalSubnet" {
			localAddr = localSubnetCIDR
		}

		remoteAddr := rule.RemoteAddresses
		if remoteAddr == "LocalSubnet" {
			remoteAddr = localSubnetCIDR
		}

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
	return fwRules, nil
}
