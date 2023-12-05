package network

type Network struct {
	NetworkInterfaces []NIC     `json:"network_interfaces"`
	Netfilter         Netfilter `json:"netfilter"`
	Bonding           []Bonding `json:"bonding"`
}
