package spider

import (
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// providerCache stores upper(name) -> canonical CSP name (e.g. "AWS").
type providerCache struct {
	loaded atomic.Bool
	at     time.Time
	mu     sync.Mutex
	m      map[string]string
}

const providerCacheTTL = 30 * time.Minute

var providerCacheInstance providerCache

func loadProviderCache(force bool) error {
	providerCacheInstance.mu.Lock()
	defer providerCacheInstance.mu.Unlock()

	if !force && providerCacheInstance.loaded.Load() &&
		time.Since(providerCacheInstance.at) < providerCacheTTL {
		return nil
	}

	list, err := ListCloudOS()
	if err != nil {
		return err
	}
	m := make(map[string]string, len(list))
	for _, n := range list {
		m[strings.ToUpper(strings.TrimSpace(n))] = n
	}
	providerCacheInstance.m = m
	providerCacheInstance.at = time.Now()
	providerCacheInstance.loaded.Store(true)
	return nil
}

// NormalizeProvider returns the cb-spider canonical CSP name for any
// case-variant input (e.g. "aws", "Aws" -> "AWS"). Errors when unsupported.
func NormalizeProvider(in string) (string, error) {
	in = strings.TrimSpace(in)
	if in == "" {
		return "", errors.New("ProviderName is empty")
	}
	if err := loadProviderCache(false); err != nil {
		return "", err
	}
	if c, ok := providerCacheInstance.m[strings.ToUpper(in)]; ok {
		return c, nil
	}
	// One retry with a fresh cache, in case spider gained a new provider.
	if err := loadProviderCache(true); err == nil {
		if c, ok := providerCacheInstance.m[strings.ToUpper(in)]; ok {
			return c, nil
		}
	}
	return "", errors.New("unsupported CSP: " + in)
}

// CanonicalKeyFromMeta returns the canonical credential key (case-corrected)
// based on the cb-spider metainfo's required keys list. Returns "" if not found.
func CanonicalKeyFromMeta(meta *CloudOSMetaInfo, key string) string {
	if meta == nil {
		return ""
	}
	target := strings.ToUpper(strings.TrimSpace(key))
	for _, k := range meta.Credential {
		if strings.ToUpper(k) == target {
			return k
		}
	}
	return ""
}
