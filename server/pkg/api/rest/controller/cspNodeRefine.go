package controller

import (
	"encoding/json"
	"net"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	inframodel "github.com/cloud-barista/cm-beetle/imdl/on-premise-model"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/infra"
	serverCommon "github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
)

// hasAgentInfra reports whether the infra holds anything the agent collected.
// doGetInfraInfo leaves everything but the csp section zero when the record has
// no agent data, which is the case of a CSP VM reached without the agent.
func hasAgentInfra(infraInfo *infra.Infra) bool {
	agentPart := *infraInfo
	agentPart.CSP = nil

	return !reflect.DeepEqual(agentPart, infra.Infra{})
}

// cspNodeAfterAgentFailure reports whether a CSP VM whose last infra import
// failed still becomes a node from its CSP data. The import of a CSP VM with an
// address collects from the CSP first and then calls the agent; when only the
// agent call fails, the record is failed but the CSP data it holds was just
// collected. connection_status follows the CSP collection, so it tells that
// case apart from a CSP collection that failed and left an older CSP data.
func cspNodeAfterAgentFailure(sourceGroup *model.SourceGroup, conn *model.ConnectionInfo, rec *model.SavedInfraInfo) bool {
	return serverCommon.IsCSPType(sourceGroup.Type) &&
		conn.ResourceType == serverCommon.ResourceTypeVM &&
		rec.Status == model.ConnectionInfoStatusFailed &&
		strings.TrimSpace(rec.CSPData) != "" &&
		conn.ConnectionStatus == model.ConnectionInfoStatusSuccess
}

// cspOnlyInfra reads the CSP data of a record into an infra without agent data.
// The agent data a record may still hold is from an earlier import, so it is
// left out rather than mixed with the CSP data of the latest one.
func cspOnlyInfra(cspData string) (*infra.Infra, error) {
	var csp infra.CSPInfo
	if err := json.Unmarshal([]byte(cspData), &csp); err != nil {
		return nil, err
	}

	return &infra.Infra{CSP: &csp}, nil
}

// refinedNodeFromCSP builds the refined node of a CSP VM from what the CSP
// reports about it, for a VM the agent did not collect. Only what the CSP knows
// is filled: the guest's routes, interface names and the like stay empty.
func refinedNodeFromCSP(csp *infra.CSPInfo) inframodel.NodeProperty {
	node := inframodel.NodeProperty{
		Hostname:  firstNonEmpty(csp.Name, csp.ID),
		MachineId: firstNonEmpty(csp.ID, csp.Name),
		RootDisk: inframodel.DiskProperty{
			Label:     csp.RootDisk.Name,
			Type:      csp.RootDisk.Type,
			TotalSize: uint64(csp.RootDisk.Size), // GB
		},
		DataDisks:     cspDataDisks(csp),
		Interfaces:    cspInterfaces(csp),
		FirewallTable: firewallRulesFromCSP(csp),
		OS:            cspOS(csp),
	}

	if spec := csp.VMSpecDetail; spec != nil {
		node.CPU = cspCPU(spec.VCPUs)
		node.Memory.TotalSize = mibToGiBCapacity(uint64(spec.MemoryMiB)) // GiB
		node.GPUCards = cspGPUCards(spec.GPUs)
		node.CPU.Architecture = spec.Architecture
	}
	if node.CPU.Architecture == "" && csp.ImageDetail != nil {
		node.CPU.Architecture = strings.ToLower(csp.ImageDetail.OSArchitecture)
	}

	return node
}

// cspCPU gives a VM's vCPU count the CPU shape cm-beetle sizes a target from:
// it takes Cpus x Threads as the vCPU count and falls back from Threads to
// Cores, so one socket with as many cores and threads as vCPUs reads as that
// many vCPUs on every path. The CSP does not tell how its vCPUs map to physical
// cores, so cores are not halved for hyper-threading.
func cspCPU(vcpus uint) inframodel.CpuProperty {
	if vcpus == 0 || uint64(vcpus) > uint64(^uint32(0)) {
		return inframodel.CpuProperty{}
	}

	return inframodel.CpuProperty{
		Cpus:    1,
		Cores:   uint32(vcpus),
		Threads: uint32(vcpus),
	}
}

