package controller

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	inframodel "github.com/cloud-barista/cm-beetle/imdl/on-premise-model"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/infra"
	"github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/cloud-barista/cm-honeybee/server/lib/spider"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
)

// cspVMData is csp_data in the shape the CSP collection stores for an AWS VM
// reached without the agent: an 8 vCPU, 16 GiB spec, an Ubuntu 22.04 image and
// a 50 GB root disk.
const cspVMData = `{
 "provider": "aws",
 "region": "ap-northeast-2",
 "zone": "ap-northeast-2a",
 "name": "src-web-01",
 "id": "i-0123456789abcdef0",
 "vm_spec": "m5.2xlarge",
 "image": "ami-0aaaabbbbccccdddd",
 "platform": "LINUX/UNIX",
 "public_ip": "198.51.100.20",
 "private_ip": "192.0.2.10",
 "root_disk": {"name": "/dev/sda1", "label": "", "type": "gp3", "size": 50, "used": 0, "available": 0},
 "data_disks": ["vol-0data1", "vol-0data2"],
 "network": {
  "vpc": {"name": "vpc-src", "cidr": "192.0.2.0/24", "subnets": [
   {"name": "subnet-other", "cidr": "192.0.2.128/25", "zone": "ap-northeast-2c"},
   {"name": "subnet-src", "cidr": "192.0.2.0/25", "zone": "ap-northeast-2a"}
  ]},
  "subnet": "subnet-src",
  "security_groups": [{"name": "sg-src", "rules": [
   {"direction": "inbound", "protocol": "tcp", "from_port": "22", "to_port": "22", "cidr": "0.0.0.0/0"}
  ]}]
 },
 "tags": {"Name": "src-web-01"},
 "start_time": "2026-11-19T01:02:03Z",
 "status": "Running",
 "data_disk_details": [
  {"name": "vol-0data1", "id": "vol-0data1", "size": 100, "type": "gp3", "status": "Attached"},
  {"name": "vol-0data2", "id": "vol-0data2", "size": 0, "type": "gp3", "status": "Attached"}
 ],
 "vm_spec_detail": {"name": "m5.2xlarge", "vcpus": 8, "memory_mib": 16384, "disk_gb": 0, "architecture": "x86_64"},
 "image_detail": {"name": "ubuntu/images/hvm-ssd/ubuntu-jammy-22.04-amd64-server-20240207", "id": "ami-0aaaabbbbccccdddd",
  "os_distribution": "ubuntu-jammy-22.04", "os_architecture": "x86_64", "os_platform": "Linux/UNIX", "status": "Available"}
}`

func cspVMFixture(t *testing.T) *infra.CSPInfo {
	t.Helper()

	var csp infra.CSPInfo
	if err := json.Unmarshal([]byte(cspVMData), &csp); err != nil {
		t.Fatal(err)
	}
	return &csp
}

// A CSP VM without agent data becomes a node of what the CSP reports. The
// second data disk was reported without a size, so it is left out.
func TestDoGetRefinedInfraInfo_CSPOnly(t *testing.T) {
	csp := cspVMFixture(t)

	got, err := doGetRefinedInfraInfo(&infra.Infra{CSP: csp})
	if err != nil {
		t.Fatalf("doGetRefinedInfraInfo: %v", err)
	}

	want := inframodel.NodeProperty{
		Hostname:  "src-web-01",
		MachineId: "i-0123456789abcdef0",
		CPU:       inframodel.CpuProperty{Architecture: "x86_64", Cpus: 1, Cores: 8, Threads: 8},
		Memory:    inframodel.MemoryProperty{TotalSize: 16},
		RootDisk:  inframodel.DiskProperty{Label: "/dev/sda1", Type: "gp3", TotalSize: 50},
		DataDisks: []inframodel.DiskProperty{{Label: "vol-0data1", Type: "gp3", TotalSize: 100}},
		Interfaces: []inframodel.NetworkInterfaceProperty{{
			IPv4CidrBlocks: []string{"192.0.2.10/25", "198.51.100.20/32"},
			IPv6CidrBlocks: []string{},
		}},
		FirewallTable: []inframodel.FirewallRuleProperty{
			{SrcCIDR: "0.0.0.0/0", DstCIDR: "0.0.0.0/0", SrcPorts: "*", DstPorts: "22", Protocol: "tcp", Direction: "inbound", Action: "allow"},
		},
		OS: inframodel.OsProperty{
			PrettyName:      "ubuntu-jammy-22.04",
			Version:         "22.04",
			Name:            "Ubuntu",
			VersionID:       "22.04",
			VersionCodename: "jammy",
			ID:              "ubuntu",
		},
	}
	if !reflect.DeepEqual(*got, want) {
		t.Errorf("node =\n%+v\nwant\n%+v", *got, want)
	}
}

