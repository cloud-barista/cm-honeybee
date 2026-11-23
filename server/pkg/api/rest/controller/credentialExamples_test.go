package controller

import (
	"testing"

	"github.com/cloud-barista/cm-honeybee/server/lib/spider"
)

// Every CSP a csp group can be of has an example and a description for each
// required credential key GET /csp/{name} lists.
func TestCredentialExamples_CoverProviders(t *testing.T) {
	useSpiderRoot(t)

	for _, p := range providers {
		if p == providerOnPrem {
			continue
		}
		canonical, err := spider.NormalizeProvider(p)
		if err != nil {
			t.Fatal(err)
		}
		meta, err := spider.GetCloudOSMetaInfo(canonical)
		if err != nil {
			t.Fatal(err)
		}
		for _, k := range meta.CredentialCSP {
			if ex, ok := credentialExamples[canonical][k]; !ok || ex.example == "" || ex.description == "" {
				t.Errorf("%s: no example for credential key %s", canonical, k)
			}
		}
	}
}
