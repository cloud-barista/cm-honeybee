package network

type Route struct {
	Destination string `json:"destination"`
	Netmask     string `json:"netmask"`
	Source      string `json:"source"`
	NextHop     string `json:"next_hop"`
	Metric      int    `json:"metric"`
	Scope       string `json:"scope"`
	Proto       string `json:"proto"`
	Link        string `json:"link"`
}
