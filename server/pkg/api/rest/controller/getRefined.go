package controller

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"

	softwaremodel "github.com/cloud-barista/cm-grasshopper/smdl"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/infra"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/kubernetes"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/network"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/software"
	"github.com/docker/docker/api/types/container"
	"github.com/jollaman999/utils/logger"

	inframodel "github.com/cloud-barista/cm-beetle/imdl/on-premise-model"
	"github.com/cloud-barista/cm-honeybee/server/dao"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/common"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"github.com/labstack/echo/v4"
)

func netmaskToCIDR(netmask string) (int, error) {
	ip := net.ParseIP(netmask)
	if ip == nil {
		return 0, fmt.Errorf("invalid netmask: %s", netmask)
	}

	mask := net.IPMask(ip.To4())
	if mask == nil {
		return 0, fmt.Errorf("invalid netmask: %s", netmask)
	}

	cidr, _ := mask.Size()

	return cidr, nil
}

func convertRouteToRefinedRoute(route *network.Route, gateway string) (*inframodel.RouteProperty, error) {
	var destination string

	switch route.Family {
	case "ipv4":
		cidr, err := netmaskToCIDR(route.Netmask)
		if err != nil {
			return nil, err
		}
		destination = route.Destination + "/" + strconv.Itoa(cidr)
	case "ipv6":
		destination = route.Destination + route.Netmask
	default:
		return nil, fmt.Errorf("invalid route family: %s", route.Family)
	}

	routeProperty := inframodel.RouteProperty{
		Destination: destination,
		Gateway:     gateway,
		Interface:   route.Interface,
		Metric:      route.Metric,
		Protocol:    route.Proto,
		Scope:       route.Scope,
		Source:      route.Source,
		LinkState:   route.Link,
	}

	return &routeProperty, nil
}

// mibToGiB rounds a mebibyte reading to the gibibytes the refined model
// carries. Truncating instead reports a 1 GiB machine as 0, because the OS
// only ever sees 1020-ish MiB of it once the kernel's reserved pages are
// taken out, and 1020/1024 is 0 in integer arithmetic.
func mibToGiB(mib uint64) uint64 {
	return (mib + 512) / 1024
}

// mibToGiBCapacity is mibToGiB for a total-capacity field. A 0 there is not a
// small number, it is a missing one: a node advertising 0 GiB of memory or of
// root disk drops out of every `memoryGiB >= ...` target-spec filter cm-beetle
// matches against, so a machine that has some capacity reports at least 1.
func mibToGiBCapacity(mib uint64) uint64 {
	if gib := mibToGiB(mib); gib > 0 {
		return gib
	}

	if mib > 0 {
		return 1
	}

	return 0
}

func doGetRefinedInfraInfo(infraInfo *infra.Infra) (*inframodel.NodeProperty, error) {
	var dataDisks []inframodel.DiskProperty

	for _, dataDisk := range infraInfo.Compute.ComputeResource.DataDisk {
		dataDisks = append(dataDisks, inframodel.DiskProperty{
			Label:     dataDisk.Label,
			Type:      dataDisk.Type,
			TotalSize: uint64(dataDisk.Size),
			Available: uint64(dataDisk.Available),
			Used:      uint64(dataDisk.Used),
		})
	}

	var interfaces []inframodel.NetworkInterfaceProperty

	for _, iface := range infraInfo.Network.Host.NetworkInterface {
		interf := inframodel.NetworkInterfaceProperty{
			Name:           iface.Interface,
			MacAddress:     iface.MACAddress,
			IPv4CidrBlocks: []string{},
			IPv6CidrBlocks: []string{},
			Mtu:            iface.MTU,
		}

		for _, address := range iface.Address {
			split := strings.Split(address, "/")
			if len(split) != 2 {
				continue
			}

			validIPv4 := net.ParseIP(split[0]).To4()
			if validIPv4 != nil {
				interf.IPv4CidrBlocks = append(interf.IPv4CidrBlocks, address)
			} else {
				interf.IPv6CidrBlocks = append(interf.IPv6CidrBlocks, address)
			}
		}

		for _, route := range infraInfo.Network.Host.Route {
			if iface.Interface == route.Interface {
				interf.State = route.Link
				break
			}
		}

		interfaces = append(interfaces, interf)
	}

	var routingTable []inframodel.RouteProperty

	for _, route := range infraInfo.Network.Host.Route {
		var gateway string

		for _, iface := range infraInfo.Network.Host.NetworkInterface {
			if iface.Interface == route.Interface {
				gateway = iface.Gateway
				break
			}
		}
		refinedRoute, err := convertRouteToRefinedRoute(&route, gateway)
		if err != nil {
			logger.Println(logger.WARN, true, err.Error())
			continue
		}

		routingTable = append(routingTable, *refinedRoute)
	}

	var firewallTable []inframodel.FirewallRuleProperty

	for _, firewall := range infraInfo.Network.Host.FirewallRule {
		firewallTable = append(firewallTable, inframodel.FirewallRuleProperty{
			SrcCIDR:   firewall.Src,
			DstCIDR:   firewall.Dst,
			SrcPorts:  firewall.SrcPorts,
			DstPorts:  firewall.DstPorts,
			Protocol:  firewall.Protocol,
			Direction: firewall.Direction,
			Action:    firewall.Action,
		})
	}

	refinedInfraInfo := inframodel.NodeProperty{
		Hostname:  infraInfo.Compute.OS.Node.Hostname,
		MachineId: infraInfo.Compute.OS.Node.Machineid,
		CPU: inframodel.CpuProperty{
			Architecture: infraInfo.Compute.OS.Kernel.Architecture,
			Cpus:         uint32(infraInfo.Compute.ComputeResource.CPU.Cpus),
			Cores:        uint32(infraInfo.Compute.ComputeResource.CPU.Cores),
			Threads:      uint32(infraInfo.Compute.ComputeResource.CPU.Threads),
			MaxSpeed:     float32(infraInfo.Compute.ComputeResource.CPU.MaxSpeed) / 1000, // GHz
			Vendor:       infraInfo.Compute.ComputeResource.CPU.Vendor,
			Model:        infraInfo.Compute.ComputeResource.CPU.Model,
		},
		Memory: inframodel.MemoryProperty{
			Type:      infraInfo.Compute.ComputeResource.Memory.Type,
			TotalSize: mibToGiBCapacity(uint64(infraInfo.Compute.ComputeResource.Memory.Size)), // GiB
			Available: mibToGiB(uint64(infraInfo.Compute.ComputeResource.Memory.Available)),    // GiB
			Used:      mibToGiB(uint64(infraInfo.Compute.ComputeResource.Memory.Used)),         // GiB
		},
		RootDisk: inframodel.DiskProperty{
			Label:     infraInfo.Compute.ComputeResource.RootDisk.Label,
			Type:      infraInfo.Compute.ComputeResource.RootDisk.Type,
			TotalSize: uint64(infraInfo.Compute.ComputeResource.RootDisk.Size),      // GiB
			Available: uint64(infraInfo.Compute.ComputeResource.RootDisk.Available), // GiB
			Used:      uint64(infraInfo.Compute.ComputeResource.RootDisk.Used),      // GiB
		},
		DataDisks:     dataDisks,
		Interfaces:    interfaces,
		RoutingTable:  routingTable,
		FirewallTable: firewallTable,
		OS: inframodel.OsProperty{
			PrettyName:      infraInfo.Compute.OS.OS.PrettyName,
			Version:         infraInfo.Compute.OS.OS.Version,
			Name:            infraInfo.Compute.OS.OS.Name,
			VersionID:       infraInfo.Compute.OS.OS.VersionID,
			VersionCodename: infraInfo.Compute.OS.OS.VersionCodename,
			ID:              infraInfo.Compute.OS.OS.ID,
			IDLike:          infraInfo.Compute.OS.OS.IDLike,
		},
		GPUCards: gpuCardsFromInfra(infraInfo.GPU),
	}

	return &refinedInfraInfo, nil
}

