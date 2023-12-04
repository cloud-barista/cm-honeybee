package network

type SlavesList struct {
	Interfaces           []string `json:"interfaces"`
	PrimarySlave         string   `json:"primary_slave"`
	CurrentlyActiveSlave string   `json:"currently_active_slave"`
}

type SlaveInterface struct {
	Name             string `json:"name"`
	MIIStatus        string `json:"mii_status"`
	Speed            string `json:"speed"`
	Duplex           string `json:"duplex"`
	LinkFailureCount uint   `json:"link_failure_count"`
	PermanentHWAddr  string `json:"permanent_hw_addr"`
	AggregatorID     string `json:"aggregator_id"`
}

type Bonding struct {
	Name               string           `json:"name"`
	BondingMode        string           `json:"bonding_mode"`
	SlavesList         SlavesList       `json:"slaves_list"`
	TransmitHashPolicy string           `json:"transmit_hash_policy"`
	AddrList           []string         `json:"addr_list"`
	MIIStatus          string           `json:"mii_status"`
	MIIPollingInterval string           `json:"mii_polling_interval"`
	SlaveInterface     []SlaveInterface `json:"slave_interface"`
}
