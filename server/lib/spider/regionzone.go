package spider

// ZoneInfo mirrors cb-spider's ZoneInfo (subset used by honeybee).
type ZoneInfo struct {
	Name           string `json:"Name"`
	DisplayName    string `json:"DisplayName"`
	CSPDisplayName string `json:"CSPDisplayName"`
	Status         string `json:"Status"`
}

// RegionZoneInfo mirrors cb-spider's RegionZoneInfo (subset used by honeybee).
type RegionZoneInfo struct {
	Name           string     `json:"Name"`
	DisplayName    string     `json:"DisplayName"`
	CSPDisplayName string     `json:"CSPDisplayName"`
	ZoneList       []ZoneInfo `json:"ZoneList"`
}