// mibToGB converts a mebibyte reading into the gigabytes the refined model
// declares (`MemoryTotalGB` and friends). The divisor is 1024, not 1e9/1048576:
// accelerator memory is sized and catalogued in binary units even though the
// field says GB, so an A100 with 40 GiB of HBM2 is listed as 40 both by NVIDIA
// and by the target spec catalogues cm-beetle matches against. Converting to
// decimal gigabytes would make the same card 42.95 and push it past the
// `acceleratorMemoryGB >= ...` filter that is meant to select it.
//
// The reading itself is the usable framebuffer the driver reports, which is
// below the capacity the card is marketed with on cards that reserve some (a
// "16GB" Tesla T4 reports 15360 MiB). It is passed through as measured rather
// than rounded up to the marketed figure, which would be a guess.
func mibToGB(mib uint64) float32 {
	return float32(mib) / 1024
}

// gpuCardsFromInfra flattens the per-vendor GPU collections into the single
// card list the refined model expects. One physical device is one entry, so a
// node holding different models keeps each of them intact.
//
// Memory is passed through as the driver reports it. A card with ECC enabled
// reports less than its marketed capacity, and the shortfall is not added back
// here: the consumer is told the state through ECCEnabled and MemoryReservedGB
// and corrects for it itself, so MemoryTotalGB stays a reading rather than a
// value whose provenance has to be guessed at.
func gpuCardsFromInfra(gpu infra.GPU) []inframodel.GpuCardProperty {
	cards := make([]inframodel.GpuCardProperty, 0, len(gpu.NVIDIA)+len(gpu.AMD))

	for _, n := range gpu.NVIDIA {
		card := inframodel.GpuCardProperty{
			DriverIndex:   strconv.Itoa(n.DeviceAttribute.Index),
			Uuid:          n.DeviceAttribute.GPUUUID,
			Vendor:        "NVIDIA",
			Model:         n.DeviceAttribute.ProductName,
			Type:          "GPU",
			Architecture:  n.DeviceAttribute.ProductArchitecture,
			DriverVersion: n.DeviceAttribute.DriverVersion,
			CudaVersion:   n.DeviceAttribute.CUDAVersion,
			PciBusId:      n.DeviceAttribute.PCIBusID,
			ECCEnabled:    n.ECC != nil && strings.EqualFold(n.ECC.Mode, "Enabled"),
		}

		// A nil reading means nvidia-smi did not report the value, which is not
		// the same as zero, so it is left out rather than converted.
		if n.Performance.FBMemoryTotal != nil {
			card.MemoryTotalGB = mibToGB(*n.Performance.FBMemoryTotal)
		}
		if n.Performance.FBMemoryReserved != nil {
			card.MemoryReservedGB = mibToGB(*n.Performance.FBMemoryReserved)
		}
		if n.Performance.FBMemoryFree != nil {
			card.MemoryFreeGB = mibToGB(*n.Performance.FBMemoryFree)
		}
		if n.Performance.FBMemoryUsed != nil {
			card.MemoryUsedGB = mibToGB(*n.Performance.FBMemoryUsed)
		}

		cards = append(cards, card)
	}

	for _, a := range gpu.AMD {
		card := inframodel.GpuCardProperty{
			DriverIndex:   a.DeviceAttribute.Card,
			Uuid:          a.DeviceAttribute.GPUUUID,
			Vendor:        "AMD",
			Model:         a.DeviceAttribute.ProductName,
			Type:          "GPU",
			DriverVersion: a.DeviceAttribute.DriverVersion,
			PciBusId:      a.DeviceAttribute.PCIBusID,
		}

		if a.Performance.VRAMMemoryTotal != nil {
			card.MemoryTotalGB = mibToGB(*a.Performance.VRAMMemoryTotal)
		}
		if a.Performance.VRAMMemoryUsed != nil {
			card.MemoryUsedGB = mibToGB(*a.Performance.VRAMMemoryUsed)
		}
		// rocm-smi reports neither a free nor a reserved figure, and deriving
		// either from the two above would put a computed number in a field the
		// consumer reads as measured, so both stay unset. It does not report
		// the ECC state either.

		cards = append(cards, card)
	}

	if len(cards) == 0 {
		return nil
	}

	return cards
}

