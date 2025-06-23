//go:build linux

package network

import (
	"bufio"
	"fmt"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/network"
	"github.com/coreos/go-iptables/iptables"
	"github.com/jollaman999/utils/logger"
	"os/exec"
	"regexp"
	"strings"
)

func parseIptablesRules(ipt *iptables.IPTables, rules []string, prevPriority *uint, direction string, isIPv6 bool) []network.FirewallRule {
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
					if ipt == nil {
						continue
					}
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
						fwSubRules := parseIptablesRules(ipt, subRules, prevPriority, direction, isIPv6)
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

type FirewalldNFTRule struct {
	Chain    string
	Protocol string
	Port     string
	Source   string
	Dest     string
	Action   string
	IsIPv6   bool
}

func parsePortRule(line, protocol string, portRegex, actionRegex, ipv4SrcRegex, ipv4DstRegex, ipv6SrcRegex, ipv6DstRegex *regexp.Regexp, isIPv6 bool) FirewalldNFTRule {
	rule := FirewalldNFTRule{
		Protocol: protocol,
		IsIPv6:   isIPv6,
	}

	if portMatch := portRegex.FindStringSubmatch(line); len(portMatch) > 1 {
		rule.Port = portMatch[1]
	}

	if actionMatch := actionRegex.FindStringSubmatch(line); len(actionMatch) > 1 {
		rule.Action = strings.ToUpper(actionMatch[1])
	}

	if isIPv6 {
		if srcMatch := ipv6SrcRegex.FindStringSubmatch(line); len(srcMatch) > 1 {
			rule.Source = srcMatch[1]
		}
		if dstMatch := ipv6DstRegex.FindStringSubmatch(line); len(dstMatch) > 1 {
			rule.Dest = dstMatch[1]
		}
	} else {
		if srcMatch := ipv4SrcRegex.FindStringSubmatch(line); len(srcMatch) > 1 {
			rule.Source = srcMatch[1]
		}
		if dstMatch := ipv4DstRegex.FindStringSubmatch(line); len(dstMatch) > 1 {
			rule.Dest = dstMatch[1]
		}
	}

	return rule
}

func parseICMPRule(line, protocol string, actionRegex, srcRegex, dstRegex *regexp.Regexp, isIPv6 bool) FirewalldNFTRule {
	rule := FirewalldNFTRule{
		Protocol: protocol,
		IsIPv6:   isIPv6,
	}

	if actionMatch := actionRegex.FindStringSubmatch(line); len(actionMatch) > 1 {
		rule.Action = strings.ToUpper(actionMatch[1])
	}

	if srcMatch := srcRegex.FindStringSubmatch(line); len(srcMatch) > 1 {
		rule.Source = srcMatch[1]
	}
	if dstMatch := dstRegex.FindStringSubmatch(line); len(dstMatch) > 1 {
		rule.Dest = dstMatch[1]
	}

	return rule
}

func parseAddressOnlyRule(line string, actionRegex, ipv4SrcRegex, ipv4DstRegex, ipv6SrcRegex, ipv6DstRegex *regexp.Regexp, isIPv6 bool) FirewalldNFTRule {
	rule := FirewalldNFTRule{
		IsIPv6: isIPv6,
	}

	if actionMatch := actionRegex.FindStringSubmatch(line); len(actionMatch) > 1 {
		rule.Action = strings.ToUpper(actionMatch[1])
	}

	if isIPv6 {
		if srcMatch := ipv6SrcRegex.FindStringSubmatch(line); len(srcMatch) > 1 {
			rule.Source = srcMatch[1]
		}
		if dstMatch := ipv6DstRegex.FindStringSubmatch(line); len(dstMatch) > 1 {
			rule.Dest = dstMatch[1]
		}
	} else {
		if srcMatch := ipv4SrcRegex.FindStringSubmatch(line); len(srcMatch) > 1 {
			rule.Source = srcMatch[1]
		}
		if dstMatch := ipv4DstRegex.FindStringSubmatch(line); len(dstMatch) > 1 {
			rule.Dest = dstMatch[1]
		}
	}

	return rule
}

func parseFirewalldNftables(output string) []FirewalldNFTRule {
	var rules []FirewalldNFTRule
	scanner := bufio.NewScanner(strings.NewReader(output))

	currentChain := "INPUT"
	chainRegex := regexp.MustCompile(`chain filter_(INPUT|OUTPUT|FORWARD)`)

	tcpPortRegex := regexp.MustCompile(`tcp dport (\d+)`)
	udpPortRegex := regexp.MustCompile(`udp dport (\d+)`)

	ipv4SrcRegex := regexp.MustCompile(`ip saddr ([0-9./]+)`)
	ipv4DstRegex := regexp.MustCompile(`ip daddr ([0-9./]+)`)
	ipv6SrcRegex := regexp.MustCompile(`ip6 saddr ([0-9a-fA-F:./]+)`)
	ipv6DstRegex := regexp.MustCompile(`ip6 daddr ([0-9a-fA-F:./]+)`)

	actionRegex := regexp.MustCompile(`(accept|drop|reject)`)
	ipv6Regex := regexp.MustCompile(`ip6|icmpv6`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if chainMatch := chainRegex.FindStringSubmatch(line); len(chainMatch) > 1 {
			currentChain = chainMatch[1]
			continue
		}

		if !strings.Contains(line, "accept") && !strings.Contains(line, "drop") && !strings.Contains(line, "reject") {
			continue
		}

		isIPv6 := ipv6Regex.MatchString(line)

		if strings.Contains(line, "tcp dport") {
			rule := parsePortRule(line, "tcp", tcpPortRegex, actionRegex, ipv4SrcRegex, ipv4DstRegex, ipv6SrcRegex, ipv6DstRegex, isIPv6)
			rule.Chain = currentChain
			if rule.Action != "" {
				rules = append(rules, rule)
			}
		}

		if strings.Contains(line, "udp dport") {
			rule := parsePortRule(line, "udp", udpPortRegex, actionRegex, ipv4SrcRegex, ipv4DstRegex, ipv6SrcRegex, ipv6DstRegex, isIPv6)
			rule.Chain = currentChain
			if rule.Action != "" {
				rules = append(rules, rule)
			}
		}

		if strings.Contains(line, "icmp type") {
			rule := parseICMPRule(line, "icmp", actionRegex, ipv4SrcRegex, ipv4DstRegex, false)
			rule.Chain = currentChain
			if rule.Action != "" {
				rules = append(rules, rule)
			}
		}

		if strings.Contains(line, "icmpv6 type") {
			rule := parseICMPRule(line, "icmpv6", actionRegex, ipv6SrcRegex, ipv6DstRegex, true)
			rule.Chain = currentChain
			if rule.Action != "" {
				rules = append(rules, rule)
			}
		}

		if (strings.Contains(line, "ip saddr") || strings.Contains(line, "ip6 saddr") || strings.Contains(line, "ip daddr") || strings.Contains(line, "ip6 daddr")) && !strings.Contains(line, "dport") {
			rule := parseAddressOnlyRule(line, actionRegex, ipv4SrcRegex, ipv4DstRegex, ipv6SrcRegex, ipv6DstRegex, isIPv6)
			rule.Chain = currentChain
			if rule.Action != "" {
				rules = append(rules, rule)
			}
		}
	}

	return rules
}

func getFirewalldNftablesRules() []FirewalldNFTRule {
	output, err := exec.Command("nft", "list", "table", "inet", "firewalld").Output()
	if err != nil {
		logger.Println(logger.INFO, false, "firewalld nftables not found: "+err.Error())
		return []FirewalldNFTRule{}
	}

	return parseFirewalldNftables(string(output))
}

func normalizeIPToCIDR(ip string, isIPv6 bool) string {
	if strings.Contains(ip, "/") {
		return ip
	}

	if isIPv6 {
		return ip + "/128"
	} else {
		return ip + "/32"
	}
}

func buildIptablesCommand(rule FirewalldNFTRule) string {
	if rule.Action == "" {
		rule.Action = "ACCEPT"
	}

	var cmd strings.Builder
	cmd.WriteString("iptables -A ")
	cmd.WriteString(rule.Chain)

	if rule.Source != "" {
		cmd.WriteString(" -s ")
		cmd.WriteString(normalizeIPToCIDR(rule.Source, false))
	}

	if rule.Dest != "" {
		cmd.WriteString(" -d ")
		cmd.WriteString(normalizeIPToCIDR(rule.Dest, false))
	}

	if rule.Protocol != "" {
		cmd.WriteString(" -p ")
		cmd.WriteString(rule.Protocol)
	}

	if rule.Port != "" {
		cmd.WriteString(" --dport ")
		cmd.WriteString(rule.Port)
	}

	cmd.WriteString(" -j ")
	cmd.WriteString(rule.Action)

	return cmd.String()
}

func buildIP6tablesCommand(rule FirewalldNFTRule) string {
	if rule.Action == "" {
		rule.Action = "ACCEPT"
	}

	var cmd strings.Builder
	cmd.WriteString("ip6tables -A ")
	cmd.WriteString(rule.Chain)

	if rule.Source != "" {
		cmd.WriteString(" -s ")
		cmd.WriteString(normalizeIPToCIDR(rule.Source, true))
	}

	if rule.Dest != "" {
		cmd.WriteString(" -d ")
		cmd.WriteString(normalizeIPToCIDR(rule.Dest, true))
	}

	if rule.Protocol == "icmpv6" {
		cmd.WriteString(" -p icmpv6")
	} else if rule.Protocol != "" {
		cmd.WriteString(" -p ")
		cmd.WriteString(rule.Protocol)
	}

	if rule.Port != "" {
		cmd.WriteString(" --dport ")
		cmd.WriteString(rule.Port)
	}

	cmd.WriteString(" -j ")
	cmd.WriteString(rule.Action)

	return cmd.String()
}

func convertToIptablesFormat(rules []FirewalldNFTRule) []string {
	var commands []string

	for _, rule := range rules {
		if rule.IsIPv6 {
			cmd := buildIP6tablesCommand(rule)
			if cmd != "" {
				commands = append(commands, cmd)
			}
		} else {
			cmd := buildIptablesCommand(rule)
			if cmd != "" {
				commands = append(commands, cmd)
			}
		}
	}

	return commands
}

func convertFirewalldNFTablesToIptablesCommands() []string {
	rules := getFirewalldNftablesRules()
	commands := convertToIptablesFormat(rules)

	return commands
}

func firewalldNFTablesToModelFirewallRule(prevPriority *uint) []network.FirewallRule {
	var fw = make([]network.FirewallRule, 0)

	rules := convertFirewalldNFTablesToIptablesCommands()

	for _, rule := range rules {
		var direction string

		s := strings.Split(rule, "-A ")
		if len(s) != 2 {
			continue
		}
		if strings.HasPrefix(s[1], "INPUT") {
			direction = "inbound"
		} else if strings.HasPrefix(s[1], "OUTPUT") {
			direction = "outbound"
		} else {
			continue
		}

		if strings.HasPrefix(rule, "iptables") {
			fw = append(fw, parseIptablesRules(nil, []string{rule}, prevPriority, direction, false)...)
		} else if strings.HasPrefix(rule, "ip6tables") {
			fw = append(fw, parseIptablesRules(nil, []string{rule}, prevPriority, direction, true)...)
		}
	}

	return fw
}

func iptablesToModelFirewallRule(prevPriority *uint, ipt *iptables.IPTables, isIPv6 bool) ([]network.FirewallRule, error) {
	var fw = make([]network.FirewallRule, 0)

	rules, err := ipt.List("filter", "INPUT")
	if err != nil {
		logger.Println(logger.DEBUG, true, "FIREWALL: "+err.Error())
		return fw, err
	}
	fw = append(fw, parseIptablesRules(ipt, rules, prevPriority, "inbound", isIPv6)...)

	rules, err = ipt.List("filter", "OUTPUT")
	if err != nil {
		logger.Println(logger.DEBUG, true, "FIREWALL: "+err.Error())
		return fw, err
	}
	fw = append(fw, parseIptablesRules(ipt, rules, prevPriority, "outbound", isIPv6)...)

	return fw, nil
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
	var fw = make([]network.FirewallRule, 0)
	var prevPriority uint = 0

	firewalldNFTablesRules := firewalldNFTablesToModelFirewallRule(&prevPriority)
	fw = append(fw, firewalldNFTablesRules...)

	ipt4, err := iptables.New(iptables.IPFamily(iptables.ProtocolIPv4))
	if err != nil {
		logger.Println(logger.ERROR, false, "FIREWALL: Failed to handle iptables.")
		return []network.FirewallRule{}, err
	}

	ipv4fw, err := iptablesToModelFirewallRule(&prevPriority, ipt4, false)
	if err != nil {
		logger.Println(logger.ERROR, false, "FIREWALL: Failed to get IPv4 rules.")
		return []network.FirewallRule{}, err
	}
	fw = append(fw, ipv4fw...)

	ipt6, err := iptables.New(iptables.IPFamily(iptables.ProtocolIPv6))
	if err != nil {
		logger.Println(logger.ERROR, false, "FIREWALL: Failed to handle ip6tables.")
	}

	ipv6fw, err := iptablesToModelFirewallRule(&prevPriority, ipt6, true)
	if err != nil {
		logger.Println(logger.ERROR, false, "FIREWALL: Failed to get IPv6 rules.")
		return []network.FirewallRule{}, err
	}
	fw = append(fw, ipv6fw...)

	removeDuplicatedRules(&fw)

	return fw, nil
}
