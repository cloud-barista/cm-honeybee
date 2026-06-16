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
			TotalSize: uint64(infraInfo.Compute.ComputeResource.Memory.Size / 1024),      // GiB
			Available: uint64(infraInfo.Compute.ComputeResource.Memory.Available / 1024), // GiB
			Used:      uint64(infraInfo.Compute.ComputeResource.Memory.Used / 1024),      // GiB
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
	}

	return &refinedInfraInfo, nil
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

	return inframodel.NodeProperty{
		Hostname:  hostname,
		MachineId: k8sNodeInfoString(node.NodeInfo, "machineID"),
		Role:      string(node.Type),
		CPU: inframodel.CpuProperty{
			Architecture: k8sNodeInfoString(node.NodeInfo, "architecture"),
			Threads:      uint32(node.NodeSpec.CPU),
		},
		Memory: inframodel.MemoryProperty{
			TotalSize: uint64(node.NodeSpec.Memory / 1024), // MiB -> GiB
		},
		RootDisk: inframodel.DiskProperty{
			TotalSize: uint64(node.NodeSpec.EphemeralStorage / 1024), // MiB -> GiB
		},
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
		if nodes[i].MachineId != "" {
			index[nodes[i].MachineId] = i
		}
	}

	for _, node := range k8sInfo.Nodes {
		machineID := k8sNodeInfoString(node.NodeInfo, "machineID")
		if machineID != "" {
			if i, ok := index[machineID]; ok {
				nodes[i].Role = string(node.Type)
				if nodes[i].CPU.Architecture == "" {
					nodes[i].CPU.Architecture = k8sNodeInfoString(node.NodeInfo, "architecture")
				}
				continue
			}
		}

		nodes = append(nodes, buildNodeFromK8s(node))
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
// For JVM applications (e.g. Tomcat) the install root and runtime dependency are
// derived rather than left as the raw executable: BinaryPath becomes the Tomcat
// install dir (catalina.home), the JDK home is added as a needed dependency, and a
// distinct catalina.base instance dir (if any) is carried as a data directory.
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
		// Dependencies are already filtered to non-package-owned paths by the agent
		// (package-provided runtimes are handled by package migration).
		neededLibraries := append([]string{}, b.Dependencies...)
		dataDirs := append([]string{}, b.DataDirs...)

		// JVM application: derive the real install root (catalina.home).
		catalinaHome, catalinaBase := extractCatalinaPaths(b.CmdlineSlice)
		if catalinaHome != "" {
			binaryPath = catalinaHome
		} else if catalinaBase != "" {
			binaryPath = catalinaBase
		}
		if catalinaBase != "" && catalinaBase != catalinaHome {
			// Separate instance directory (multi-instance Tomcat) holds conf/webapps/logs.
			dataDirs = appendUniquePath(dataDirs, catalinaBase)
		}

		// Wine application: the WINEPREFIX bottle holds the app, registry and config,
		// so it becomes the path to copy.
		if b.IsWine && b.WinePrefix != "" {
			binaryPath = b.WinePrefix
		}

		result = append(result, softwaremodel.Binary{
			Name:             b.Name,
			Version:          b.Version,
			UIDs:             b.UIDs,
			GIDs:             b.GIDs,
			CmdlineSlice:     b.CmdlineSlice,
			Envs:             b.Environ,
			NeededLibraries:  neededLibraries,
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

func doGetRefinedSoftwareInfo(softwareInfo *software.Software) (*softwaremodel.SoftwareList, error) {
	binaries := convertToBinaries(softwareInfo.Legacy)

	var packages []softwaremodel.Package

	debPackages := convertToPackages(softwareInfo.DEB)
	rpmPackages := convertToPackages(softwareInfo.RPM)

	packages = append(packages, debPackages...)
	packages = append(packages, rpmPackages...)

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