// gpuCardsFromK8s turns the GPU extended resources a node advertises into card
// entries. Kubernetes counts devices rather than describing them, so one
// resource with a capacity of N becomes N identical entries carrying whatever
// the feature-discovery labels said. UUID and PCI address stay empty because
// the cluster API never reports them.
func gpuCardsFromK8s(gpus []kubernetes.NodeGPU) []inframodel.GpuCardProperty {
	var cards []inframodel.GpuCardProperty

	for _, g := range gpus {
		for i := int64(0); i < g.Capacity; i++ {
			card := inframodel.GpuCardProperty{
				DriverIndex:   strconv.Itoa(len(cards)),
				Vendor:        strings.ToUpper(g.Vendor),
				Model:         g.Product,
				Type:          "GPU",
				DriverVersion: g.DriverVersion,
			}
			if g.Memory > 0 {
				card.MemoryTotalGB = mibToGB(uint64(g.Memory))
			}

			cards = append(cards, card)
		}
	}

	return cards
}

// tryGetKubernetesInfo returns the collected Kubernetes information of the
// connection, or nil if the connection has no Kubernetes data. Unlike
// doGetKubernetesInfo, missing data is not treated as an error because most
// connections are not Kubernetes nodes.
func tryGetKubernetesInfo(connID string) *kubernetes.Kubernetes {
	savedKubernetesInfo, err := dao.SavedKubernetesInfoGet(connID)
	if err != nil {
		return nil
	}

	var kubernetesInfo kubernetes.Kubernetes
	err = json.Unmarshal([]byte(savedKubernetesInfo.KubernetesData), &kubernetesInfo)
	if err != nil {
		logger.Println(logger.WARN, false, "Error occurred while parsing kubernetes information."+
			" (ConnectionID = "+connID+")")
		return nil
	}

	return &kubernetesInfo
}

// k8sNodeInfoString reads a string field from a Kubernetes node's NodeInfo
// (the raw status.nodeInfo map), returning "" when absent.
func k8sNodeInfoString(nodeInfo interface{}, key string) string {
	m, ok := nodeInfo.(map[string]interface{})
	if !ok {
		return ""
	}
	s, _ := m[key].(string)
	return s
}

// normalizeUUID lowercases and strips hyphens so a machine/system UUID compares
// equal regardless of formatting (e.g. "34C7FF99-E46C-..." == "34c7ff99e46c...").
func normalizeUUID(s string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(s), "-", ""))
}

// buildK8sCluster builds the refined cluster property from the collected
// Kubernetes information, falling back to a node's kubelet version when the
// cluster version was not collected directly.
func buildK8sCluster(k8sInfo *kubernetes.Kubernetes) *inframodel.K8sClusterProperty {
	version := k8sInfo.Cluster.Version
	if version == "" {
		for _, node := range k8sInfo.Nodes {
			if kubeletVersion := k8sNodeInfoString(node.NodeInfo, "kubeletVersion"); kubeletVersion != "" {
				version = strings.TrimPrefix(kubeletVersion, "v")
				break
			}
		}
	}

	name := k8sInfo.Cluster.Name
	if name == "" {
		name = "kubernetes" // kubeadm default cluster name
	}

	return &inframodel.K8sClusterProperty{
		Name:          name,
		Version:       version,
		PodCIDR:       k8sInfo.Cluster.PodCIDR,
		ServiceCIDR:   k8sInfo.Cluster.ServiceCIDR,
		CNIPlugin:     k8sInfo.Cluster.CNIPlugin,
		NodePortRange: k8sInfo.Cluster.NodePortRange,
	}
}

// buildNodeFromK8s maps a single Kubernetes node (enumerated via the cluster
// API) into a refined NodeProperty. It is used for cluster nodes that have no
// matching host-level (SSH) collection, so only the K8s-derived fields are
// populated; node_spec sizes are collected in MiB and converted to GiB.
func buildNodeFromK8s(node kubernetes.Node) inframodel.NodeProperty {
	hostname, _ := node.Name.(string)

	// Use systemUUID (the DMI product UUID) as the machine id so it lines up with
	// what SSH collection records; fall back to machineID (/etc/machine-id).
	machineID := k8sNodeInfoString(node.NodeInfo, "systemUUID")
	if machineID == "" {
		machineID = k8sNodeInfoString(node.NodeInfo, "machineID")
	}

	return inframodel.NodeProperty{
		Hostname:  hostname,
		MachineId: machineID,
		Role:      string(node.Type),
		CPU: inframodel.CpuProperty{
			Architecture: k8sNodeInfoString(node.NodeInfo, "architecture"),
			Threads:      uint32(node.NodeSpec.CPU),
		},
		Memory: inframodel.MemoryProperty{
			TotalSize: mibToGiBCapacity(uint64(node.NodeSpec.Memory)), // MiB -> GiB
		},
		RootDisk: inframodel.DiskProperty{
			TotalSize: mibToGiBCapacity(uint64(node.NodeSpec.EphemeralStorage)), // MiB -> GiB
		},
		GPUCards: gpuCardsFromK8s(node.NodeSpec.GPU),
	}
}

