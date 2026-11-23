package controller

import (
	"errors"
	"sort"
	"strings"

	serverCommon "github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/cloud-barista/cm-honeybee/server/lib/openbao"
	"github.com/cloud-barista/cm-honeybee/server/lib/spider"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"github.com/jollaman999/utils/logger"
)

// normalizeRegion trims a csp group's region_name and spells it as the CSP's
// region list does, matched case-insensitively, since the region goes to the
// driver as it is stored and some CSPs take only their own spelling (ncp's SDK
// puts the region in the API path and answers 404 for "kr" where it wants
// "KR"). The zone of a "<region>/<zone>" value is matched among that region's
// zones the same way. The list is queried live with credential. A part the
// list does not name, or every part when the query fails, is kept as given
// (trimmed): some CSPs list regions incompletely, and refusing the group for
// it would block a value the driver may still take.
func normalizeRegion(provider string, credential []spider.KeyValue, regionName string) string {
	region, zone, hasZone := splitRegionName(regionName)
	if region == "" {
		return joinRegionName(region, zone, hasZone)
	}

	list, err := listRegionZone(provider, credential)
	if err != nil {
		logger.Println(logger.WARN, true, "CSP: region "+region+" kept as given, failed to list "+provider+
			" regions: "+maskCredential(err, credential).Error())
		return joinRegionName(region, zone, hasZone)
	}
	regionNames := make([]string, 0, len(list))
	for _, r := range list {
		regionNames = append(regionNames, r.Name)
	}
	i := matchName(regionNames, region)
	if i < 0 {
		logger.Println(logger.WARN, true, "CSP: region "+region+" kept as given, not in the "+provider+" region list")
		return joinRegionName(region, zone, hasZone)
	}
	region = list[i].Name
	if zone == "" {
		return joinRegionName(region, zone, hasZone)
	}
	zoneNames := make([]string, 0, len(list[i].ZoneList))
	for _, z := range list[i].ZoneList {
		zoneNames = append(zoneNames, z.Name)
	}
	if j := matchName(zoneNames, zone); j >= 0 {
		zone = zoneNames[j]
	} else {
		logger.Println(logger.WARN, true, "CSP: zone "+zone+" kept as given, not in the zones of "+provider+
			" region "+region)
	}
	return joinRegionName(region, zone, hasZone)
}

// splitRegionName splits region_name into its region and the zone of a
// "<region>/<zone>" value, each trimmed. hasZone tells whether a zone part was
// written at all, so joinRegionName gives back the same form.
func splitRegionName(regionName string) (region, zone string, hasZone bool) {
	region = strings.TrimSpace(regionName)
	if i := strings.Index(region, "/"); i >= 0 {
		return strings.TrimSpace(region[:i]), strings.TrimSpace(region[i+1:]), true
	}
	return region, "", false
}

// joinRegionName is the inverse of splitRegionName.
func joinRegionName(region, zone string, hasZone bool) string {
	if hasZone {
		return region + "/" + zone
	}
	return region
}

// matchName returns the index of want in names, an exact match before one that
// differs only in case, or -1.
func matchName(names []string, want string) int {
	for i, name := range names {
		if name == want {
			return i
		}
	}
	for i, name := range names {
		if strings.EqualFold(name, want) {
			return i
		}
	}
	return -1
}

// normalizeStoredRegion is normalizeRegion for a csp group whose credential is
// not given anew: the region list is queried with the stored credential. When
// that cannot be read the region is kept as given, trimmed as normalizeRegion
// trims it. A value with no region part is returned without reading anything,
// for the caller to refuse.
func normalizeStoredRegion(sg *model.SourceGroup, regionName string) string {
	region, zone, hasZone := splitRegionName(regionName)
	if region == "" {
		return joinRegionName(region, zone, hasZone)
	}
	provider, credential, err := groupCredential(sg)
	if err != nil {
		logger.Println(logger.WARN, true, "CSP: region "+region+
			" kept as given, failed to read the credential of source group "+sg.ID+": "+err.Error())
		return joinRegionName(region, zone, hasZone)
	}
	return normalizeRegion(provider, credential, regionName)
}

// optionalCredentialKeys are the credential keys a csp source group of the
// provider takes besides the required ones (CredentialCSP). They are the S3
// keys buckets are read with where the main keys do not reach object storage
// (spider.UsesS3Keys), so they are needed for object_storage only and must be
// given together.
func optionalCredentialKeys(provider string) []string {
	if spider.UsesS3Keys(provider) {
		return []string{spider.S3AccessKey, spider.S3SecretKey}
	}
	return []string{}
}