// What the CSP leaves out stays empty rather than guessed: no spec detail means
// no CPU or memory, the image's architecture stands in for the spec's, a
// private IP outside the listed subnet gets a host prefix, and a GPU spec gives
// one card per device.
func TestRefinedNodeFromCSP_Partial(t *testing.T) {
	t.Run("no spec detail", func(t *testing.T) {
		csp := cspVMFixture(t)
		csp.VMSpecDetail = nil
		csp.PublicIP = ""
		csp.Network.Subnet = "subnet-unlisted"

		got := refinedNodeFromCSP(csp)
		if got.CPU != (inframodel.CpuProperty{Architecture: "x86_64"}) || got.Memory.TotalSize != 0 {
			t.Errorf("CPU = %+v, Memory = %+v, want only the image architecture", got.CPU, got.Memory)
		}
		if want := []string{"192.0.2.10/32"}; len(got.Interfaces) != 1 ||
			!reflect.DeepEqual(got.Interfaces[0].IPv4CidrBlocks, want) {
			t.Errorf("Interfaces = %+v, want %v", got.Interfaces, want)
		}
	})

	t.Run("no address", func(t *testing.T) {
		csp := cspVMFixture(t)
		csp.PrivateIP, csp.PublicIP = "", ""
		if got := refinedNodeFromCSP(csp); got.Interfaces != nil {
			t.Errorf("Interfaces = %+v, want none", got.Interfaces)
		}
	})

	t.Run("no name", func(t *testing.T) {
		csp := cspVMFixture(t)
		csp.Name = ""
		got := refinedNodeFromCSP(csp)
		if got.Hostname != "i-0123456789abcdef0" || got.MachineId != "i-0123456789abcdef0" {
			t.Errorf("Hostname/MachineId = %q/%q, want the VM id", got.Hostname, got.MachineId)
		}
	})

	t.Run("gpu", func(t *testing.T) {
		csp := cspVMFixture(t)
		csp.VMSpecDetail.GPUs = []infra.CSPGPU{
			{Count: 2, Vendor: "NVIDIA", Model: "T4", MemoryGB: 16},
			{Count: 0, Vendor: "NVIDIA", Model: "A10G"},
		}
		want := []inframodel.GpuCardProperty{
			{DriverIndex: "0", Vendor: "NVIDIA", Model: "T4", Type: "GPU", MemoryTotalGB: 16},
			{DriverIndex: "1", Vendor: "NVIDIA", Model: "T4", Type: "GPU", MemoryTotalGB: 16},
			{DriverIndex: "2", Vendor: "NVIDIA", Model: "A10G", Type: "GPU"},
		}
		if got := refinedNodeFromCSP(csp).GPUCards; !reflect.DeepEqual(got, want) {
			t.Errorf("GPUCards =\n%+v\nwant\n%+v", got, want)
		}
	})

	t.Run("no image detail", func(t *testing.T) {
		csp := cspVMFixture(t)
		csp.ImageDetail = nil
		if got := refinedNodeFromCSP(csp).OS; got != (inframodel.OsProperty{}) {
			t.Errorf("OS = %+v, want empty for an AMI id", got)
		}
	})
}