// mergeK8sNodes reflects the collected Kubernetes cluster nodes into the refined
// node list. A node whose machine ID matches a host-level (SSH) collected node
// only contributes its role (the richer host data is kept); cluster nodes
// without a host-level counterpart (e.g. workers reachable only via the API)
// are appended from the K8s data. Host-level nodes that are not part of the
// cluster are marked as standalone.
func mergeK8sNodes(nodes []inframodel.NodeProperty, k8sInfo *kubernetes.Kubernetes) []inframodel.NodeProperty {
	index := make(map[string]int)
	for i := range nodes {
		if id := normalizeUUID(nodes[i].MachineId); id != "" {
			index[id] = i
		}
	}

	for _, node := range k8sInfo.Nodes {
		// SSH collection records the host's DMI system UUID as the machine id,
		// which corresponds to the K8s node's systemUUID — NOT its machineID
		// (/etc/machine-id, a different value). Match on systemUUID first, then
		// fall back to machineID, comparing normalized (hyphen/case-insensitive).
		matched := false
		for _, key := range []string{"systemUUID", "machineID"} {
			id := normalizeUUID(k8sNodeInfoString(node.NodeInfo, key))
			if id == "" {
				continue
			}
			if i, ok := index[id]; ok {
				nodes[i].Role = string(node.Type)
				if nodes[i].CPU.Architecture == "" {
					nodes[i].CPU.Architecture = k8sNodeInfoString(node.NodeInfo, "architecture")
				}
				matched = true
				break
			}
		}

		if !matched {
			nodes = append(nodes, buildNodeFromK8s(node))
		}
	}

	// Host-level nodes that did not match any cluster node are standalone.
	for i := range nodes {
		if nodes[i].Role == "" {
			nodes[i].Role = "standalone"
		}
	}

	return nodes
}

func doGetRefinedNetworkInfo(networkProperty *inframodel.NetworkProperty, routes *[]network.Route, machineID *string) {
	for _, route := range *routes {
		if strings.ToLower(route.Interface) == "lo" {
			continue
		}

		switch route.Family {
		case "ipv4":
			if route.Destination == "0.0.0.0" && route.Netmask == "0.0.0.0" {
				var gatewayProperty inframodel.GatewayProperty

				gatewayProperty.IP = route.NextHop
				gatewayProperty.InterfaceName = route.Interface
				gatewayProperty.MachineId = *machineID

				networkProperty.IPv4Networks.DefaultGateways = append(networkProperty.IPv4Networks.DefaultGateways, gatewayProperty)
			}
		case "ipv6":
			if route.Destination == "::" && route.Netmask == "/0" {
				var gatewayProperty inframodel.GatewayProperty

				gatewayProperty.IP = route.NextHop
				gatewayProperty.InterfaceName = route.Interface
				gatewayProperty.MachineId = *machineID

				networkProperty.IPv6Networks.DefaultGateways = append(networkProperty.IPv6Networks.DefaultGateways, gatewayProperty)
			}
		}
	}
}

func convertToPackages(packages interface{}) []softwaremodel.Package {
	var result []softwaremodel.Package

	switch p := packages.(type) {
	case []software.DEB:
		for _, pkg := range p {
			result = append(result, softwaremodel.Package{
				Name:    pkg.Package,
				Type:    softwaremodel.SoftwarePackageTypeDEB,
				Version: pkg.Version,
			})
		}
	case []software.RPM:
		for _, pkg := range p {
			result = append(result, softwaremodel.Package{
				Name:    pkg.Name,
				Type:    softwaremodel.SoftwarePackageTypeRPM,
				Version: pkg.Version,
			})
		}
	case []software.Snap:
		for _, pkg := range p {
			result = append(result, softwaremodel.Package{
				Name:        pkg.Name,
				Type:        softwaremodel.SoftwarePackageTypeSnap,
				Version:     pkg.Version,
				Channel:     pkg.Tracking,
				Revision:    pkg.Revision,
				Confinement: pkg.Confinement,
				Base:        pkg.Base,
				BlobPath:    pkg.BlobPath,
			})
		}
	case []software.Flatpak:
		for _, pkg := range p {
			name := pkg.ApplicationID
			if name == "" {
				name = pkg.Name
			}
			result = append(result, softwaremodel.Package{
				Name:          name,
				Type:          softwaremodel.SoftwarePackageTypeFlatpak,
				Version:       pkg.Version,
				Origin:        pkg.Origin,
				OriginURL:     pkg.OriginURL,
				ApplicationID: pkg.ApplicationID,
				Runtime:       pkg.Runtime,
				Branch:        pkg.Branch,
				Scope:         pkg.Installation,
			})
		}
	}

	return result
}

// appendUniquePath appends p to paths if it is a non-empty absolute path not
// already present.
func appendUniquePath(paths []string, p string) []string {
	p = strings.TrimSpace(p)
	if p == "" || !strings.HasPrefix(p, "/") {
		return paths
	}
	for _, existing := range paths {
		if existing == p {
			return paths
		}
	}
	return append(paths, p)
}

// appRefinement is the migration target re-derived for a recognized application:
// a friendly name, the install root to copy, and any extra data directories. Empty
// fields leave the corresponding raw value (process name / executable) unchanged.
type appRefinement struct {
	name       string
	binaryPath string
	dataDirs   []string
}

// appRefiner inspects a raw collected binary and, if it recognizes the
// application it belongs to, returns how to refine it (ok == true).
type appRefiner func(b software.Binary) (appRefinement, bool)

// appRefiners is the ordered list of application detectors. The first one that
// matches a process wins. Support a new interpreter-hosted application (e.g. a
// Python/Node service) by adding a refiner here — convertToBinaries needs no
// change.
var appRefiners = []appRefiner{
	tomcatRefiner,
	wineRefiner,
}

