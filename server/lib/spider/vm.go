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
