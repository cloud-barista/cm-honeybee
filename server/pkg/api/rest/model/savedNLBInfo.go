package model

import "time"

// SavedNLBInfo stores what was collected for a CSP network load balancer
// connection (resource_type nlb). NLBData holds a CSPNLBRecord.
type SavedNLBInfo struct {
	ConnectionID string    `gorm:"primaryKey" json:"connection_id" validate:"required"`
	NLBData      string    `gorm:"column:nlb_data" json:"nlb_data" validate:"required"`
	Status       string    `gorm:"column:status" json:"status"`
	SavedTime    time.Time `gorm:"column:saved_time" json:"saved_time"`
	// ErrorMessage says why the last import of the connection failed. It is
	// filled only in a group import response and is never stored.
	ErrorMessage string `gorm:"-" json:"error_message,omitempty"`
}

// CSPNLBRecord is a CSP network load balancer as the CSP driver reported it,
// with the values drivers spell differently brought to one form: Type is
// PUBLIC or INTERNAL, or empty when the driver did not tell, and the scope and
// protocols are upper case. Ports are kept as the driver reported them.
type CSPNLBRecord struct {
	SystemID      string              `json:"system_id"`
	Name          string              `json:"name"`
	VPC           string              `json:"vpc,omitempty"`
	Type          string              `json:"type"`
	Scope         string              `json:"scope,omitempty"`
	Listener      CSPNLBListener      `json:"listener"`
	VMGroup       CSPNLBVMGroup       `json:"vm_group"`
	HealthChecker CSPNLBHealthChecker `json:"health_checker"`
	CreatedTime   string              `json:"created_time,omitempty"`
	// VMGroupUnknown is true when the driver reported no target group at all
	// (no protocol, port or VMs), so VMGroup and HealthChecker are unknown
	// rather than empty. The aws driver looks the target group up by the NLB's
	// name and reports a target group of another name as none. A target group
	// with no targets still has its protocol and port, so it stays false.
	VMGroupUnknown bool `json:"vm_group_unknown"`
}

// CSPNLBListener is the frontend of an NLB. IP is empty on drivers that do not
// report it (an AWS NLB without elastic IPs) and may hold several addresses
// joined by commas (AWS, one per zone).
type CSPNLBListener struct {
	Protocol string `json:"protocol"`
	IP       string `json:"ip,omitempty"`
	Port     string `json:"port"`
	DNSName  string `json:"dns_name,omitempty"`
}

// CSPNLBVMGroup is the backend of an NLB. The drivers report its VMs by their
// identifiers only, so their addresses are not part of the record.
type CSPNLBVMGroup struct {
	Protocol string     `json:"protocol"`
	Port     string     `json:"port"`
	VMs      []CSPNLBVM `json:"vms"`
}

// CSPNLBVM is one backend VM of an NLB. Name is empty when the CSP gives the
// VM no name apart from its ID.
type CSPNLBVM struct {
	SystemID string `json:"system_id"`
	Name     string `json:"name,omitempty"`
}

// CSPNLBHealthChecker is the health check of an NLB's VM group. Interval and
// Timeout are in seconds; a value the driver reports as not supported (Azure's
// timeout of -1) is 0.
type CSPNLBHealthChecker struct {
	Protocol  string `json:"protocol,omitempty"`
	Port      string `json:"port,omitempty"`
	Interval  int    `json:"interval"`
	Timeout   int    `json:"timeout"`
	Threshold int    `json:"threshold"`
}

// NLBInfoList is the collected NLBs of every connection in a source group whose
// last import succeeded.
type NLBInfoList struct {
	Servers []CSPNLBRecord `json:"servers" validate:"required"`
}
