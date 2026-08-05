package spider

type vmListResp struct {
	VM []VMInfo `json:"vm"`
}

// ListVM returns all VMs reachable through the given connection.
func ListVM(connectionName string) ([]VMInfo, error) {
	if err := mustNonEmpty("ConnectionName", connectionName); err != nil {
		return nil, err
	}
	var out vmListResp
	if err := do("GET", "/vm?ConnectionName="+encodePath(connectionName), nil, &out); err != nil {
		return nil, err
	}
	return out.VM, nil
}

// GetVM fetches a single VM by name (NameId or SystemId, depending on driver).
func GetVM(connectionName, vmName string) (*VMInfo, error) {
	if err := mustNonEmpty("ConnectionName", connectionName); err != nil {
		return nil, err
	}
	if err := mustNonEmpty("Name", vmName); err != nil {
		return nil, err
	}
	var out VMInfo
	if err := do("GET", "/vm/"+encodePath(vmName)+"?ConnectionName="+encodePath(connectionName), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetCSPVM fetches an existing VM by its CSP native ID (e.g. an Azure ARM
// resource ID). Unlike GetVM — which resolves names against cb-spider's own
// managed-VM store and fails for VMs cb-spider did not create — GetCSPVM queries
// the CSP live, which is what source discovery of a pre-existing VM requires.
func GetCSPVM(connectionName, cspID string) (*VMInfo, error) {
	if err := mustNonEmpty("ConnectionName", connectionName); err != nil {
		return nil, err
	}
	if err := mustNonEmpty("Id", cspID); err != nil {
		return nil, err
	}
	var out VMInfo
	if err := do("GET", "/cspvm/"+encodePath(cspID)+"?ConnectionName="+encodePath(connectionName), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
