package network

type FirewallRule struct {
	Priority  uint   `json:"priority"`  // Lower has higher priority
	Src       string `json:"src"`       // e.g., "123.123.123.123/32", "123.123.123.123/24", "0.0.0.0/0", "2001:db8:4567::/48", "2001:db8:1234:0::/64", "::/0"
	Dst       string `json:"dst"`       // e.g., "123.123.123.123/32", "123.123.123.123/24", "0.0.0.0/0", "2001:db8:4567::/48", "2001:db8:1234:0::/64", "::/0"
	SrcPorts  string `json:"src_ports"` // e.g., "80", "80,443", "1024-65535"
	DstPorts  string `json:"dst_ports"` // e.g., "80", "80,443", "1024-65535"
	Protocol  string `json:"protocol"`  // *, tcp, udp, icmp, icmpv6
	Direction string `json:"direction"` // inbound, outbound
	Action    string `json:"action"`    // allow, deny
}