// cspDataDisks lists the data disks whose size the CSP reported. A disk named
// on the VM but not resolved has no size, and a disk of 0 GB would be a wrong
// reading rather than a missing one, so it is left out.
func cspDataDisks(csp *infra.CSPInfo) []inframodel.DiskProperty {
	var disks []inframodel.DiskProperty
	for _, d := range csp.DataDiskDetails {
		if d.Size == 0 {
			continue
		}
		disks = append(disks, inframodel.DiskProperty{
			Label:     firstNonEmpty(d.Name, d.ID),
			Type:      d.Type,
			TotalSize: uint64(d.Size), // GB
		})
	}

	return disks
}

// cspInterfaces gives the VM one interface holding its private IP, with the
// prefix of the subnet the VM is in when the CSP data has it, and its public
// IP. The guest's interface name is not known to the CSP and stays empty.
func cspInterfaces(csp *infra.CSPInfo) []inframodel.NetworkInterfaceProperty {
	iface := inframodel.NetworkInterfaceProperty{
		IPv4CidrBlocks: []string{},
		IPv6CidrBlocks: []string{},
	}

	add := func(ip net.IP, prefix int) {
		if ip.To4() != nil {
			if prefix < 0 {
				prefix = 32
			}
			iface.IPv4CidrBlocks = append(iface.IPv4CidrBlocks, ip.String()+"/"+strconv.Itoa(prefix))
			return
		}
		if prefix < 0 {
			prefix = 128
		}
		iface.IPv6CidrBlocks = append(iface.IPv6CidrBlocks, ip.String()+"/"+strconv.Itoa(prefix))
	}

	if ip := net.ParseIP(strings.TrimSpace(csp.PrivateIP)); ip != nil {
		add(ip, cspSubnetPrefix(csp, ip))
	}
	if ip := net.ParseIP(strings.TrimSpace(csp.PublicIP)); ip != nil {
		add(ip, -1)
	}

	if len(iface.IPv4CidrBlocks) == 0 && len(iface.IPv6CidrBlocks) == 0 {
		return nil
	}

	return []inframodel.NetworkInterfaceProperty{iface}
}

// cspSubnetPrefix returns the prefix length of the VM's subnet when the subnet
// is listed in the VPC detail and holds ip, or -1 when it is not known.
func cspSubnetPrefix(csp *infra.CSPInfo, ip net.IP) int {
	for _, sn := range csp.Network.VPC.Subnets {
		if csp.Network.Subnet == "" || sn.Name != csp.Network.Subnet {
			continue
		}
		_, ipNet, err := net.ParseCIDR(strings.TrimSpace(sn.CIDR))
		if err != nil || !ipNet.Contains(ip) {
			continue
		}
		prefix, _ := ipNet.Mask.Size()
		return prefix
	}

	return -1
}

// cspGPUCards lists one card per GPU device of the spec. An entry whose count
// the CSP does not expose still stands for at least one device.
func cspGPUCards(gpus []infra.CSPGPU) []inframodel.GpuCardProperty {
	var cards []inframodel.GpuCardProperty
	for _, g := range gpus {
		count := g.Count
		if count == 0 {
			count = 1
		}
		for i := uint(0); i < count; i++ {
			cards = append(cards, inframodel.GpuCardProperty{
				DriverIndex:   strconv.Itoa(len(cards)),
				Vendor:        g.Vendor,
				Model:         g.Model,
				Type:          "GPU",
				MemoryTotalGB: float32(g.MemoryGB),
			})
		}
	}

	return cards
}

// cspOSFamily is a distribution as /etc/os-release names it (ID, NAME), with
// the words a CSP image description uses for it and the release codenames that
// give its version when the description has no number.
type cspOSFamily struct {
	id        string
	name      string
	keys      []string
	codenames []cspOSCodename
}

type cspOSCodename struct {
	name    string
	version string
}

