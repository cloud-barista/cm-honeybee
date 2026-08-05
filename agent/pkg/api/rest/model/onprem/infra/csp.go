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
	Name      string            `json:"name"`       // VM NameId
	ID        string            `json:"id"`         // VM SystemId (CSP native / ARM ID)
	VMSpec    string            `json:"vm_spec"`    // e.g. Standard_D2s_v3
	Image     string            `json:"image"`      // ImageIId NameId
	Platform  string            `json:"platform"`   // e.g. LINUX/UNIX
	PublicIP  string            `json:"public_ip"`
	PrivateIP string            `json:"private_ip"`
	RootDisk  Disk              `json:"root_disk"`
	DataDisks []string          `json:"data_disks"`
	Network   CSPNetwork        `json:"network"`
	Tags      map[string]string `json:"tags"`
	StartTime string            `json:"start_time"`
}

// CSPNetwork groups the CSP-side network resources attached to the VM.
type CSPNetwork struct {
	VPC            CSPVPC             `json:"vpc"`
	Subnet        string             `json:"subnet"`
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
