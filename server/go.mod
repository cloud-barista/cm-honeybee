module github.com/cloud-barista/cm-honeybee/server

go 1.27.1

require (
	github.com/Azure/azure-sdk-for-go/sdk/storage/azblob v1.8.2
	github.com/cloud-barista/cb-spider v0.13.12
	github.com/cloud-barista/cm-beetle/imdl v0.1.15
	github.com/cloud-barista/cm-centipede/dmdl v0.1.0
	github.com/cloud-barista/cm-centipede/transx-ex v0.1.0
	github.com/cloud-barista/cm-grasshopper/smdl v0.1.4
	github.com/cloud-barista/cm-honeybee/agent v0.0.0-20261002095523-bbf42e221751
	github.com/docker/docker v28.5.2+incompatible
	github.com/glebarez/sqlite v1.11.0
	github.com/google/uuid v1.6.0
	github.com/jollaman999/utils v1.0.10 // Original codes from https://github.com/hcloud-classic/hcc-harp
	github.com/labstack/echo/v4 v4.16.0
	github.com/minio/minio-go/v7 v7.3.0
	github.com/pkg/sftp v1.13.11
	github.com/swaggo/echo-swagger v1.5.2
	github.com/swaggo/swag v1.16.6
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cam v1.3.186
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common v1.3.190
	golang.org/x/crypto v0.57.0
	gopkg.in/yaml.v3 v3.0.1
	gorm.io/gorm v1.31.2
)

