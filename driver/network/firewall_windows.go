//go:build windows

package network

import (
	"github.com/cloud-barista/cm-honeybee/pkg/api/rest/model/onprem/network"
)

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
