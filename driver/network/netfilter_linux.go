// Getting netfilter list for Linux

//go:build linux

package network

import (
	"github.com/cloud-barista/cm-honeybee/model/network"
	"github.com/coreos/go-iptables/iptables"
	"github.com/jollaman999/utils/logger"
)

const (
	TableFilter   = "filter"
	TableNat      = "nat"
	TableMangle   = "mangle"
	TableRaw      = "raw"
	TableSecurity = "security"
)

var Tables = []string{TableFilter, TableNat, TableMangle, TableRaw, TableSecurity}

func iptablesToModelTables(ipt *iptables.IPTables) []network.Table {
	var ts = make([]network.Table, 0)

	for _, table := range Tables {
		var t network.Table
		var cs []network.Chain

		chains, err := ipt.ListChains(table)
		if err != nil {
			logger.Println(logger.DEBUG, true, "NETFILTER: "+err.Error())
			continue
		}

		for _, chain := range chains {
			var c network.Chain
			var rs []string

			rules, err := ipt.List(table, chain)
			if err != nil {
				logger.Println(logger.DEBUG, true, "NETFILTER: "+err.Error())
				continue
			}

			rs = append(rs, rules...)

			c.ChainName = chain
			c.Rules = rs
			cs = append(cs, c)
		}

		t.TableName = table
		t.Chains = cs
		ts = append(ts, t)
	}

	return ts
}

func GetNetfilterList() (network.Netfilter, error) {
	var netfilter network.Netfilter

	ipt4, err := iptables.New(iptables.IPFamily(iptables.ProtocolIPv4))
	if err != nil {
		return netfilter, err
	}

	ipt6, err := iptables.New(iptables.IPFamily(iptables.ProtocolIPv6))
	if err != nil {
		logger.Println(logger.ERROR, false, "NETFILTER: Failed to get IPv6 tables.")
	}

	netfilter = network.Netfilter{
		IPv4Tables: iptablesToModelTables(ipt4),
		IPv6Tables: iptablesToModelTables(ipt6),
	}

	return netfilter, nil
}