require (
	cloud.google.com/go v0.123.0 // indirect
	cloud.google.com/go/auth v0.24.0 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.3.0 // indirect
	cloud.google.com/go/cloudquotas v1.13.0 // indirect
	cloud.google.com/go/compute/metadata v0.10.0 // indirect
	cloud.google.com/go/filestore v1.17.0 // indirect
	cloud.google.com/go/longrunning v1.3.0 // indirect
	cloud.google.com/go/monitoring v1.31.0 // indirect
	cyphar.com/go-pathrs v0.2.6 // indirect
	filippo.io/edwards25519 v1.2.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.23.2 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.14.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.13.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/monitor/azquery v1.2.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v8 v8.4.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v9 v9.6.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dns/armdns v1.2.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/mysql/armmysqlflexibleservers v1.2.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v9 v9.0.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/privatedns/armprivatedns v1.3.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v3 v3.0.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage/v3 v3.0.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/subscription/armsubscription v1.2.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/storage/internal v0.1.0 // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest/to v0.4.1 // indirect
	github.com/AzureAD/microsoft-authentication-library-for-go v1.10.1 // indirect
	github.com/IBM/cloud-databases-go-sdk v0.8.1 // indirect
	github.com/IBM/go-sdk-core/v5 v5.23.5 // indirect
	github.com/IBM/platform-services-go-sdk v0.103.4 // indirect
	github.com/IBM/vpc-go-sdk v0.92.0 // indirect
	github.com/KyleBanks/depth v1.2.1 // indirect
	github.com/NaverCloudPlatform/ncloud-sdk-go-v2 v1.6.31 // indirect
	github.com/alibabacloud-go/alibabacloud-gateway-spi v0.0.5 // indirect
	github.com/alibabacloud-go/cs-20151215/v4 v4.5.8 // indirect
	github.com/alibabacloud-go/darabonba-openapi/v2 v2.2.4 // indirect
	github.com/alibabacloud-go/debug v1.0.1 // indirect
	github.com/alibabacloud-go/ecs-20140526/v4 v4.26.10 // indirect
	github.com/alibabacloud-go/endpoint-util v1.1.1 // indirect
	github.com/alibabacloud-go/openapi-util v0.1.2 // indirect
	github.com/alibabacloud-go/tea v1.5.3 // indirect
	github.com/alibabacloud-go/tea-utils/v2 v2.0.9 // indirect
	github.com/alibabacloud-go/vpc-20160428/v6 v6.16.0 // indirect
	github.com/aliyun/alibaba-cloud-sdk-go v1.63.107 // indirect
	github.com/aliyun/credentials-go v1.4.13 // indirect
	github.com/apache/arrow-go/v18 v18.8.0 // indirect
	github.com/apparentlymart/go-cidr v1.1.1 // indirect
	github.com/aws/aws-sdk-go v1.55.8 // indirect
	github.com/aws/aws-sdk-go-v2 v1.47.1 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.20.6 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.5.4 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.8.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.338.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.19 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.14.4 // indirect
	github.com/aws/smithy-go v1.28.2 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/clbanning/mxj/v2 v2.7.0 // indirect
	github.com/cloud-barista/cb-log v0.12.0 // indirect
	github.com/cloud-barista/ktcloud-sdk-go v0.2.1-0.20251103105234-6cbc5279b7fb // indirect
	github.com/cloud-barista/ktcloudvpc-sdk-go v0.0.0-20260713072217-58ff4a0ab139 // indirect
	github.com/cloud-barista/nhncloud-sdk-go v0.0.2-0.20260522111605-48d0ba7e7251 // indirect
	github.com/cyphar/filepath-securejoin v0.7.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/docker/go-connections v0.8.1 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/dustin/go-humanize v1.1.0 // indirect
	github.com/emicklei/go-restful/v3 v3.13.0 // indirect
	github.com/felixge/httpsnoop v1.1.0 // indirect
	github.com/fsnotify/fsnotify v1.10.1 // indirect
	github.com/fxamacker/cbor/v2 v2.9.4 // indirect
	github.com/gabriel-vasile/mimetype v1.4.15 // indirect
	github.com/glebarez/go-sqlite v1.23.0 // indirect
	github.com/go-logr/logr v1.4.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/errors v0.22.9 // indirect
	github.com/go-openapi/jsonpointer v1.0.2 // indirect
	github.com/go-openapi/jsonreference v1.0.3 // indirect
	github.com/go-openapi/spec v1.0.1 // indirect
	github.com/go-openapi/strfmt v0.27.2 // indirect
	github.com/go-openapi/swag v0.28.0 // indirect
	github.com/go-openapi/swag/cmdutils v0.28.0 // indirect
	github.com/go-openapi/swag/conv v0.29.2 // indirect
	github.com/go-openapi/swag/fileutils v0.28.0 // indirect
	github.com/go-openapi/swag/jsonutils v0.29.2 // indirect
	github.com/go-openapi/swag/loading v0.29.2 // indirect
	github.com/go-openapi/swag/mangling v0.28.0 // indirect
	github.com/go-openapi/swag/netutils v0.28.0 // indirect
	github.com/go-openapi/swag/pools v0.29.2 // indirect
	github.com/go-openapi/swag/stringutils v0.29.2 // indirect
	github.com/go-openapi/swag/typeutils v0.29.2 // indirect
	github.com/go-openapi/swag/yamlutils v0.29.2 // indirect
	github.com/go-playground/locales v0.14.2 // indirect
	github.com/go-playground/universal-translator v0.18.2 // indirect
	github.com/go-playground/validator/v10 v10.30.5 // indirect
	github.com/go-sql-driver/mysql v1.10.1 // indirect
	github.com/go-viper/mapstructure/v2 v2.5.0 // indirect
	github.com/goccy/go-json v0.11.2 // indirect
	github.com/gofrs/flock v0.13.1 // indirect
	github.com/golang-jwt/jwt/v5 v5.3.1 // indirect
	github.com/google/flatbuffers v25.12.19+incompatible // indirect
	github.com/google/gnostic-models v0.7.1 // indirect
	github.com/google/s2a-go v0.1.11 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.22 // indirect
	github.com/googleapis/gax-go/v2 v2.26.2 // indirect
	github.com/gophercloud/gophercloud/v2 v2.15.0 // indirect
	github.com/gorilla/websocket v1.5.4-0.20250319132907-e064f32e3674 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.8 // indirect
	github.com/hashicorp/go-version v1.9.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.11.0 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.20.1 // indirect
	github.com/klauspost/cpuid/v2 v2.4.0 // indirect
	github.com/klauspost/crc32 v1.3.0 // indirect
	github.com/kr/fs v0.1.0 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/labstack/gommon v0.5.0 // indirect
	github.com/leodido/go-urn v1.5.0 // indirect
	github.com/mattn/go-colorable v0.1.15 // indirect
	github.com/mattn/go-isatty v0.0.24 // indirect
	github.com/minio/crc64nvme v1.1.1 // indirect
	github.com/minio/md5-simd v1.1.2 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/ncruces/go-strftime v1.1.0 // indirect
	github.com/oklog/ulid/v2 v2.1.2 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/opencontainers/selinux v1.15.1 // indirect
	github.com/opentracing/opentracing-go v1.2.1-0.20220228012449-10b1cf09e00b // indirect
	github.com/oracle/oci-go-sdk/v65 v65.126.1 // indirect
	github.com/philhofer/fwd v1.2.0 // indirect
	github.com/pierrec/lz4/v4 v4.1.31 // indirect
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/rs/xid v1.6.0 // indirect
	github.com/rs/zerolog v1.35.1 // indirect
	github.com/sirupsen/logrus v1.10.2 // indirect
	github.com/snowzach/rotatefilehook v0.0.0-20220211133110-53752135082d // indirect
	github.com/sony/gobreaker/v2 v2.4.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/sv-tools/openapi v0.4.0 // indirect
	github.com/swaggo/files/v2 v2.0.2 // indirect
	github.com/swaggo/swag/v2 v2.0.0-rc6 // indirect
	github.com/tencentcloud/tencentcloud-sdk-go-intl-en v3.0.1499+incompatible // indirect
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/as v1.3.173 // indirect
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cbs v1.3.179 // indirect
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cdb v1.3.183 // indirect
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cfs v1.3.188 // indirect
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb v1.3.187 // indirect
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm v1.0.1064 // indirect
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tag v1.3.110 // indirect
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tke v1.3.180 // indirect
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc v1.3.188 // indirect
	github.com/tinylib/msgp v1.6.5 // indirect
	github.com/tjfoc/gmsm v1.4.1 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.2 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.2.0 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/youmark/pkcs8 v0.0.0-20240726163527-a2c0da244d78 // indirect
	github.com/zeebo/xxh3 v1.1.0 // indirect
	go.mongodb.org/mongo-driver/v2 v2.7.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.71.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.71.0 // indirect
	go.opentelemetry.io/otel v1.46.0 // indirect
	go.opentelemetry.io/otel/metric v1.46.0 // indirect
	go.opentelemetry.io/otel/trace v1.46.0 // indirect
	go.yaml.in/yaml/v2 v2.4.4 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/exp v0.0.0-20260908205506-85c1c2202aba // indirect
	golang.org/x/mod v0.41.0 // indirect
	golang.org/x/net v0.59.0 // indirect
	golang.org/x/oauth2 v0.37.0 // indirect
	golang.org/x/sync v0.23.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
	golang.org/x/term v0.46.0 // indirect
	golang.org/x/text v0.42.0 // indirect
	golang.org/x/time v0.16.0 // indirect
	golang.org/x/tools v0.50.0 // indirect
	google.golang.org/api v0.300.0 // indirect
	google.golang.org/genproto v0.0.0-20260928230214-8a89bd6388cc // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260928230214-8a89bd6388cc // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260928230214-8a89bd6388cc // indirect
	google.golang.org/grpc v1.84.0 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
	gopkg.in/evanphx/json-patch.v4 v4.13.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.67.3 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gorm.io/driver/postgres v1.6.3 // indirect
	k8s.io/api v0.37.1 // indirect
	k8s.io/apimachinery v0.37.1 // indirect
	k8s.io/client-go v0.37.1 // indirect
	k8s.io/klog/v2 v2.140.0 // indirect
	k8s.io/kube-openapi v0.0.0-20260721132016-d427ff9ee9ad // indirect
	k8s.io/utils v0.0.0-20260707023825-cf1189d6abe3 // indirect
	modernc.org/libc v1.77.1 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.12.1 // indirect
	modernc.org/sqlite v1.60.1 // indirect
	sigs.k8s.io/json v0.0.0-20260909141634-11ed52e25bc5 // indirect
	sigs.k8s.io/randfill v1.0.0 // indirect
	sigs.k8s.io/structured-merge-diff/v6 v6.4.2 // indirect
	sigs.k8s.io/yaml v1.6.0 // indirect
)

replace github.com/alibabacloud-go/ecs-20140526/v4 => github.com/alibabacloud-go/ecs-20140526/v4 v4.0.1
