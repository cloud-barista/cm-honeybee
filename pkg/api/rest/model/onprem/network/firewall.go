package network

// FirewallRule TODO
type FirewallRule struct {
	Priority  uint   `json:"priority"` // Lower has higher priority
	Src       string `json:"src"`
	Dst       string `json:"dst"`
	SrcPort   uint   `json:"src_port"`
	DstPort   uint   `json:"dst_port"`
	Protocol  string `json:"protocol"`  // TCP, UDP, ICMP
	Direction string `json:"direction"` // inbound, outbound
	Action    string `json:"action"`    // allow, deny
}
