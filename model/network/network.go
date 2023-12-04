package network

type Network struct {
	NetworkInterfaces []NIC     `json:"network_interfaces"`
	Bonding           []Bonding `json:"bonding"`
}