var cspOSFamilies = []cspOSFamily{
	{id: "ubuntu", name: "Ubuntu", keys: []string{"ubuntu"}, codenames: []cspOSCodename{
		{"bionic", "18.04"}, {"focal", "20.04"}, {"jammy", "22.04"}, {"noble", "24.04"},
	}},
	{id: "debian", name: "Debian GNU/Linux", keys: []string{"debian"}, codenames: []cspOSCodename{
		{"buster", "10"}, {"bullseye", "11"}, {"bookworm", "12"}, {"trixie", "13"},
	}},
	{id: "rocky", name: "Rocky Linux", keys: []string{"rocky"}},
	{id: "almalinux", name: "AlmaLinux", keys: []string{"almalinux", "alma linux"}},
	{id: "centos", name: "CentOS Linux", keys: []string{"centos"}},
	{id: "rhel", name: "Red Hat Enterprise Linux", keys: []string{"red hat", "redhat", "rhel"}},
	{id: "ol", name: "Oracle Linux Server", keys: []string{"oracle linux", "oraclelinux"}},
	{id: "amzn", name: "Amazon Linux", keys: []string{"amazon linux", "amzn"}},
	{id: "sles", name: "SLES", keys: []string{"sles", "suse"}},
	{id: "fedora", name: "Fedora Linux", keys: []string{"fedora"}},
	{id: "windows", name: "Windows", keys: []string{"windows"}},
}

// cspOSVersion is the first number in an image description that stands on its
// own ("22.04", "22_04", "7.9", "2019"), so the digits of "amd64", "64bit" or a
// "v20240110" build stamp are not taken for it. Only major and minor are kept,
// as VERSION_ID has them.
var cspOSVersion = regexp.MustCompile(`(?:^|[^0-9a-z])(\d+)(?:[._](\d+))?(?:[._]\d+)*(?:$|[^0-9a-z])`)

// cspOSArch is the architecture names an image description may hold. They are
// blanked before the version is read, since "x86_64" splits into numbers that
// stand on their own.
var cspOSArch = regexp.MustCompile(`x86[_-]64|x86|amd64|arm64|aarch64|i[36]86`)

// cspOS reads the OS of the image the VM was created from. The CSP describes it
// in free text that differs per driver ("Ubuntu 22.04", "ubuntu-jammy-22.04",
// an Azure offer and SKU, an image name), so the distribution and its version
// are picked out of it; cm-beetle looks the target image up by ID and
// VersionID. The description itself is kept as the pretty name.
func cspOS(csp *infra.CSPInfo) inframodel.OsProperty {
	var candidates []string
	var pretty string
	if img := csp.ImageDetail; img != nil {
		pretty = strings.TrimSpace(img.OSDistribution)
		candidates = append(candidates, img.OSDistribution, img.Name)
	}
	candidates = append(candidates, csp.Image)

	for _, c := range candidates {
		osp, ok := parseCSPOS(c)
		if !ok {
			continue
		}
		osp.PrettyName = firstNonEmpty(pretty, strings.TrimSpace(c))
		return osp
	}

	return inframodel.OsProperty{PrettyName: pretty}
}

// parseCSPOS picks the distribution and version out of an image description.
// The distribution whose name appears first wins, and the version is the first
// number after it.
func parseCSPOS(desc string) (inframodel.OsProperty, bool) {
	s := strings.ToLower(desc)

	var family *cspOSFamily
	at, end := -1, 0
	for i := range cspOSFamilies {
		for _, key := range cspOSFamilies[i].keys {
			if p := strings.Index(s, key); p >= 0 && (at < 0 || p < at) {
				family, at, end = &cspOSFamilies[i], p, p+len(key)
			}
		}
	}
	if family == nil {
		return inframodel.OsProperty{}, false
	}

	osp := inframodel.OsProperty{ID: family.id, Name: family.name}

	if m := cspOSVersion.FindStringSubmatch(cspOSArch.ReplaceAllString(s[end:], " ")); m != nil {
		osp.VersionID = m[1]
		if m[2] != "" {
			osp.VersionID += "." + m[2]
		} else if family.id == "ubuntu" && len(m[1]) == 4 {
			// GCP names Ubuntu images "ubuntu-2204-..."; the release is YY.MM.
			osp.VersionID = m[1][:2] + "." + m[1][2:]
		}
	}

	for _, c := range family.codenames {
		if !strings.Contains(s, c.name) {
			continue
		}
		osp.VersionCodename = c.name
		if osp.VersionID == "" {
			osp.VersionID = c.version
		}
		break
	}

	osp.Version = osp.VersionID

	return osp, true
}
