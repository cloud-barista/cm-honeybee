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

// normalizeRegion case-corrects region against the CSP's metainfo when possible.
// If the region is not in the metainfo list it is returned as-is (some CSPs
// return an incomplete list).
func normalizeRegion(meta *spider.CloudOSMetaInfo, region string) string {
	region = strings.TrimSpace(region)
	if meta == nil {
		return region
	}
	target := strings.ToUpper(region)
	for _, r := range meta.Region {
		if strings.ToUpper(r) == target {
			return r
		}
	}
	return region
}

// canonicalizeCredentialKV normalizes credential KV against the CSP's required
// keys. It returns an error when:
//   - any required key is missing, or
//   - any provided key is not in the required set.
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

	out := make([]model.KeyValue, 0, len(in))
	provided := make(map[string]bool, len(in))
	for _, kv := range in {
		canonical, ok := accept[strings.ToUpper(strings.TrimSpace(kv.Key))]
		if !ok {
			return nil, errors.New("credential key not accepted by " + provider + " CSP: " + kv.Key)
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
// OpenBao. OpenBao is the only secret store — no credential is kept in the DB.
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

// validateAndCanonicalizeCSP validates the supplied plaintext credential and
// region against the CSP metainfo and records canonical provider/region/credential
// on sg. It performs NO writes outside honeybee: credentials go to the drivers
// only in memory at discovery/collection time (see withCSPConn).
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
	region := normalizeRegion(meta, sg.RegionName)
	if region == "" {
		return errors.New("region_name is empty")
	}

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