// tomcatRefiner recognizes a Tomcat instance (raw process name is just "java")
// from its catalina.home/base, and migrates the install dir. A separate
// catalina.base instance dir (multi-instance Tomcat) is carried as a data dir.
func tomcatRefiner(b software.Binary) (appRefinement, bool) {
	home, base := extractCatalinaPaths(b.CmdlineSlice)
	if home == "" && base == "" {
		return appRefinement{}, false
	}

	root := home
	if root == "" {
		root = base
	}
	r := appRefinement{name: "tomcat", binaryPath: root}
	if base != "" && base != home {
		r.dataDirs = []string{base}
	}
	return r, true
}

// wineRefiner migrates a Wine application by its WINEPREFIX bottle, which holds
// the app, registry and config.
func wineRefiner(b software.Binary) (appRefinement, bool) {
	if b.IsWine && b.WinePrefix != "" {
		return appRefinement{binaryPath: b.WinePrefix}, true
	}
	return appRefinement{}, false
}

// extractCatalinaPaths derives Tomcat's install (catalina.home) and instance
// (catalina.base) directories from a JVM process command line. Either may be "".
func extractCatalinaPaths(cmdline []string) (home string, base string) {
	for _, arg := range cmdline {
		if v, ok := strings.CutPrefix(arg, "-Dcatalina.home="); ok {
			home = strings.TrimSpace(v)
		} else if v, ok := strings.CutPrefix(arg, "-Dcatalina.base="); ok {
			base = strings.TrimSpace(v)
		}
	}
	return home, base
}

// convertToBinaries maps the raw collected legacy binaries (process-level info
// gathered on the source host) into the refined software model consumed by the
// migration tools. Launch provenance (systemd vs command) is carried through so
// the target can faithfully reproduce how the software was started.
//
// When a process is an interpreter/runtime hosting an application (e.g. a JVM
// running Tomcat, or Wine), the raw executable is not the right migration target.
// The per-application refiners in appRefiners re-derive the real install root,
// a recognizable name and any extra data directories; convertToBinaries itself
// stays generic. The runtime (JDK, Wine, ...) is carried as a dependency.
func convertToBinaries(legacy []software.Binary) []softwaremodel.Binary {
	var result []softwaremodel.Binary

	for _, b := range legacy {
		var configs []string
		for _, cf := range b.ConfigFiles {
			if cf.Path != "" {
				configs = append(configs, cf.Path)
			}
		}

		binaryPath := b.ExecutablePath
		name := b.Name
		// Dependencies are already filtered to non-package-owned paths by the agent
		// (package-provided runtimes are handled by package migration).
		neededLibraries := append([]string{}, b.Dependencies...)
		dataDirs := append([]string{}, b.DataDirs...)

		// Apply the first application refiner that recognizes this process, so an
		// interpreter-hosted app (Tomcat, Wine, ...) is migrated by its install
		// root rather than the interpreter binary.
		for _, refine := range appRefiners {
			r, ok := refine(b)
			if !ok {
				continue
			}
			if r.name != "" {
				name = r.name
			}
			if r.binaryPath != "" {
				binaryPath = r.binaryPath
			}
			for _, d := range r.dataDirs {
				dataDirs = appendUniquePath(dataDirs, d)
			}
			break
		}

		result = append(result, softwaremodel.Binary{
			Name:             name,
			Version:          b.Version,
			UIDs:             b.UIDs,
			GIDs:             b.GIDs,
			CmdlineSlice:     b.CmdlineSlice,
			Envs:             migrationEnvs(b),
			NeededLibraries:  neededLibraries,
			RequiredPackages: b.RequiredPackages,
			BinaryPath:       binaryPath,
			CustomDataPaths:  dataDirs,
			CustomConfigs:    configs,
			IsWine:           b.IsWine,
			WinePrefix:       b.WinePrefix,
			LaunchType:       b.LaunchType,
			SystemdUnitName:  b.SystemdUnitName,
			SystemdUnitPath:  b.SystemdUnitPath,
			SystemdEnabled:   b.SystemdEnabled,
			WorkingDirectory: b.WorkingDirectory,
			ServiceType:      b.ServiceType,
			PIDFile:          b.PIDFile,
		})
	}

	return result
}

func getContainerName(summary *container.Summary) string {
	if len(summary.Names) > 0 {
		return strings.TrimPrefix(summary.Names[0], "/")
	}
	return ""
}

func getDockerComposePath(labels map[string]string) string {
	if workingDir, ok := labels["com.docker.compose.project.config_files"]; ok {
		return workingDir
	}
	return ""
}

func getImageName(image *string) string {
	parts := strings.Split(*image, ":")
	if len(parts) > 0 {
		return parts[0]
	}
	return *image
}

func getImageTag(image *string) string {
	parts := strings.Split(*image, ":")
	if len(parts) > 1 {
		return parts[1]
	}
	return "latest"
}

func convertPorts(ports *[]container.Port) []softwaremodel.ContainerPort {
	var result []softwaremodel.ContainerPort

	for _, port := range *ports {
		result = append(result, softwaremodel.ContainerPort{
			ContainerPort: int(port.PrivatePort),
			HostPort:      int(port.PublicPort),
			Protocol:      port.Type,
			HostIP:        port.IP,
		})
	}

	return result
}

func convertMountPaths(mounts *[]container.MountPoint) []string {
	var result []string

	for _, mount := range *mounts {
		var mountPath string

		if mount.Source == mount.Destination {
			mountPath = mount.Source
		} else {
			mountPath = mount.Source + ":" + mount.Destination
		}

		if mount.Mode != "" {
			mountPath += ":" + mount.Mode
		}

		if mountPath != "" {
			result = append(result, mountPath)
		}
	}

	return result
}