func TestParseCSPOS(t *testing.T) {
	tests := []struct {
		desc     string
		id       string
		version  string
		codename string
	}{
		{"ubuntu-jammy-22.04", "ubuntu", "22.04", "jammy"},
		{"Ubuntu 22.04", "ubuntu", "22.04", ""},
		{"Ubuntu Server 24.04 LTS 64bit", "ubuntu", "24.04", ""},
		{"ubuntu/images/hvm-ssd/ubuntu-jammy-22.04-amd64-server-20240207", "ubuntu", "22.04", "jammy"},
		{"0001-com-ubuntu-server-jammy:22_04-lts-gen2", "ubuntu", "22.04", "jammy"},
		{"Canonical, Ubuntu, 22.04 LTS, amd64 jammy image built on 2024-01-10", "ubuntu", "22.04", "jammy"},
		{"ubuntu-2204-jammy-v20240110", "ubuntu", "22.04", "jammy"},
		{"ubuntu-noble-x86_64", "ubuntu", "24.04", "noble"},
		{"debian-12-bookworm-v20240110", "debian", "12", "bookworm"},
		{"Rocky Linux 8.10", "rocky", "8.10", ""},
		{"CentOS 7.9 64 bit", "centos", "7.9", ""},
		{"Red Hat Enterprise Linux 9.3", "rhel", "9.3", ""},
		{"amzn2-ami-hvm-2.0.20240131-x86_64-gp2", "amzn", "2", ""},
		{"Windows Server 2019 Datacenter", "windows", "2019", ""},
		{"ubuntu x86_64", "ubuntu", "", ""},
	}

	for _, tt := range tests {
		got, ok := parseCSPOS(tt.desc)
		if !ok || got.ID != tt.id || got.VersionID != tt.version || got.Version != tt.version ||
			got.VersionCodename != tt.codename {
			t.Errorf("parseCSPOS(%q) = %+v, %v, want id %q version %q codename %q",
				tt.desc, got, ok, tt.id, tt.version, tt.codename)
		}
	}

	for _, desc := range []string{"", "ami-0aaaabbbbccccdddd", "custom-golden-image-v3"} {
		if got, ok := parseCSPOS(desc); ok {
			t.Errorf("parseCSPOS(%q) = %+v, want no distribution", desc, got)
		}
	}
}

// When the agent collected the VM, the node is the agent's view and the CSP
// only adds its security group rules; nothing of the CSP data fills the rest.
func TestDoGetRefinedInfraInfo_AgentFirst(t *testing.T) {
	agent := hostInfra("guest-hostname")
	agent.Compute.OS.Node.Machineid = "guest-machine-id"
	agent.Compute.ComputeResource.CPU = infra.CPU{Cpus: 1, Cores: 4, Threads: 8}
	agent.Compute.ComputeResource.Memory.Size = 15990
	agent.CSP = cspVMFixture(t)

	got, err := doGetRefinedInfraInfo(agent)
	if err != nil {
		t.Fatalf("doGetRefinedInfraInfo: %v", err)
	}

	if got.Hostname != "guest-hostname" || got.MachineId != "guest-machine-id" {
		t.Errorf("Hostname/MachineId = %q/%q, want the agent's", got.Hostname, got.MachineId)
	}
	if got.CPU != (inframodel.CpuProperty{Cpus: 1, Cores: 4, Threads: 8}) || got.Memory.TotalSize != 16 {
		t.Errorf("CPU = %+v, Memory = %+v, want the agent's", got.CPU, got.Memory)
	}
	if got.RootDisk.TotalSize != 0 || got.OS.ID != "" || got.Interfaces != nil || got.DataDisks != nil {
		t.Errorf("CSP data leaked into an agent node: RootDisk %+v, OS %+v, Interfaces %+v, DataDisks %+v",
			got.RootDisk, got.OS, got.Interfaces, got.DataDisks)
	}
	if len(got.FirewallTable) != 1 || got.FirewallTable[0].DstPorts != "22" {
		t.Errorf("FirewallTable = %+v, want the security group rule", got.FirewallTable)
	}

	if hasAgentInfra(&infra.Infra{CSP: agent.CSP}) {
		t.Error("hasAgentInfra = true for an infra with only the csp section")
	}
	if !hasAgentInfra(agent) {
		t.Error("hasAgentInfra = false for an infra the agent collected")
	}
}

