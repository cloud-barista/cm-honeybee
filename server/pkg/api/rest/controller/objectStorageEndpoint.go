package controller

import (
	"errors"
	"net/url"
	"strings"

	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
)

// regionRequired reports whether a minio source group of this provider needs
// region_name: the endpoint host carries the region (aws, alibaba, tencent,
// ncp, nhn, ibm), or requests must be signed with it (gcp).
func regionRequired(provider string) bool {
	switch provider {
	case "aws", "alibaba", "tencent", "ncp", "nhn", "ibm", "gcp":
		return true
	}
	return false
}

// normalizeS3Endpoint reduces os_endpoint to the host[:port] minio-go accepts.
// os_endpoint may be a bare host or an http(s) URL, and minio-go rejects a value
// with a scheme ("Endpoint url cannot have fully qualified paths"), so the
// scheme is taken off here. When there was one, sslFromScheme is true and useSSL
// follows it. A path, query or fragment is an error: minio-go cannot take one.
func normalizeS3Endpoint(raw string) (host string, useSSL bool, sslFromScheme bool, err error) {
	raw = strings.TrimSpace(raw)

	if strings.Contains(raw, "://") {
		u, err := url.Parse(raw)
		if err != nil {
			return "", false, false, errors.New("os_endpoint is not a valid URL: " + err.Error())
		}
		switch strings.ToLower(u.Scheme) {
		case "http":
			useSSL = false
		case "https":
			useSSL = true
		default:
			return "", false, false, errors.New("os_endpoint scheme must be http or https")
		}
		if (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" {
			return "", false, false, errors.New("os_endpoint must not contain a path")
		}
		if u.Host == "" {
			return "", false, false, errors.New("os_endpoint has no host")
		}

		return u.Host, useSSL, true, nil
	}

	host = strings.TrimSuffix(raw, "/")
	if strings.ContainsAny(host, "/?#") {
		return "", false, false, errors.New("os_endpoint must not contain a path")
	}
	if host == "" {
		return "", false, false, errors.New("os_endpoint has no host")
	}

	return host, false, false, nil
}

// resolveS3Endpoint works out how to reach a minio connection's object storage
// from its source group's provider_name and region_name and the connection's
// os_* fields. The hosts follow cb-spider's GetS3ConnectionInfo.
//
// sg.ProviderName and sg.RegionName are folded to lower case and trimmed here,
// as the source group handler stores them, so a row written another way still
// resolves. The regionRequired check before the switch means no case has to
// test sgRegion again.
//
// region is the signing region. It is left empty unless the provider needs it
// (aws, tencent, nhn, gcp — the providers cb-spider marks RegionRequired): with
// no region, minio-go asks the bucket for its location and signs with the
// answer, which is the only way to reach a store whose signing region differs
// from the one in its host (NCP serves "kr.…" and signs "kr-standard"). The
// cost is one extra request per client, and a location probe the key may not
// call (AccessDenied) quietly falls back to us-east-1.
//
// bucketLookup is "" (auto), "dns" or "path", as transx-ex S3MinioConfig takes it.
//
// os_endpoint, when set, overrides the derived host, except for azure whose
// host is derived from the storage account name in os_access_key_id. It is
// required for onprem and openstack, which have no host to derive.
func resolveS3Endpoint(sg *model.SourceGroup, ci *model.ConnectionInfo) (endpoint string, region string, useSSL bool, bucketLookup string, err error) {
	if sg == nil || ci == nil {
		return "", "", false, "", errors.New("source group or connection info is empty")
	}

	provider := strings.ToLower(strings.TrimSpace(sg.ProviderName))
	sgRegion := strings.ToLower(strings.TrimSpace(sg.RegionName))

	if regionRequired(provider) && sgRegion == "" {
		return "", "", false, "", errors.New("region_name is required for provider '" + provider + "'")
	}

	useSSL = true

	switch provider {
	case "aws":
		endpoint = "s3." + sgRegion + ".amazonaws.com"
		region = sgRegion
	case "alibaba":
		endpoint = "oss-" + sgRegion + ".aliyuncs.com"
	case "tencent":
		// The bucket name must carry the AppId ("<name>-<AppId>"). The connection
		// supplies it in os_scan_bucket, so the bucket is addressed DNS-style.
		endpoint = "cos." + sgRegion + ".myqcloud.com"
		region = sgRegion
		bucketLookup = "dns"
	case "ncp":
		endpoint = sgRegion + ".object.ncloudstorage.com"
	case "nhn":
		endpoint = sgRegion + "-api-object-storage.nhncloudservice.com"
		region = sgRegion
	case "ibm":
		endpoint = "s3." + sgRegion + ".cloud-object-storage.appdomain.cloud"
	case "gcp":
		// Needs HMAC interoperability keys on the connection, not a service
		// account key.
		endpoint = "storage.googleapis.com"
		region = sgRegion
	case "kt":
		endpoint = "obj-e-1.ktcloud.com"
	case "azure":
		// transx-ex switches to azblob on a *.blob.core.windows.net endpoint.
		if ci.OSAccessKeyId == "" {
			return "", "", false, "", errors.New("os_access_key_id (storage account name) is required for provider 'azure'")
		}
		return ci.OSAccessKeyId + ".blob.core.windows.net", "", true, "", nil
	case "openstack", providerOnPrem:
		// openstack is listed apart from onprem so that the endpoint can later be
		// derived from the identity endpoint, as cb-spider does.
		if strings.TrimSpace(ci.OSEndpoint) == "" {
			return "", "", false, "", errors.New("os_endpoint is required for provider '" + provider + "'")
		}
		useSSL = ci.OSUseSSL
	default:
		return "", "", false, "", errors.New(unsupportedProviderMsg(provider))
	}

	if strings.TrimSpace(ci.OSEndpoint) != "" {
		host, ssl, sslFromScheme, err := normalizeS3Endpoint(ci.OSEndpoint)
		if err != nil {
			return "", "", false, "", err
		}
		endpoint = host
		if sslFromScheme {
			useSSL = ssl
		}
	}

	return endpoint, region, useSSL, bucketLookup, nil
}