func getArchitectureType(arch, variant string) softwaremodel.SoftwareArchitecture {
	switch arch {
	case "386":
		return softwaremodel.SoftwareArchitectureX86
	case "amd64":
		return softwaremodel.SoftwareArchitectureX8664
	case "arm":
		switch variant {
		case "v5":
			return softwaremodel.SoftwareArchitectureARMv5
		case "v6":
			return softwaremodel.SoftwareArchitectureARMv6
		case "v7":
			return softwaremodel.SoftwareArchitectureARMv7
		}
	case "arm64":
		switch variant {
		case "v8":
			return softwaremodel.SoftwareArchitectureARM64v8
		}
	}

	return "Unknown"
}

// runtimeEnvDenyList names variables that systemd or the login session injects at
// start. They describe this host's current boot and session -- a cgroup path, an
// invocation id, an X display, an agent socket -- so replaying them on a
// migration target is at best meaningless and at worst harmful (a stale
// SSH_AUTH_SOCK or DISPLAY forced on every login there).
var runtimeEnvDenyList = map[string]bool{
	// systemd, per-invocation
	"INVOCATION_ID": true, "JOURNAL_STREAM": true, "NOTIFY_SOCKET": true,
	"SYSTEMD_EXEC_PID": true, "MANAGERPID": true, "MANAGERPIDFDID": true,
	"LISTEN_PID": true, "LISTEN_FDS": true, "LISTEN_FDNAMES": true,
	"MEMORY_PRESSURE_WATCH": true, "MEMORY_PRESSURE_WRITE": true,
	// login session / shell
	"USER": true, "USERNAME": true, "LOGNAME": true, "HOME": true, "SHELL": true,
	"PWD": true, "OLDPWD": true, "SHLVL": true, "TERM": true, "MAIL": true,
	"_": true, "PATH": true, "HOSTNAME": true, "LS_COLORS": true, "container": true,
	"SSH_CLIENT": true, "SSH_CONNECTION": true, "SSH_TTY": true, "SSH_AUTH_SOCK": true,
	"GPG_AGENT_INFO": true,
	// desktop session
	"DISPLAY": true, "WAYLAND_DISPLAY": true, "XAUTHORITY": true,
	"DBUS_SESSION_BUS_ADDRESS": true, "DESKTOP_SESSION": true, "DESKTOP_STARTUP_ID": true,
	"GDMSESSION": true, "GNOME_DESKTOP_SESSION_ID": true, "GNOME_SETUP_DISPLAY": true,
	"GTK_MODULES": true, "QT_ACCESSIBILITY": true, "QT_IM_MODULE": true,
	"QT_IM_MODULES": true, "XMODIFIERS": true,
	"GIO_LAUNCHED_DESKTOP_FILE": true, "GIO_LAUNCHED_DESKTOP_FILE_PID": true,
	"XDG_ACTIVATION_TOKEN": true, "XDG_CURRENT_DESKTOP": true, "XDG_MENU_PREFIX": true,
	"XDG_RUNTIME_DIR": true, "XDG_SESSION_CLASS": true, "XDG_SESSION_DESKTOP": true,
	"XDG_SESSION_ID": true, "XDG_SESSION_TYPE": true,
	"XDG_SESSION_EXTRA_DEVICE_ACCESS": true, "XDG_DATA_DIRS": true,
}

// secretEnvKeyTokens flag a variable whose value is likely a credential.
var secretEnvKeyTokens = []string{
	"PASSWORD", "PASSWD", "SECRET", "TOKEN", "APIKEY", "API_KEY",
	"ACCESS_KEY", "PRIVATE_KEY", "CREDENTIAL", "PASSPHRASE",
}

// isSecretEnvKey reports whether key names something that likely holds a credential.
func isSecretEnvKey(key string) bool {
	upper := strings.ToUpper(key)
	for _, token := range secretEnvKeyTokens {
		if strings.Contains(upper, token) {
			return true
		}
	}

	return false
}

// migrationEnvs picks the environment to carry to the migration target.
//
// For a process systemd started, the unit's own declaration (Environment= /
// EnvironmentFile=) is authoritative: it is what the software was configured
// with. /proc/<pid>/environ additionally holds everything systemd and the login
// session injected, which belongs to this host. Anything else falls back to the
// runtime environment with those injected variables removed.
//
// Values of credential-looking variables are dropped: the refined model is
// served over an API with no authentication and is applied to the target's
// system-wide /etc/environment, so neither is a place for a plaintext secret.
// The keys are logged so an operator knows what has to be supplied by hand.
func migrationEnvs(b software.Binary) []string {
	source := b.DeclaredEnviron
	if len(source) == 0 {
		if b.LaunchType == "systemd" {
			// systemd started it and the unit declares nothing, so every variable
			// the process holds was injected. Nothing to carry.
			return nil
		}

		source = b.Environ
	}

	var envs []string
	var omitted []string

	for _, entry := range source {
		key, _, found := strings.Cut(entry, "=")
		if !found || key == "" || runtimeEnvDenyList[key] {
			continue
		}
		if isSecretEnvKey(key) {
			omitted = append(omitted, key)
			continue
		}

		envs = append(envs, entry)
	}

	if len(omitted) > 0 {
		logger.Println(logger.WARN, false, "Refined: omitted credential-looking environment variables from "+
			b.Name+" ("+strings.Join(omitted, ", ")+"); set them on the target by hand")
	}

	return envs
}

func convertEnvs(env *[]string) []softwaremodel.Env {
	var result []softwaremodel.Env

	for _, e := range *env {
		var name string
		var value string

		ee := strings.Split(e, "=")
		if len(ee) >= 1 {
			name = ee[0]
		}
		if len(ee) >= 2 {
			value = ee[1]
		}

		result = append(result, softwaremodel.Env{
			Name:  name,
			Value: value,
		})
	}

	return result
}

