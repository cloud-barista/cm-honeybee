package controller

import (
	"encoding/json"
	"reflect"
	"testing"

	inframodel "github.com/cloud-barista/cm-beetle/imdl/on-premise-model"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/infra"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/network"
)

func TestMibToGiB(t *testing.T) {
	cases := []struct {
		mib      uint64
		rounded  uint64
		capacity uint64
	}{
		{mib: 0, rounded: 0, capacity: 0},
		{mib: 249, rounded: 0, capacity: 1},
		{mib: 362, rounded: 0, capacity: 1},
		{mib: 511, rounded: 0, capacity: 1},
		{mib: 512, rounded: 1, capacity: 1},
		{mib: 1020, rounded: 1, capacity: 1}, // 1 GiB VM as the OS sees it
		{mib: 1024, rounded: 1, capacity: 1},
		{mib: 1536, rounded: 2, capacity: 2},
		{mib: 2000, rounded: 2, capacity: 2},
		{mib: 8192, rounded: 8, capacity: 8},
	}

	for _, c := range cases {
		if got := mibToGiB(c.mib); got != c.rounded {
			t.Errorf("mibToGiB(%d) = %d, want %d", c.mib, got, c.rounded)
		}

		if got := mibToGiBCapacity(c.mib); got != c.capacity {
			t.Errorf("mibToGiBCapacity(%d) = %d, want %d", c.mib, got, c.capacity)
		}
	}
}

// TestDoGetRefinedInfraInfoMemory pins the reading that motivated the rounding:
// a 1 GiB KT Cloud instance whose collected memory truncated to a totalSize of
// 0, which reads downstream as a node with no memory at all.
func TestDoGetRefinedInfraInfoMemory(t *testing.T) {
	infraInfo := &infra.Infra{}
	infraInfo.Compute.ComputeResource.Memory = infra.Memory{
		Type:      "RAM",
		Speed:     0,
		Size:      1020,
		Used:      249,
		Available: 362,
	}

	refined, err := doGetRefinedInfraInfo(infraInfo)
	if err != nil {
		t.Fatalf("doGetRefinedInfraInfo: %v", err)
	}

	if refined.Memory.TotalSize != 1 {
		t.Errorf("Memory.TotalSize = %d, want 1", refined.Memory.TotalSize)
	}

	got, err := json.Marshal(refined.Memory)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	want := `{"type":"RAM","totalSize":1}`
	if string(got) != want {
		t.Errorf("refined memory = %s, want %s", got, want)
	}
}

// TestFirewallRulesFromCSP covers the rule shapes cb-spider reported for the
// AWS, Alibaba, NCP and OpenStack source VMs (protocol case, -1 ports, a
// 1-65535 span, an IPv6 CIDR) plus a port range and a rule with no CIDR.
func TestFirewallRulesFromCSP(t *testing.T) {
	csp := &infra.CSPInfo{Network: infra.CSPNetwork{SecurityGroups: []infra.CSPSecurityGroup{
		{Name: "sg-1", Rules: []infra.CSPSecurityRule{
			{Direction: "inbound", Protocol: "tcp", FromPort: "22", ToPort: "22", CIDR: "0.0.0.0/0"},
			{Direction: "outbound", Protocol: "ALL", FromPort: "-1", ToPort: "-1", CIDR: "0.0.0.0/0"},
			{Direction: "outbound", Protocol: "udp", FromPort: "1", ToPort: "65535", CIDR: "0.0.0.0/0"},
			{Direction: "inbound", Protocol: "icmp", FromPort: "-1", ToPort: "-1", CIDR: "198.51.100.0/24"},
			{Direction: "inbound", Protocol: "TCP", FromPort: "8080", ToPort: "8090", CIDR: "203.0.113.0/24"},
			{Direction: "outbound", Protocol: "all", FromPort: "-1", ToPort: "-1", CIDR: "::/0"},
			{Direction: "inbound", Protocol: "tcp", FromPort: "443", ToPort: "443", CIDR: ""},
		}},
		{Name: "sg-2", Rules: []infra.CSPSecurityRule{
			{Direction: "inbound", Protocol: "tcp", FromPort: "22", ToPort: "22", CIDR: "0.0.0.0/0"}, // same as sg-1
		}},
	}}}

	want := []inframodel.FirewallRuleProperty{
		{SrcCIDR: "0.0.0.0/0", DstCIDR: "0.0.0.0/0", SrcPorts: "*", DstPorts: "22", Protocol: "tcp", Direction: "inbound", Action: "allow"},
		{SrcCIDR: "0.0.0.0/0", DstCIDR: "0.0.0.0/0", SrcPorts: "*", DstPorts: "*", Protocol: "*", Direction: "outbound", Action: "allow"},
		{SrcCIDR: "0.0.0.0/0", DstCIDR: "0.0.0.0/0", SrcPorts: "*", DstPorts: "1-65535", Protocol: "udp", Direction: "outbound", Action: "allow"},
		{SrcCIDR: "198.51.100.0/24", DstCIDR: "0.0.0.0/0", SrcPorts: "*", DstPorts: "*", Protocol: "icmp", Direction: "inbound", Action: "allow"},
		{SrcCIDR: "203.0.113.0/24", DstCIDR: "0.0.0.0/0", SrcPorts: "*", DstPorts: "8080-8090", Protocol: "tcp", Direction: "inbound", Action: "allow"},
		{SrcCIDR: "::/0", DstCIDR: "::/0", SrcPorts: "*", DstPorts: "*", Protocol: "*", Direction: "outbound", Action: "allow"},
		{SrcCIDR: "", DstCIDR: "0.0.0.0/0", SrcPorts: "*", DstPorts: "443", Protocol: "tcp", Direction: "inbound", Action: "allow"},
	}

	if got := firewallRulesFromCSP(csp); !reflect.DeepEqual(got, want) {
		t.Errorf("firewallRulesFromCSP() =\n%+v\nwant\n%+v", got, want)
	}
	if got := firewallRulesFromCSP(nil); got != nil {
		t.Errorf("firewallRulesFromCSP(nil) = %+v, want nil", got)
	}

	// The host firewall rules come first and the security group rules follow.
	infraInfo := &infra.Infra{CSP: csp}
	infraInfo.Network.Host.FirewallRule = []network.FirewallRule{
		{Src: "0.0.0.0/0", Dst: "0.0.0.0/0", SrcPorts: "*", DstPorts: "80", Protocol: "tcp", Direction: "inbound", Action: "allow"},
	}
	refined, err := doGetRefinedInfraInfo(infraInfo)
	if err != nil {
		t.Fatalf("doGetRefinedInfraInfo: %v", err)
	}
	if len(refined.FirewallTable) != 1+len(want) || refined.FirewallTable[0].DstPorts != "80" ||
		!reflect.DeepEqual(refined.FirewallTable[1:], want) {
		t.Errorf("FirewallTable = %+v", refined.FirewallTable)
	}
}
