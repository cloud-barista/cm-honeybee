package controller

import (
	"encoding/base64"
	"errors"
	"sort"
	"strings"

	serverCommon "github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/cloud-barista/cm-honeybee/server/lib/rsautil"
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
	if meta == nil || len(meta.Credential) == 0 {
		// Without metainfo we trust the caller.
		return in, nil
	}

	required := make(map[string]string, len(meta.Credential)) // upper -> canonical
	for _, k := range meta.Credential {
		required[strings.ToUpper(k)] = k
	}

	out := make([]model.KeyValue, 0, len(in))
	provided := make(map[string]bool, len(in))
	for _, kv := range in {
		canonical, ok := required[strings.ToUpper(strings.TrimSpace(kv.Key))]
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
	for _, canonical := range required {
		if !provided[canonical] {
			missing = append(missing, canonical)
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

// encryptCredentialValues encrypts each KV value with the server's RSA public
// key, base64-encoded. The keys are left in plaintext.
func encryptCredentialValues(in []model.KeyValue) ([]model.KeyValue, error) {
	out := make([]model.KeyValue, 0, len(in))
	for _, kv := range in {
		enc, err := rsautil.EncryptWithPublicKey([]byte(kv.Value), serverCommon.PubKey)
		if err != nil {
			return nil, errors.New("failed to encrypt credential value for key " + kv.Key + ": " + err.Error())
		}
		out = append(out, model.KeyValue{
			Key:   kv.Key,
			Value: base64.StdEncoding.EncodeToString(enc),
		})
	}
	return out, nil
}

// decryptCredentialValues reverses encryptCredentialValues: each base64-encoded,
// RSA-encrypted value is decrypted back to plaintext. Keys are left untouched.
func decryptCredentialValues(in []model.KeyValue) ([]model.KeyValue, error) {
	if serverCommon.PrivKey == nil {
		return nil, errors.New("server private key is not loaded; cannot decrypt CSP credential")
	}
	out := make([]model.KeyValue, 0, len(in))
	for _, kv := range in {
		raw, err := base64.StdEncoding.DecodeString(kv.Value)
		if err != nil {
			return nil, errors.New("failed to base64-decode credential value for key " + kv.Key + ": " + err.Error())
		}
		dec, err := rsautil.DecryptWithPrivateKey(raw, serverCommon.PrivKey)
		if err != nil {
			return nil, errors.New("failed to decrypt credential value for key " + kv.Key + ": " + err.Error())
		}
		out = append(out, model.KeyValue{Key: kv.Key, Value: string(dec)})
	}
	return out, nil
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

	sg.ProviderName = provider
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
	plainKV, err := decryptCredentialValues(sg.Credential)
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

	plainKV, err := decryptCredentialValues(sg.Credential)
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
