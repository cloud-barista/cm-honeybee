package network

type NetworkSubsystem struct {
	NetworkInterfaces []NIC     `json:"network_interfaces"`
	Netfilter         Netfilter `json:"netfilter"`
	Bonding           []Bonding `json:"bonding"`
}

type Network struct {
	NetworkSubsystem NetworkSubsystem `json:"network_subsystem"`
}
