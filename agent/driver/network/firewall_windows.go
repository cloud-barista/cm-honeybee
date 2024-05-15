//go:build windows

package network

import (
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/network"
	"github.com/kumako/go-win64api"
)

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
	for _, rule := range rules {
		if rule.Enabled {
			var fwRule network.FirewallRule

			protocol := parseProtocol(rule.Protocol)
			if protocol == protocolUnknown {
				continue
			}

			// Skip all of any-any allows
			if (rule.LocalAddresses == "*" || rule.LocalAddresses == "LocalSubnet") &&
				(rule.RemoteAddresses == "*" || rule.RemoteAddresses == "LocalSubnet") &&
				(rule.LocalPorts == "*" || rule.LocalPorts == "LocalSubnet" || rule.LocalPorts == "") &&
				(rule.RemotePorts == "*" || rule.RemotePorts == "LocalSubnet" || rule.RemotePorts == "") {
				continue
			}

			fwRule.Protocol = protocol

			if rule.Direction == winapi.NET_FW_RULE_DIR_IN {
				fwRule.Direction = "inbound"
				fwRule.Src = rule.RemoteAddresses
				fwRule.SrcPorts = rule.RemotePorts
				fwRule.Dst = rule.LocalAddresses
				fwRule.DstPorts = rule.LocalPorts
			} else if rule.Direction == winapi.NET_FW_RULE_DIR_OUT {
				fwRule.Direction = "outbound"
				fwRule.Src = rule.LocalAddresses
				fwRule.SrcPorts = rule.LocalPorts
				fwRule.Dst = rule.RemoteAddresses
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

	return fwRules, nil
}
