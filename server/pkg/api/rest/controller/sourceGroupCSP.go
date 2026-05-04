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

// spiderNamesForSourceGroup returns the deterministic spider Credential and
// ConnectionConfig names tied to the given SourceGroup ID.
func spiderNamesForSourceGroup(sgID string) (credName, connName string) {
	return "honeybee-cred-" + sgID, "honeybee-conn-" + sgID
}

// registerCSPOnSpider provisions Credential + ConnectionConfig for a CSP
// SourceGroup in cb-spider. On any failure it cleans up partially-created
// spider resources.
//
// Returns plaintext credential KV (canonical-cased) — the caller is expected to
// encrypt it before persisting to honeybee's DB.
//
// sg is mutated in-place to record the spider names, canonical provider/region.
func registerCSPOnSpider(sg *model.SourceGroup, plainKV []model.KeyValue) error {
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

	driverName, err := spider.EnsureDriver(provider)
	if err != nil {
		return errors.New("failed to ensure driver: " + err.Error())
	}

	credName, connName := spiderNamesForSourceGroup(sg.ID)

	// Credential
	if _, err := spider.RegisterCredential(credName, provider, toSpiderKV(canonicalKV)); err != nil {
		return errors.New("failed to register credential on cb-spider: " + err.Error())
	}

	// Region (some drivers reuse a single region row; we name it after the SG too)
	regionEntry := credName // reuse name for simplicity; region row name is opaque
	regionKV := []spider.KeyValue{{Key: "Region", Value: region}}
	if _, err := spider.RegisterRegion(regionEntry, provider, regionKV); err != nil {
		_ = spider.UnregisterCredential(credName)
		return errors.New("failed to register region on cb-spider: " + err.Error())
	}

	// ConnectionConfig
	cfg := spider.ConnectionConfigInfo{
		ConfigName:     connName,
		ProviderName:   provider,
		DriverName:     driverName,
		CredentialName: credName,
		RegionName:     regionEntry,
	}
	if _, err := spider.RegisterConnectionConfig(cfg); err != nil {
		_ = spider.UnregisterRegion(regionEntry)
		_ = spider.UnregisterCredential(credName)
		return errors.New("failed to register connectionconfig on cb-spider: " + err.Error())
	}

	// Mutate sg with canonical values + spider names. The caller writes them to DB.
	sg.ProviderName = provider
	sg.RegionName = region
	sg.SpiderCredentialName = credName
	sg.SpiderConnectionName = connName
	// Canonical-cased KV stored on the SourceGroup. Caller encrypts.
	sg.Credential = canonicalKV
	return nil
}

// unregisterCSPFromSpider removes the spider resources tied to the SourceGroup.
// Failures are logged but do not block deletion of the local SourceGroup.
func unregisterCSPFromSpider(sg *model.SourceGroup) {
	if sg == nil || sg.Type != serverCommon.SourceGroupTypeCSP {
		return
	}
	if sg.SpiderConnectionName != "" {
		if err := spider.UnregisterConnectionConfig(sg.SpiderConnectionName); err != nil {
			logger.Println(logger.WARN, true, "failed to unregister spider connectionconfig: "+err.Error())
		}
	}
	if sg.SpiderCredentialName != "" {
		if err := spider.UnregisterRegion(sg.SpiderCredentialName); err != nil {
			logger.Println(logger.WARN, true, "failed to unregister spider region: "+err.Error())
		}
		if err := spider.UnregisterCredential(sg.SpiderCredentialName); err != nil {
			logger.Println(logger.WARN, true, "failed to unregister spider credential: "+err.Error())
		}
	}
}
