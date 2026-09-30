package spider

// DriverCapability is the subset of cb-spider's DriverCapabilityInfo that
// honeybee consults before asking a driver for a resource it may not implement.
type DriverCapability struct {
	NLBHandler     bool `json:"NLBHandler"`
	ClusterHandler bool `json:"ClusterHandler"`
	VMHandler      bool `json:"VMHandler"`
}
