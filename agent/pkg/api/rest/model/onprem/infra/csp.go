package infra

// CSPInfo holds the VM information that can only be obtained from the cloud
// provider (via cb-spider), as opposed to the in-guest information collected by
// the agent. It is populated for CSP-type source connections and is kept
// separate from the agent-collected compute/network data so neither overwrites
// the other.
type CSPInfo struct {
	Provider  string            `json:"provider"`
	Region    string            `json:"region"`
	Zone      string            `json:"zone"`
	Name      string            `json:"name"`     // VM NameId
	ID        string            `json:"id"`       // VM SystemId (CSP native / ARM ID)
	VMSpec    string            `json:"vm_spec"`  // e.g. Standard_D2s_v3
	Image     string            `json:"image"`    // ImageIId NameId
	Platform  string            `json:"platform"` // e.g. LINUX/UNIX
	PublicIP  string            `json:"public_ip"`
	PrivateIP string            `json:"private_ip"`
	RootDisk  Disk              `json:"root_disk"`
	DataDisks []string          `json:"data_disks"`
	Network   CSPNetwork        `json:"network"`
	Tags      map[string]string `json:"tags"`
	StartTime string            `json:"start_time"`

	// The fields below come from separate driver calls made after GetVM. Each
	// is left empty when its call fails or the driver has no such handler.
	KeyPair         *CSPKeyPair   `json:"keypair,omitempty"`
	Status          string        `json:"status,omitempty"` // e.g. Running, Suspended
	DataDiskDetails []CSPDataDisk `json:"data_disk_details,omitempty"`
	VMSpecDetail    *CSPVMSpec    `json:"vm_spec_detail,omitempty"`
	ImageDetail     *CSPImage     `json:"image_detail,omitempty"`
}

// CSPKeyPair is the key pair the VM was created with. Name is the VM's
// KeyPairIId NameId, or its SystemId when the driver left NameId empty.
// Fingerprint is empty when the key pair could not be looked up.
type CSPKeyPair struct {
	Name        string `json:"name"`
	ID          string `json:"id,omitempty"`
	Fingerprint string `json:"fingerprint,omitempty"`
}

// CSPDataDisk is one data disk attached to the VM.
type CSPDataDisk struct {
	Name   string `json:"name"`
	ID     string `json:"id"`
	Size   uint   `json:"size"` // GB
	Type   string `json:"type"`
	Status string `json:"status"`
}

// CSPVMSpec is the size of the VM's spec. A number the CSP does not expose
// is 0.
type CSPVMSpec struct {
	Name         string   `json:"name"`
	VCPUs        uint     `json:"vcpus"`
	MemoryMiB    uint     `json:"memory_mib"`
	DiskGB       uint     `json:"disk_gb"` // 0 when the spec has no fixed disk
	Architecture string   `json:"architecture,omitempty"`
	GPUs         []CSPGPU `json:"gpus,omitempty"`
}

type CSPGPU struct {
	Count    uint   `json:"count"`
	Vendor   string `json:"vendor,omitempty"`
	Model    string `json:"model,omitempty"`
	MemoryGB uint   `json:"memory_gb,omitempty"` // per device
}

// CSPImage is the public image the VM was created from, as the driver
// describes it.
type CSPImage struct {
	Name           string `json:"name"`
	ID             string `json:"id,omitempty"`
	OSDistribution string `json:"os_distribution,omitempty"` // e.g. Ubuntu 22.04
	OSArchitecture string `json:"os_architecture,omitempty"` // e.g. x86_64
	OSPlatform     string `json:"os_platform,omitempty"`     // e.g. Linux/UNIX
	OSDiskType     string `json:"os_disk_type,omitempty"`
	OSDiskSizeGB   uint   `json:"os_disk_size_gb,omitempty"`
	Status         string `json:"status,omitempty"`
}

// CSPNetwork groups the CSP-side network resources attached to the VM.
type CSPNetwork struct {
	VPC            CSPVPC             `json:"vpc"`
	Subnet         string             `json:"subnet"`
	SecurityGroups []CSPSecurityGroup `json:"security_groups"`
}

type CSPVPC struct {
	Name    string      `json:"name"`
	CIDR    string      `json:"cidr"`
	Subnets []CSPSubnet `json:"subnets"`
}

type CSPSubnet struct {
	Name string `json:"name"`
	CIDR string `json:"cidr"`
	Zone string `json:"zone"`
}

type CSPSecurityGroup struct {
	Name  string            `json:"name"`
	Rules []CSPSecurityRule `json:"rules"`
}

type CSPSecurityRule struct {
	Direction string `json:"direction"`
	Protocol  string `json:"protocol"`
	FromPort  string `json:"from_port"`
	ToPort    string `json:"to_port"`
	CIDR      string `json:"cidr"`
}
