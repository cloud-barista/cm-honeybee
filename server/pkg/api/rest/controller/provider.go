package controller

import (
	"slices"
	"strings"
)

// providerOnPrem is the provider_name of sources the operator runs on their own
// servers rather than at a CSP.
const providerOnPrem = "onprem"

// providers are the provider_name values a minio or db source group accepts.
//
// The spelling must match the provider constants of transx-ex
// (dbmsx/driver/base) letter for letter: a name resolved here is handed through
// unchanged, so a difference in case or wording would split one provider into
// two. transx-ex checks its side in TestProviderConstants_MatchHoneybeeSpelling.
//
// minio needs every name here to be handled by resolveS3Endpoint as well.
var providers = []string{
	"aws", "alibaba", "tencent", "ncp", "nhn", "ibm", "gcp", "kt",
	"azure",
	"openstack", providerOnPrem,
}

// isSupportedProvider reports whether provider (already lower-cased) is one of
// providers.
func isSupportedProvider(provider string) bool {
	return slices.Contains(providers, provider)
}

// unsupportedProviderMsg is the error for a provider_name outside providers.
func unsupportedProviderMsg(provider string) string {
	return "unsupported provider_name '" + provider + "'. (must be one of " +
		strings.Join(providers, ", ") + ")"
}
