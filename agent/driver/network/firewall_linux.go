//go:build linux

package network

import (
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/network"
	"github.com/coreos/go-iptables/iptables"
	"github.com/jollaman999/utils/logger"
	"strings"
)

func parseIptablesRules(ipt *iptables.IPTables, rules []string, prevPriority *uint, direction string) []network.FirewallRule {
	var fwRules = make([]network.FirewallRule, 0)

	for _, rule := range rules {
		var fwRule network.FirewallRule
		var skip bool

		ruleSplited := strings.Split(rule, " ")
		ruleSplitedLen := len(ruleSplited)
		for i, str := range ruleSplited {
			if strings.HasPrefix(str, "-") && ruleSplitedLen > i+1 {
				switch str {
				case "-P":
					fallthrough
				case "-N":
					skip = true
				case "-j":
					jump := strings.ToLower(ruleSplited[i+1])
					switch jump {
					case "accept":
						fwRule.Action = "allow"
					case "drop", "deny":
						fwRule.Action = "deny"
					default:
						subRules, err := ipt.List("filter", ruleSplited[i+1])
						if err != nil {
							logger.Println(logger.DEBUG, true, "FIREWALL: "+err.Error())
							skip = true
							break
						}
						fwSubRules := parseIptablesRules(ipt, subRules, prevPriority, direction)
						fwRules = append(fwRules, fwSubRules...)
						skip = true
					}
				case "-s":
					fwRule.Src = ruleSplited[i+1]
				case "-d":
					fwRule.Dst = ruleSplited[i+1]
				case "-p":
					protocol := strings.ToLower(ruleSplited[i+1])
					fwRule.Protocol = protocol
					if protocol == "tcp" || protocol == "udp" {
						for j, str := range ruleSplited {
							if strings.HasPrefix(str, "--") && ruleSplitedLen > j+1 {
								switch str {
								case "--sport":
									fallthrough
								case "--sports":
									fwRule.SrcPorts = ruleSplited[j+1]
								case "--dport":
									fallthrough
								case "--dports":
									fwRule.DstPorts = ruleSplited[j+1]
								}
							}
						}
						fwRule.SrcPorts = strings.ReplaceAll(fwRule.SrcPorts, ":", "-")
						fwRule.DstPorts = strings.ReplaceAll(fwRule.DstPorts, ":", "-")
					} else if protocol == "ipv6-icmp" {
						fwRule.Protocol = "icmpv6"
					}
				}
			}
		}
		if skip {
			continue
		}

		*prevPriority++

		fwRule.Direction = direction
		fwRule.Priority = *prevPriority
		if len(fwRule.Protocol) == 0 {
			fwRule.Protocol = "*"
		}
		if len(fwRule.SrcPorts) == 0 {
			fwRule.SrcPorts = "*"
		}
		if len(fwRule.DstPorts) == 0 {
			fwRule.DstPorts = "*"
		}

		fwRules = append(fwRules, fwRule)
	}

	return fwRules
}

func iptablesToModelFirewallRule(ipt *iptables.IPTables) ([]network.FirewallRule, error) {
	var fw = make([]network.FirewallRule, 0)
	var prevPriority uint

	rules, err := ipt.List("filter", "INPUT")
	if err != nil {
		logger.Println(logger.DEBUG, true, "FIREWALL: "+err.Error())
		return fw, err
	}
	fw = append(fw, parseIptablesRules(ipt, rules, &prevPriority, "inbound")...)

	rules, err = ipt.List("filter", "OUTPUT")
	if err != nil {
		logger.Println(logger.DEBUG, true, "FIREWALL: "+err.Error())
		return fw, err
	}
	fw = append(fw, parseIptablesRules(ipt, rules, &prevPriority, "outbound")...)

	return fw, nil
}

func GetFirewallRules() ([]network.FirewallRule, error) {
	var fw = make([]network.FirewallRule, 0)

	ipt4, err := iptables.New(iptables.IPFamily(iptables.ProtocolIPv4))
	if err != nil {
		logger.Println(logger.ERROR, false, "FIREWALL: Failed to handle iptables.")
		return []network.FirewallRule{}, err
	}

	ipv4fw, err := iptablesToModelFirewallRule(ipt4)
	if err != nil {
		logger.Println(logger.ERROR, false, "FIREWALL: Failed to get IPv4 rules.")
		return []network.FirewallRule{}, err
	}
	fw = append(fw, ipv4fw...)

	ipt6, err := iptables.New(iptables.IPFamily(iptables.ProtocolIPv6))
	if err != nil {
		logger.Println(logger.ERROR, false, "FIREWALL: Failed to handle ip6tables.")
	}

	ipv6fw, err := iptablesToModelFirewallRule(ipt6)
	if err != nil {
		logger.Println(logger.ERROR, false, "FIREWALL: Failed to get IPv4 rules.")
		return []network.FirewallRule{}, err
	}
	fw = append(fw, ipv6fw...)

	return fw, nil
}
