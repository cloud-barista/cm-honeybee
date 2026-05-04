package spider

type regionCreateReq struct {
	RegionName       string     `json:"RegionName"`
	ProviderName     string     `json:"ProviderName"`
	KeyValueInfoList []KeyValue `json:"KeyValueInfoList"`
}

// RegisterRegion registers a region entry. cb-spider requires a region row even
// when ConnectionConfig only references a region by name.
func RegisterRegion(name, providerName string, kv []KeyValue) (*RegionInfo, error) {
	if err := mustNonEmpty("RegionName", name); err != nil {
		return nil, err
	}
	if err := mustNonEmpty("ProviderName", providerName); err != nil {
		return nil, err
	}
	body := regionCreateReq{
		RegionName:       name,
		ProviderName:     providerName,
		KeyValueInfoList: kv,
	}
	var out RegionInfo
	if err := do("POST", "/region", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UnregisterRegion deletes a region entry. 404 is treated as success.
func UnregisterRegion(name string) error {
	if err := mustNonEmpty("RegionName", name); err != nil {
		return err
	}
	err := do("DELETE", "/region/"+encodePath(name), nil, nil)
	if notFoundError(err) {
		return nil
	}
	return err
}

// GetRegion fetches a region entry by name.
func GetRegion(name string) (*RegionInfo, error) {
	if err := mustNonEmpty("RegionName", name); err != nil {
		return nil, err
	}
	var out RegionInfo
	if err := do("GET", "/region/"+encodePath(name), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
