package spider

type allNLBInfoResp struct {
	AllListInfo struct {
		MappedInfoList  []NLBInfo `json:"MappedInfoList"`
		OnlyCSPInfoList []NLBInfo `json:"OnlyCSPInfoList"`
	} `json:"AllListInfo"`
}

// ListAllNLBInfo returns every NLB the CSP actually has — both the ones
// cb-spider manages and the ones it does not.
//
// GET /nlb walks cb-spider's own meta-DB, so an NLB cb-spider did not create —
// i.e. every NLB in a migration source — never appears there. This is the same
// trap that made VM collection use GetCSPVM instead of GetVM (see
// cspVMIdentifier's comment in cspAdapter.go).
//
// The values are raw driver output: unlike GET /nlb this path never runs
// cb-spider's transformArgsToUpper(), so Type/Scope/Protocol keep whatever case
// the driver produced and VpcIID keeps the driver's value instead of the
// meta-DB's. Normalising that is the caller's job — see nlbInfoToNLB().
func ListAllNLBInfo(connectionName string) ([]NLBInfo, error) {
	if err := mustNonEmpty("ConnectionName", connectionName); err != nil {
		return nil, err
	}
	var r allNLBInfoResp
	if err := do("GET", "/allnlbinfo?ConnectionName="+encodePath(connectionName), nil, &r); err != nil {
		return nil, err
	}
	return append(append([]NLBInfo{}, r.AllListInfo.MappedInfoList...),
		r.AllListInfo.OnlyCSPInfoList...), nil
}
