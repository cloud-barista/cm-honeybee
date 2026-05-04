package spider

type cloudOSListResp struct {
	Cloudos []string `json:"cloudos"`
}

// ListCloudOS returns the list of supported CSP names (canonical casing).
func ListCloudOS() ([]string, error) {
	var out cloudOSListResp
	if err := do("GET", "/cloudos", nil, &out); err != nil {
		return nil, err
	}
	return out.Cloudos, nil
}

// GetCloudOSMetaInfo returns metadata for the given Cloud OS, including the
// required credential keys and supported regions.
func GetCloudOSMetaInfo(cloudOSName string) (*CloudOSMetaInfo, error) {
	if err := mustNonEmpty("CloudOSName", cloudOSName); err != nil {
		return nil, err
	}
	var out CloudOSMetaInfo
	if err := do("GET", "/cloudos/metainfo/"+encodePath(cloudOSName), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
