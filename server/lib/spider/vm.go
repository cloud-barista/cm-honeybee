package spider

type vmListResp struct {
	VM []VMInfo `json:"vm"`
}

type allVMInfoResp struct {
	AllListInfo struct {
		MappedInfoList  []VMInfo `json:"MappedInfoList"`
		OnlyCSPInfoList []VMInfo `json:"OnlyCSPInfoList"`
	} `json:"AllListInfo"`
}

// ListVM returns the VMs cb-spider itself manages on the given connection.
//
// This walks cb-spider's own meta-DB, so a VM cb-spider did not create - i.e.
// every VM in a migration source - never appears here. Use ListAllVMInfo for
// discovery; this is the list counterpart of GetVM, not of GetCSPVM.
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

// ListAllVMInfo returns every VM the CSP actually has - both the ones cb-spider
// manages (MappedInfoList) and the ones it does not (OnlyCSPInfoList).
//
// Source discovery needs this rather than ListVM for the same reason collection
// needs GetCSPVM rather than GetVM: a pre-existing VM is invisible to the paths
// that resolve against cb-spider's managed-resource store.
func ListAllVMInfo(connectionName string) ([]VMInfo, error) {
	if err := mustNonEmpty("ConnectionName", connectionName); err != nil {
		return nil, err
	}
	var r allVMInfoResp
	if err := do("GET", "/allvminfo?ConnectionName="+encodePath(connectionName), nil, &r); err != nil {
		return nil, err
	}
	return append(append([]VMInfo{}, r.AllListInfo.MappedInfoList...),
		r.AllListInfo.OnlyCSPInfoList...), nil
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
// resource ID). Unlike GetVM - which resolves names against cb-spider's own
// managed-VM store and fails for VMs cb-spider did not create - GetCSPVM queries
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