// A CSP VM whose CSP collection succeeded but whose agent call failed is a node
// of its CSP data, in the group and the single refined infra, and the agent
// data left from an earlier import is not used. A VM whose CSP collection
// failed stays out.
func TestRefinedInfra_CSPVMAfterAgentFailure(t *testing.T) {
	openTestDB(t)
	conns := []model.ConnectionInfo{
		testConn("vm-agent-failed", cspGroup, common.ResourceTypeVM, "198.51.100.30"),
		testConn("vm-csp-failed", cspGroup, common.ResourceTypeVM, "198.51.100.31"),
		testConn("vm-ok", cspGroup, common.ResourceTypeVM, "198.51.100.32"),
	}
	conns[0].ConnectionStatus = model.ConnectionInfoStatusSuccess
	conns[1].ConnectionStatus = model.ConnectionInfoStatusFailed
	conns[2].ConnectionStatus = model.ConnectionInfoStatusSuccess
	createGroup(t, cspGroup, conns)

	saveInfra(t, "vm-agent-failed", "failed", hostInfra("stale-agent-host"), cspVMData)
	saveInfra(t, "vm-csp-failed", "failed", nil, cspVMData)
	saveInfra(t, "vm-ok", "success", hostInfra("agent-host"), cspVMData)

	var ids []string
	for _, s := range refinedInfraSources(cspGroup, conns) {
		ids = append(ids, s.connID)
	}
	if want := []string{"vm-agent-failed", "vm-ok"}; !reflect.DeepEqual(ids, want) {
		t.Errorf("sources = %v, want %v", ids, want)
	}

	rec := groupRequest(t, cspGroup.ID, GetInfraInfoSourceGroupRefined)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var got model.RefinedOnpremiseInfraModel
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	var hostnames []string
	for _, n := range got.OnpremiseInfraModel.Nodes {
		hostnames = append(hostnames, n.Hostname)
	}
	if want := []string{"src-web-01", "agent-host"}; !reflect.DeepEqual(hostnames, want) {
		t.Fatalf("node hostnames = %q, want %q", hostnames, want)
	}
	if n := got.OnpremiseInfraModel.Nodes[0]; n.CPU.Threads != 8 || n.Memory.TotalSize != 16 || n.OS.ID != "ubuntu" {
		t.Errorf("node of the CSP data = %+v", n.NodeProperty)
	}

	single := connRequest(t, http.MethodGet, cspGroup.ID, "vm-agent-failed", GetInfraInfoRefined)
	if single.Code != http.StatusOK {
		t.Fatalf("single status = %d, body = %s", single.Code, single.Body.String())
	}
	var one model.RefinedOnpremiseInfraModel
	if err := json.Unmarshal(single.Body.Bytes(), &one); err != nil {
		t.Fatal(err)
	}
	if len(one.OnpremiseInfraModel.Nodes) != 1 || one.OnpremiseInfraModel.Nodes[0].Hostname != "src-web-01" ||
		one.OnpremiseInfraModel.Nodes[0].RootDisk.TotalSize != 50 {
		t.Errorf("single nodes = %+v, want the node of the CSP data", one.OnpremiseInfraModel.Nodes)
	}
}

func TestRefineManagedRDBMS_DisplayNameFallsBackToSystemId(t *testing.T) {
	got := refineManagedRDBMS(&model.RDBMSInfo{RDBMS: &spider.RDBMSInfo{
		IId:      spider.IID{SystemId: "db-abc123"},
		DBEngine: "mysql",
	}})
	if got.DisplayName != "db-abc123" {
		t.Errorf("DisplayName = %q, want the SystemId", got.DisplayName)
	}

	got = refineManagedRDBMS(&model.RDBMSInfo{RDBMS: &spider.RDBMSInfo{
		IId:      spider.IID{NameId: "billing-db", SystemId: "db-abc123"},
		DBEngine: "mysql",
	}})
	if got.DisplayName != "billing-db" {
		t.Errorf("DisplayName = %q, want the NameId", got.DisplayName)
	}
}
