//go:build linux

package network

import (
	"github.com/cloud-barista/cm-honeybee/pkg/api/rest/model/onprem/network"
	"github.com/coreos/go-iptables/iptables"
	"github.com/jollaman999/utils/logger"
	"strconv"
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
					if jump == "accept" {
						fwRule.Action = "allow"
					} else if jump == "drop" || jump == "deny" {
						fwRule.Action = "deny"
					} else {
						subRules, err := ipt.List("filter", ruleSplited[i+1])
						if err != nil {
							logger.Println(logger.DEBUG, true, "FIREWALL: "+err.Error())
							skip = true
							break
						}
						fwSubRules := parseIptablesRules(ipt, subRules, prevPriority, direction)
						fwRules = append(fwRules, fwSubRules...)
						skip = true
						break
					}
				case "-s":
					fwRule.Src = ruleSplited[i+1]
				case "-d":
					fwRule.Dst = ruleSplited[i+1]
				case "-p":
					protocol := strings.ToLower(ruleSplited[i+1])
					if protocol == "tcp" || protocol == "udp" {
						fwRule.Protocol = protocol
						for j, str := range ruleSplited {
							if strings.HasPrefix(str, "--") && ruleSplitedLen > j+1 {
								switch str {
								case "--sport":
									sport, _ := strconv.Atoi(ruleSplited[j+1])
									fwRule.SrcPort = uint(sport)
								case "--dport":
									dport, _ := strconv.Atoi(ruleSplited[j+1])
									fwRule.DstPort = uint(dport)
								}
							}
						}
					}
					fwRule.Protocol = protocol
				}
			}
		}
		if skip {
			continue
		}

		*prevPriority++

		fwRule.Direction = direction
		fwRule.Priority = *prevPriority

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

// GetFirewallRules
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
