package controller

import (
	"errors"
	"sort"
	"strings"

	serverCommon "github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/cloud-barista/cm-honeybee/server/lib/openbao"
	"github.com/cloud-barista/cm-honeybee/server/lib/spider"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"github.com/google/uuid"
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
	// clientId/clientSecret/…, AWS aws_access_key_id/…). cb-spider auto-maps these
	// to its internal keys on RegisterCredential, so honeybee stores/advertises
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
// on sg. It performs NO writes to cb-spider — credentials are registered only
// transiently at discovery/collection time (see withSpiderConnection).
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

// buildRegionKV builds the cb-spider RegionInfo key/values for a source group's
// region. cb-spider requires exactly the keys listed in the provider's CloudOS
// metainfo — Azure/AWS/GCP require both "Region" and "Zone", so sending only
// "Region" makes cb-spider reject the registration ("want [Region Zone]").
//
// The zone may be supplied by writing region_name as "<region>/<zone>"
// (e.g. "koreacentral/1"). When a Zone key is required but no zone is given, it
// defaults to "1": the value is not used for resource lookups by ID (Get by
// resource_id), but the key must be present for cb-spider to accept the region.
func buildRegionKV(provider, regionName, zoneOverride string) ([]spider.KeyValue, error) {
	region := strings.TrimSpace(regionName)
	// Zone precedence: explicit override (connection_info.zone) > "<region>/<zone>"
	// embedded in region_name > provider default ("1") applied below.
	zone := strings.TrimSpace(zoneOverride)
	if i := strings.Index(region, "/"); i >= 0 {
		if zone == "" {
			zone = strings.TrimSpace(region[i+1:])
		}
		region = strings.TrimSpace(region[:i])
	}
	if region == "" {
		return nil, errors.New("source group has no region")
	}

	meta, err := spider.GetCloudOSMetaInfo(provider)
	if err != nil || meta == nil || len(meta.Region) == 0 {
		// Metainfo unavailable — fall back to a bare Region key.
		return []spider.KeyValue{{Key: "Region", Value: region}}, nil
	}

	kv := make([]spider.KeyValue, 0, len(meta.Region))
	for _, key := range meta.Region {
		if strings.EqualFold(key, "Zone") {
			z := zone
			if z == "" {
				z = "1"
			}
			kv = append(kv, spider.KeyValue{Key: key, Value: z})
			continue
		}
		kv = append(kv, spider.KeyValue{Key: key, Value: region})
	}

	return kv, nil
}

// withSpiderCredential registers a TEMPORARY cb-spider credential (+ ensures the
// driver) for the given CSP SourceGroup, invokes fn with the driver and
// credential names, and unregisters the credential before returning. Unlike
// withSpiderConnection it registers NO region/connection — used for lookups that
// only need a credential + driver (e.g. listing the CSP's regions).
func withSpiderCredential(sg *model.SourceGroup, fn func(driverName, credName string) error) error {
	if sg == nil || sg.Type != serverCommon.SourceGroupTypeCSP {
		return errors.New("source group is not a csp-type group")
	}
	provider, err := spider.NormalizeProvider(sg.ProviderName)
	if err != nil {
		return err
	}
	plainKV, err := loadCSPCredential(sg)
	if err != nil {
		return err
	}
	driverName, err := spider.EnsureDriver(provider)
	if err != nil {
		return errors.New("failed to ensure driver: " + err.Error())
	}

	credName := "honeybee-tmp-cred-" + uuid.New().String()
	if _, err := spider.RegisterCredential(credName, provider, toSpiderKV(plainKV)); err != nil {
		return errors.New("failed to register temporary credential on cb-spider: " + err.Error())
	}
	defer func() {
		if err := spider.UnregisterCredential(credName); err != nil {
			logger.Println(logger.WARN, true, "failed to unregister temporary spider credential: "+err.Error())
		}
	}()

	return fn(driverName, credName)
}

// withSpiderConnection registers a TEMPORARY cb-spider credential + region +
// connection for the given CSP SourceGroup, invokes fn with the resulting
// ConnectionName, and unregisters everything before returning. Credentials are
// therefore never persisted in cb-spider — honeybee remains the only store
// (encrypted at rest). Per-call unique names make concurrent calls collision-free.
func withSpiderConnection(sg *model.SourceGroup, zoneOverride string, fn func(connName string) error) error {
	if sg == nil || sg.Type != serverCommon.SourceGroupTypeCSP {
		return errors.New("source group is not a csp-type group")
	}
	provider, err := spider.NormalizeProvider(sg.ProviderName)
	if err != nil {
		return err
	}

	regionKV, err := buildRegionKV(provider, sg.RegionName, zoneOverride)
	if err != nil {
		return err
	}

	plainKV, err := loadCSPCredential(sg)
	if err != nil {
		return err
	}

	driverName, err := spider.EnsureDriver(provider)
	if err != nil {
		return errors.New("failed to ensure driver: " + err.Error())
	}

	token := uuid.New().String()
	credName := "honeybee-tmp-cred-" + token
	regionName := "honeybee-tmp-region-" + token
	connName := "honeybee-tmp-conn-" + token

	if _, err := spider.RegisterCredential(credName, provider, toSpiderKV(plainKV)); err != nil {
		return errors.New("failed to register temporary credential on cb-spider: " + err.Error())
	}
	defer func() {
		if err := spider.UnregisterCredential(credName); err != nil {
			logger.Println(logger.WARN, true, "failed to unregister temporary spider credential: "+err.Error())
		}
	}()

	if _, err := spider.RegisterRegion(regionName, provider, regionKV); err != nil {
		return errors.New("failed to register temporary region on cb-spider: " + err.Error())
	}
	defer func() {
		if err := spider.UnregisterRegion(regionName); err != nil {
			logger.Println(logger.WARN, true, "failed to unregister temporary spider region: "+err.Error())
		}
	}()

	cfg := spider.ConnectionConfigInfo{
		ConfigName:     connName,
		ProviderName:   provider,
		DriverName:     driverName,
		CredentialName: credName,
		RegionName:     regionName,
	}
	if _, err := spider.RegisterConnectionConfig(cfg); err != nil {
		return errors.New("failed to register temporary connectionconfig on cb-spider: " + err.Error())
	}
	defer func() {
		if err := spider.UnregisterConnectionConfig(connName); err != nil {
			logger.Println(logger.WARN, true, "failed to unregister temporary spider connectionconfig: "+err.Error())
		}
	}()

	return fn(connName)
}