func convertToContainers(containers *[]software.Container, runtime softwaremodel.SoftwareContainerRuntimeType) []softwaremodel.Container {
	var result []softwaremodel.Container

	for _, c := range *containers {
		result = append(result, softwaremodel.Container{
			Name:        getContainerName(&c.ContainerSummary),
			Runtime:     runtime,
			ContainerId: c.ContainerSummary.ID,
			ContainerImage: softwaremodel.ContainerImage{
				ImageName:         getImageName(&c.ContainerSummary.Image),
				ImageVersion:      getImageTag(&c.ContainerSummary.Image),
				ImageArchitecture: getArchitectureType(c.ImageInspect.Architecture, c.ImageInspect.Variant),
				ImageHash:         c.ContainerInspect.Image,
			},
			ContainerPorts:    convertPorts(&c.ContainerSummary.Ports),
			ContainerStatus:   c.ContainerInspect.State.Status,
			DockerComposePath: getDockerComposePath(c.ContainerSummary.Labels),
			MountPaths:        convertMountPaths(&c.ContainerInspect.Mounts),
			Envs:              convertEnvs(&c.ContainerInspect.Config.Env),
			NetworkMode:       c.ContainerSummary.HostConfig.NetworkMode,
			RestartPolicy:     string(c.ContainerInspect.HostConfig.RestartPolicy.Name),
		})
	}

	return result
}

// dropSnapShadowedDebs removes deb packages whose name matches an installed
// snap, so a transitional deb stub does not cause a duplicate install alongside
// the snap it points to.
func dropSnapShadowedDebs(debs, snaps []softwaremodel.Package) []softwaremodel.Package {
	if len(snaps) == 0 {
		return debs
	}
	snapNames := make(map[string]bool, len(snaps))
	for _, s := range snaps {
		snapNames[s.Name] = true
	}
	kept := debs[:0]
	for _, d := range debs {
		if snapNames[d.Name] {
			continue
		}
		kept = append(kept, d)
	}
	return kept
}

func doGetRefinedSoftwareInfo(softwareInfo *software.Software) (*softwaremodel.SoftwareList, error) {
	binaries := convertToBinaries(softwareInfo.Legacy)

	var packages []softwaremodel.Package

	debPackages := convertToPackages(softwareInfo.DEB)
	rpmPackages := convertToPackages(softwareInfo.RPM)
	snapPackages := convertToPackages(softwareInfo.Snap)
	flatpakPackages := convertToPackages(softwareInfo.Flatpak)

	// Dedup: on Ubuntu a deb like "firefox" is often a transitional stub whose
	// only job is to pull the snap of the same name. Keeping both double-installs
	// the app, so when a snap and a deb share a name the snap (the real install)
	// wins and the deb is dropped.
	debPackages = dropSnapShadowedDebs(debPackages, snapPackages)

	packages = append(packages, debPackages...)
	packages = append(packages, rpmPackages...)
	packages = append(packages, snapPackages...)
	packages = append(packages, flatpakPackages...)

	var containers []softwaremodel.Container

	dockerContainers := convertToContainers(&softwareInfo.Docker, "docker")
	podmanContainers := convertToContainers(&softwareInfo.Podman, "podman")

	containers = append(containers, dockerContainers...)
	containers = append(containers, podmanContainers...)

	var kubernetes []softwaremodel.Kubernetes

	// TODO: Refine kubernetes resources

	refinedSoftwareInfo := &softwaremodel.SoftwareList{
		Binaries:   binaries,
		Packages:   packages,
		Containers: containers,
		Kubernetes: kubernetes,
	}

	return refinedSoftwareInfo, nil
}

