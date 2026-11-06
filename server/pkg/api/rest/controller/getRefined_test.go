package controller

import (
	"encoding/json"
	"reflect"
	"testing"

	inframodel "github.com/cloud-barista/cm-beetle/imdl/on-premise-model"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/infra"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/network"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/software"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
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

func TestListeningPortsFromListeners(t *testing.T) {
	listeners := []software.Listener{
		{Name: "nginx", Ports: []int{80}, ListenSockets: []software.ListenSocket{
			{Protocol: "tcp", BindAddress: "0.0.0.0", Port: 80},
			{Protocol: "tcp", BindAddress: "::", Port: 80},
		}},
		{Name: "named", ListenSockets: []software.ListenSocket{ // UDP only: no Ports
			{Protocol: "udp", BindAddress: "0.0.0.0", Port: 53},
			{Protocol: "tcp", BindAddress: "127.0.0.1", Port: 953},
		}},
		{Name: "sshd", Ports: []int{22}}, // collected by an agent without listen_sockets
	}

	want := []model.ListeningPortProperty{
		{Protocol: "tcp", Port: 22, ProcessName: "sshd"},
		{Protocol: "tcp", BindAddress: "0.0.0.0", Port: 80, ProcessName: "nginx"},
		{Protocol: "tcp", BindAddress: "::", Port: 80, ProcessName: "nginx"},
		{Protocol: "tcp", BindAddress: "127.0.0.1", Port: 953, ProcessName: "named"},
		{Protocol: "udp", BindAddress: "0.0.0.0", Port: 53, ProcessName: "named"},
	}

	if got := listeningPortsFromListeners(listeners); !reflect.DeepEqual(got, want) {
		t.Errorf("listeningPortsFromListeners() =\n%+v\nwant\n%+v", got, want)
	}
	if got := listeningPortsFromListeners(nil); got != nil {
		t.Errorf("listeningPortsFromListeners(nil) = %+v, want nil", got)
	}
}

// TestBuildRefinedInfraModelJSON pins the JSON path cm-beetle reads the
// listening ports from: onpremiseInfraModel.nodes[].listeningPorts[], next to
// the node's other fields rather than nested under the embedded NodeProperty.
func TestBuildRefinedInfraModelJSON(t *testing.T) {
	onprem := inframodel.OnpremInfra{Nodes: []inframodel.NodeProperty{
		{Hostname: "web-1"},
		{Hostname: "k8s-worker"}, // appended from the Kubernetes API
	}}
	ports := [][]model.ListeningPortProperty{
		{{Protocol: "tcp", BindAddress: "0.0.0.0", Port: 8080, ProcessName: "nginx"}},
	}

	b, err := json.Marshal(buildRefinedInfraModel(onprem, ports))
	if err != nil {
		t.Fatal(err)
	}

	var got struct {
		OnpremiseInfraModel struct {
			Nodes []map[string]json.RawMessage `json:"nodes"`
		} `json:"onpremiseInfraModel"`
	}
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	nodes := got.OnpremiseInfraModel.Nodes
	if len(nodes) != 2 {
		t.Fatalf("nodes = %s", b)
	}
	if string(nodes[0]["hostname"]) != `"web-1"` {
		t.Errorf("hostname not at the node's top level: %s", b)
	}
	if want := `[{"protocol":"tcp","bindAddress":"0.0.0.0","port":8080,"processName":"nginx"}]`; string(nodes[0]["listeningPorts"]) != want {
		t.Errorf("listeningPorts = %s, want %s", nodes[0]["listeningPorts"], want)
	}
	if _, ok := nodes[1]["listeningPorts"]; ok {
		t.Errorf("node without ports carries listeningPorts: %s", b)
	}
}
