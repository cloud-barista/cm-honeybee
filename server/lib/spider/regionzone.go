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

type regionZoneListResp struct {
	RegionZone []RegionZoneInfo `json:"regionzone"`
}

// ListRegionZonePreConfig lists the CSP's available regions and zones by querying
// the cloud live via a registered driver + credential (no cb-spider region /
// connection registration needed). This is how honeybee lists the real regions
// for a CSP source group once its credential is stored.
func ListRegionZonePreConfig(driverName, credentialName string) ([]RegionZoneInfo, error) {
	if err := mustNonEmpty("DriverName", driverName); err != nil {
		return nil, err
	}
	if err := mustNonEmpty("CredentialName", credentialName); err != nil {
		return nil, err
	}
	var r regionZoneListResp
	if err := do("GET", "/preconfig/regionzone?DriverName="+encodePath(driverName)+
		"&CredentialName="+encodePath(credentialName), nil, &r); err != nil {
		return nil, err
	}
	return r.RegionZone, nil
}
