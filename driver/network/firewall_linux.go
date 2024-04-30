//go:build linux

package network

import (
	"github.com/cloud-barista/cm-honeybee/pkg/api/rest/model/onprem/network"
)

const (
	TableFilter   = "filter"
	TableNat      = "nat"
	TableMangle   = "mangle"
	TableRaw      = "raw"
	TableSecurity = "security"
)

var Tables = []string{TableFilter, TableNat, TableMangle, TableRaw, TableSecurity}

//
//func iptablesToModelTables(ipt *iptables.IPTables) []network.Table {
//	var ts = make([]network.Table, 0)
//
//	for _, table := range Tables {
//		var t network.Table
//		var cs []network.Chain
//
//		chains, err := ipt.ListChains(table)
//		if err != nil {
//			logger.Println(logger.DEBUG, true, "NETFILTER: "+err.Error())
//			continue
//		}
//
//		for _, chain := range chains {
//			var c network.Chain
//			var rs []string
//
//			rules, err := ipt.List(table, chain)
//			if err != nil {
//				logger.Println(logger.DEBUG, true, "NETFILTER: "+err.Error())
//				continue
//			}
//
//			rs = append(rs, rules...)
//
//			c.ChainName = chain
//			c.Rules = rs
//			cs = append(cs, c)
//		}
//
//		t.TableName = table
//		t.Chains = cs
//		ts = append(ts, t)
//	}
//
//	return ts
//}

// GetFirewallRules TODO
func GetFirewallRules() ([]network.FirewallRule, error) {
	return []network.FirewallRule{
		{
			Priority:  0,
			Src:       "TODO",
			Dst:       "TODO",
			SrcPort:   0,
			DstPort:   0,
			Protocol:  "TODO",
			Direction: "TODO",
			Action:    "TODO",
		},
	}, nil
}