// GetInfraInfoRefined godoc
//
//	@ID				get-infra-info-refined
//	@Summary		Get Refined Infra Information
//	@Description	Get the refined infra information of the connection information.
//	@Tags			[Get] Get refined source info
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the source group."
//	@Param			connId path string true "ID of the connection info."
//	@Success		200	{object}	inframodel.OnpremiseInfraModel	"Successfully get refined information of the infra."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get refined information of the infra."
//	@Router			/source_group/{sgId}/connection_info/{connId}/infra/refined [get]
func GetInfraInfoRefined(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}

	connID := c.Param("connId")
	if connID == "" {
		return common.ReturnErrorMsg(c, "Please provide the connId.")
	}

	_, err := dao.SourceGroupGet(sgID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	infraInfo, err := doGetInfraInfo(connID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	refinedInfraInfo, err := doGetRefinedInfraInfo(infraInfo)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	var onpremiseInfraModel inframodel.OnpremiseInfraModel
	var onpremiseInfra inframodel.OnpremInfra

	onpremiseInfra.Nodes = append(onpremiseInfra.Nodes, *refinedInfraInfo)
	doGetRefinedNetworkInfo(&onpremiseInfra.Network, &infraInfo.Network.Host.Route, &infraInfo.Compute.OS.Node.Machineid)

	// Reflect the collected Kubernetes cluster: its metadata, the node roles,
	// and any cluster nodes (e.g. workers) only visible through the API.
	if k8sInfo := tryGetKubernetesInfo(connID); k8sInfo != nil {
		onpremiseInfra.K8sCluster = buildK8sCluster(k8sInfo)
		onpremiseInfra.Nodes = mergeK8sNodes(onpremiseInfra.Nodes, k8sInfo)
	}

	onpremiseInfraModel.OnpremiseInfraModel = onpremiseInfra

	return c.JSONPretty(http.StatusOK, onpremiseInfraModel, " ")
}

// GetInfraInfoSourceGroupRefined godoc
//
//	@ID				get-infra-info-source-group-refined
//	@Summary		Get Refined Infra Information Source Group
//	@Description	Get the refined infra information for all connections in the source group.
//	@Tags			[Get] Get refined source info
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the source group."
//	@Success		200	{object}	inframodel.OnpremiseInfraModel		"Successfully get refined information of the infra."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get refined information of the infra."
//	@Router		/source_group/{sgId}/infra/refined [get]
func GetInfraInfoSourceGroupRefined(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}

	_, err := dao.SourceGroupGet(sgID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	list, err := dao.ConnectionInfoGetList(&model.ConnectionInfo{SourceGroupID: sgID}, 0, 0)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	var onpremiseInfraModel inframodel.OnpremiseInfraModel
	var onpremiseInfra inframodel.OnpremInfra
	var k8sInfo *kubernetes.Kubernetes

	for _, conn := range *list {
		infraInfo, err := doGetInfraInfo(conn.ID)
		if err != nil {
			return common.ReturnErrorMsg(c, err.Error())
		}
		refinedInfraInfo, err := doGetRefinedInfraInfo(infraInfo)
		if err != nil {
			return common.ReturnErrorMsg(c, err.Error())
		}
		onpremiseInfra.Nodes = append(onpremiseInfra.Nodes, *refinedInfraInfo)
		doGetRefinedNetworkInfo(&onpremiseInfra.Network, &infraInfo.Network.Host.Route, &infraInfo.Compute.OS.Node.Machineid)

		// The Kubernetes cluster is collected once per source group, from the
		// connection of a control plane node whose kubeconfig enumerates the
		// whole cluster.
		if k8sInfo == nil {
			k8sInfo = tryGetKubernetesInfo(conn.ID)
		}
	}

	// Reflect the cluster once all host-level nodes are gathered, so the merge
	// covers nodes appended before the cluster data was found.
	if k8sInfo != nil {
		onpremiseInfra.K8sCluster = buildK8sCluster(k8sInfo)
		onpremiseInfra.Nodes = mergeK8sNodes(onpremiseInfra.Nodes, k8sInfo)
	}

	onpremiseInfraModel.OnpremiseInfraModel = onpremiseInfra

	return c.JSONPretty(http.StatusOK, onpremiseInfraModel, " ")
}

// GetSoftwareInfoRefined godoc
//
//	@ID				get-software-info-refined
//	@Summary		Get Refined Software Information
//	@Description	Get the refined software information of the connection information.
//	@Tags			[Get] Get refined source info
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the source group."
//	@Param			connId path string true "ID of the connection info."
//	@Success		200	{object}	softwaremodel.SourceConnectionInfoSoftwareProperty	"Successfully get refined information of softwares."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get refined information of the infra."
//	@Router			/source_group/{sgId}/connection_info/{connId}/software/refined [get]
func GetSoftwareInfoRefined(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}

	connID := c.Param("connId")
	if connID == "" {
		return common.ReturnErrorMsg(c, "Please provide the connId.")
	}

	_, err := dao.SourceGroupGet(sgID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	softwareInfo, err := doGetSoftwareInfo(connID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	refinedSoftwareInfo, err := doGetRefinedSoftwareInfo(softwareInfo)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	sourceConnectionInfoSoftwareProperty := softwaremodel.SourceConnectionInfoSoftwareProperty{
		ConnectionId: connID,
		Softwares:    *refinedSoftwareInfo,
	}

	var sourceSoftwareModel softwaremodel.SourceSoftwareModel
	sourceSoftwareModel.SourceSoftwareModel.SourceGroupId = sgID
	sourceSoftwareModel.SourceSoftwareModel.ConnectionInfoList =
		append(sourceSoftwareModel.SourceSoftwareModel.ConnectionInfoList, sourceConnectionInfoSoftwareProperty)

	return c.JSONPretty(http.StatusOK, sourceSoftwareModel, " ")
}

// GetSoftwareInfoSourceGroupRefined godoc
//
//	@ID				get-software-info-source-group-refined
//	@Summary		Get Refined Software Information Source Group
//	@Description	Get the refined software information for all connections in the source group.
//	@Tags			[Get] Get refined source info
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the source group."
//	@Success		200	{object}	softwaremodel.SourceGroupSoftwareProperty		"Successfully get refined information of softwares."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get refined information of the software."
//	@Router		/source_group/{sgId}/software/refined [get]
func GetSoftwareInfoSourceGroupRefined(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}

	_, err := dao.SourceGroupGet(sgID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	list, err := dao.ConnectionInfoGetList(&model.ConnectionInfo{SourceGroupID: sgID}, 0, 0)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	var sourceGroupSoftwareProperty softwaremodel.SourceGroupSoftwareProperty

	sourceGroupSoftwareProperty.SourceGroupId = sgID

	for _, conn := range *list {
		softwareInfo, err := doGetSoftwareInfo(conn.ID)
		if err != nil {
			return common.ReturnErrorMsg(c, err.Error())
		}
		refinedSoftwareInfo, err := doGetRefinedSoftwareInfo(softwareInfo)
		if err != nil {
			return common.ReturnErrorMsg(c, err.Error())
		}
		sourceGroupSoftwareProperty.ConnectionInfoList = append(sourceGroupSoftwareProperty.ConnectionInfoList, softwaremodel.SourceConnectionInfoSoftwareProperty{
			ConnectionId: conn.ID,
			Softwares:    *refinedSoftwareInfo,
		})
	}

	var sourceSoftwareModel softwaremodel.SourceSoftwareModel
	sourceSoftwareModel.SourceSoftwareModel = sourceGroupSoftwareProperty

	return c.JSONPretty(http.StatusOK, sourceSoftwareModel, " ")
}
