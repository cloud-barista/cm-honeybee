package spider

// RegisterConnectionConfig binds a Credential + Region + Driver into a single
// named connection that all resource APIs use as the ConnectionName.
func RegisterConnectionConfig(cfg ConnectionConfigInfo) (*ConnectionConfigInfo, error) {
	if err := mustNonEmpty("ConfigName", cfg.ConfigName); err != nil {
		return nil, err
	}
	if err := mustNonEmpty("ProviderName", cfg.ProviderName); err != nil {
		return nil, err
	}
	if err := mustNonEmpty("DriverName", cfg.DriverName); err != nil {
		return nil, err
	}
	if err := mustNonEmpty("CredentialName", cfg.CredentialName); err != nil {
		return nil, err
	}
	if err := mustNonEmpty("RegionName", cfg.RegionName); err != nil {
		return nil, err
	}

	var out ConnectionConfigInfo
	if err := do("POST", "/connectionconfig", cfg, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UnregisterConnectionConfig deletes the connection config. 404 is treated as success.
func UnregisterConnectionConfig(name string) error {
	if err := mustNonEmpty("ConfigName", name); err != nil {
		return err
	}
	err := do("DELETE", "/connectionconfig/"+encodePath(name), nil, nil)
	if notFoundError(err) {
		return nil
	}
	return err
}

// GetConnectionConfig fetches a connection config by name.
func GetConnectionConfig(name string) (*ConnectionConfigInfo, error) {
	if err := mustNonEmpty("ConfigName", name); err != nil {
		return nil, err
	}
	var out ConnectionConfigInfo
	if err := do("GET", "/connectionconfig/"+encodePath(name), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
