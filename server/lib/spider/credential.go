package spider

type credentialCreateReq struct {
	CredentialName   string     `json:"CredentialName"`
	ProviderName     string     `json:"ProviderName"`
	KeyValueInfoList []KeyValue `json:"KeyValueInfoList"`
}

// RegisterCredential creates a credential entry in cb-spider.
func RegisterCredential(name, providerName string, kv []KeyValue) (*CredentialInfo, error) {
	if err := mustNonEmpty("CredentialName", name); err != nil {
		return nil, err
	}
	if err := mustNonEmpty("ProviderName", providerName); err != nil {
		return nil, err
	}
	body := credentialCreateReq{
		CredentialName:   name,
		ProviderName:     providerName,
		KeyValueInfoList: kv,
	}
	var out CredentialInfo
	if err := do("POST", "/credential", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UnregisterCredential deletes a credential entry. 404 is treated as success.
func UnregisterCredential(name string) error {
	if err := mustNonEmpty("CredentialName", name); err != nil {
		return err
	}
	err := do("DELETE", "/credential/"+encodePath(name), nil, nil)
	if notFoundError(err) {
		return nil
	}
	return err
}

// GetCredential fetches a credential entry by name.
func GetCredential(name string) (*CredentialInfo, error) {
	if err := mustNonEmpty("CredentialName", name); err != nil {
		return nil, err
	}
	var out CredentialInfo
	if err := do("GET", "/credential/"+encodePath(name), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