// optionalCredentialAliases maps other names of the optional keys, upper-cased,
// to the optional key they are stored as. Azure's S3 keys are a storage account
// name and key, and lib/spider reads them under either name.
func optionalCredentialAliases(provider string) map[string]string {
	if strings.EqualFold(provider, "AZURE") {
		return map[string]string{
			"STORAGEACCOUNTNAME": spider.S3AccessKey,
			"STORAGEACCOUNTKEY":  spider.S3SecretKey,
		}
	}
	return nil
}

// canonicalizeCredentialKV normalizes credential KV against the CSP's required
// and optional keys (optionalCredentialKeys). It returns an error when:
//   - any required key is missing, or
//   - any provided key is neither required nor optional, or
//   - only some of the optional keys are provided.
//
// An optional key with a blank value counts as not provided and is dropped.
func canonicalizeCredentialKV(provider string, meta *spider.CloudOSMetaInfo, in []model.KeyValue) ([]model.KeyValue, error) {
	// Canonical credential keys follow cb-spider's "credentialcsp" convention,
	// which matches cb-tumblebug's template.credentials.yaml (e.g. Azure
	// clientId/clientSecret/…, AWS aws_access_key_id/…). lib/spider maps these to
	// cb-spider's internal keys before a driver sees them, so honeybee stores/advertises
	// the tumblebug-aligned names. The generic keys (ClientId/…) are also accepted
	// as input and normalized to the csp names.
	if meta == nil || len(meta.CredentialCSP) == 0 {
		return in, nil
	}

	accept := make(map[string]string, len(meta.CredentialCSP)*2) // upper(any key) -> canonical csp key
	for i, cspKey := range meta.CredentialCSP {
		accept[strings.ToUpper(cspKey)] = cspKey
		if i < len(meta.Credential) {
			accept[strings.ToUpper(meta.Credential[i])] = cspKey
		}
	}
	optional := optionalCredentialKeys(provider)
	isOptional := make(map[string]bool, len(optional))
	for _, key := range optional {
		accept[strings.ToUpper(key)] = key
		isOptional[key] = true
	}
	for alias, key := range optionalCredentialAliases(provider) {
		accept[alias] = key
	}

	out := make([]model.KeyValue, 0, len(in))
	provided := make(map[string]bool, len(in))
	for _, kv := range in {
		canonical, ok := accept[strings.ToUpper(strings.TrimSpace(kv.Key))]
		if !ok {
			return nil, errors.New("credential key not accepted by " + provider + " CSP: " + kv.Key)
		}
		if isOptional[canonical] && strings.TrimSpace(kv.Value) == "" {
			continue
		}
		if provided[canonical] {
			return nil, errors.New("duplicate credential key: " + canonical)
		}
		provided[canonical] = true
		out = append(out, model.KeyValue{Key: canonical, Value: kv.Value})
	}

	missing := make([]string, 0)
	for _, cspKey := range meta.CredentialCSP {
		if !provided[cspKey] {
			missing = append(missing, cspKey)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return nil, errors.New("missing required credential keys: " + strings.Join(missing, ", "))
	}

	given := 0
	for _, key := range optional {
		if provided[key] {
			given++
		}
	}
	if given > 0 && given < len(optional) {
		return nil, errors.New(strings.Join(optional, " and ") + " must be given together")
	}

	return out, nil
}

// toSpiderKV converts model.KeyValue list into the spider client KV type.
func toSpiderKV(in []model.KeyValue) []spider.KeyValue {
	out := make([]spider.KeyValue, 0, len(in))
	for _, kv := range in {
		out = append(out, spider.KeyValue{Key: kv.Key, Value: kv.Value})
	}
	return out
}

// cspCredentialPath is the OpenBao KV path for a source group's CSP credential.
func cspCredentialPath(sgID string) string { return "honeybee/csp/" + sgID }

func kvToMap(in []model.KeyValue) map[string]string {
	m := make(map[string]string, len(in))
	for _, kv := range in {
		m[kv.Key] = kv.Value
	}
	return m
}

func mapToKV(m map[string]string) []model.KeyValue {
	out := make([]model.KeyValue, 0, len(m))
	for k, v := range m {
		out = append(out, model.KeyValue{Key: k, Value: v})
	}
	return out
}

// storeCSPCredential writes a source group's canonical plaintext credential to
// OpenBao. OpenBao is the only secret store - no credential is kept in the DB.
func storeCSPCredential(sgID string, plain []model.KeyValue) (model.KeyValueList, error) {
	if !openbao.Enabled() {
		return nil, errors.New("OpenBao is required to store CSP credentials (set cm-honeybee.openbao.address)")
	}
	if err := openbao.Put(cspCredentialPath(sgID), kvToMap(plain)); err != nil {
		return nil, err
	}
	return nil, nil
}

// loadCSPCredential returns a source group's plaintext credential from OpenBao.
func loadCSPCredential(sg *model.SourceGroup) ([]model.KeyValue, error) {
	if !openbao.Enabled() {
		return nil, errors.New("OpenBao is required to read CSP credentials (set cm-honeybee.openbao.address)")
	}
	data, err := openbao.Get(cspCredentialPath(sg.ID))
	if err != nil {
		return nil, err
	}
	return mapToKV(data), nil
}

// deleteCSPCredential removes a source group's CSP credential from OpenBao. It is
// a no-op for DB storage (the row delete handles that).
func deleteCSPCredential(sgID string) {
	if !openbao.Enabled() {
		return
	}
	if err := openbao.Delete(cspCredentialPath(sgID)); err != nil {
		logger.Println(logger.WARN, true, "OpenBao: failed to delete CSP credential ("+sgID+"): "+err.Error())
	}
}

// validateAndCanonicalizeCSP validates the supplied plaintext credential against
// the CSP metainfo, matches the region to the CSP's region list read with that
// credential (normalizeRegion), and records canonical provider/region/credential
// on sg. It performs NO writes outside honeybee: credentials go to the drivers
// only in memory, for that read and at discovery/collection time (see withCSPConn).
//
// sg.Credential is left as canonical-cased plaintext; the caller must encrypt it
// before persisting to honeybee's DB.
func validateAndCanonicalizeCSP(sg *model.SourceGroup, plainKV []model.KeyValue) error {
	provider, err := spider.NormalizeProvider(sg.ProviderName)
	if err != nil {
		return err
	}
	meta, err := spider.GetCloudOSMetaInfo(provider)
	if err != nil {
		return errors.New("failed to load CSP metainfo: " + err.Error())
	}
	canonicalKV, err := canonicalizeCredentialKV(provider, meta, plainKV)
	if err != nil {
		return err
	}
	if strings.TrimSpace(sg.RegionName) == "" {
		return errors.New("region_name is empty")
	}
	region := normalizeRegion(provider, toSpiderKV(canonicalKV), sg.RegionName)

	sg.ProviderName = strings.ToLower(provider)
	sg.RegionName = region
	sg.Credential = canonicalKV
	return nil
}

// splitRegionZone splits a source group's region_name into region and zone.
// The zone may be supplied by writing region_name as "<region>/<zone>"
// (e.g. "koreacentral/1"). Zone precedence: explicit override
// (connection_info.zone) > "<region>/<zone>" embedded in region_name > none.
func splitRegionZone(regionName, zoneOverride string) (string, string, error) {
	region := strings.TrimSpace(regionName)
	zone := strings.TrimSpace(zoneOverride)
	if i := strings.Index(region, "/"); i >= 0 {
		if zone == "" {
			zone = strings.TrimSpace(region[i+1:])
		}
		region = strings.TrimSpace(region[:i])
	}
	if region == "" {
		return "", "", errors.New("source group has no region")
	}
	return region, zone, nil
}

// cspCredential returns the canonical provider name and the stored plaintext
// credential of a CSP SourceGroup.
func cspCredential(sg *model.SourceGroup) (string, []spider.KeyValue, error) {
	if sg == nil || sg.Type != serverCommon.SourceGroupTypeCSP {
		return "", nil, errors.New("source group is not a csp-type group")
	}
	provider, err := spider.NormalizeProvider(sg.ProviderName)
	if err != nil {
		return "", nil, err
	}
	plainKV, err := loadCSPCredential(sg)
	if err != nil {
		return "", nil, err
	}
	return provider, toSpiderKV(plainKV), nil
}

// groupCredential reads a csp group's stored credential where the region of
// the group is matched to the CSP's spelling. It is a variable so tests can
// stand in for OpenBao.
var groupCredential = cspCredential

// withCSPConn builds the CSP connection of a SourceGroup from its region and
// its credential in OpenBao, and invokes fn with it. Nothing is registered or
// written anywhere: the credential stays in memory for the duration of fn.
func withCSPConn(sg *model.SourceGroup, zoneOverride string, fn func(conn spider.Conn) error) error {
	provider, credential, err := cspCredential(sg)
	if err != nil {
		return err
	}
	region, zone, err := splitRegionZone(sg.RegionName, zoneOverride)
	if err != nil {
		return err
	}
	return fn(spider.Conn{
		Provider:   provider,
		Region:     region,
		Zone:       zone,
		Credential: credential,
	})
}
