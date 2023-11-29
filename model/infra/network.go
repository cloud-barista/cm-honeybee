package infra

type Route struct {
	Destination string `json:"destination"`
	Netmask     string `json:"netmask"`
	NextHop     string `json:"next_hop"`
}

type PhysicalNetwork struct {
	Interface string   `json:"interface"`
	Address   []string `json:"address"`
	Gateway   []string `json:"gateway"`
	Route     []Route  `json:"route"`
	MAC       string   `json:"mac"`
	MTU       int      `json:"mtu"`
}

type Network struct {
	PhysicalNetwork []PhysicalNetwork `json:"physical_network"`
}
