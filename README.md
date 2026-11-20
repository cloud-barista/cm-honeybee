# Collecting and Aggregating Information From Source Computing
This repository provides a features of collection and aggregation for all source computing information. This is a sub-system on Cloud-Barista platform and utilizes CM-Beetle to migrate a multi-cloud.

## Overview

Collecting and Aggregating Information From Source Computing framework (codename: cm-honeybee) is going to support:

* collect and aggregate information from source computing about infrastructure, software, data
* provides the Agent for collecting source computing information

<details>
    <summary>Terminology</summary>

* Source Computing  
  The source computing, serving as the target for configuration and information collection, for the migration to multi-cloud
* Target Computing  
  The target computing is migration target as multi-cloud

</details>

## Execution and development environment
* Tested operating systems (OSs):
  * Ubuntu 24.04, Ubuntu 22.04, Ubuntu 18.04, Rocky Linux 9, Windows 11
* Language:
  * Go: 1.27.1

## How to run

### 1. Build and run server

1.1. Write the configuration file. (Optional)

(You can skip this step and the default settings will be used instead.)

- Configuration file name is 'cm-honeybee.yaml'
- The configuration file must be placed in one of the following directories.
    - .cm-honeybee/conf directory under user's home directory
    - 'conf' directory where running the binary
    - 'conf' directory where placed in the path of 'CMHONEYBEE_ROOT' environment variable
- Configuration options
    - listen
        - port : Listen port of the server's API.
    - agent
        - port : Fallback port of the agent's API. The agent binds a
          kernel-chosen free port and publishes it to
          `/etc/cloud-migrator/cm-honeybee-agent/port`, which the server reads
          over SSH before each request. This value is used only when that file
          is absent, i.e. against an agent old enough to sit on a fixed port.
    - The `spider` block (endpoint, username, password) of earlier versions is
      no longer read. honeybee links the cb-spider drivers and calls the CSPs
      itself, so no cb-spider server is needed. A config file that still has
      the block loads as before; the block is ignored.
- Configuration file example
  ```yaml
  cm-honeybee:
      listen:
          port: 8081
      agent:
          # Fallback only; see the option description above.
          port: 8082
  ```

1.2. Prepare the cb-spider driver files

CSP-type source groups go through the cb-spider drivers linked into the server.
The drivers read files from two directories given by environment variables.

- `CBSPIDER_ROOT` : `cloud-driver-libs/cloudos.yaml`, `cloudos_meta.yaml`,
  `cloud-driver-libs/region/*_region_meta.yaml` and `conf/calllog_conf.yaml`.
  The drivers also write `meta_db/` and `log/` here.
  When unset, the server uses `$CMHONEYBEE_ROOT/cb-spider`, or
  `~/.cm-honeybee/cb-spider` when `CMHONEYBEE_ROOT` is unset too.
- `CBLOG_ROOT` : `conf/log_conf.yaml` for cb-log.

`make run` (in `server/`) runs `make cb-spider-root`, which copies the files from
the cb-spider version pinned in `go.mod` into `$CBSPIDER_ROOT` (or the default
directory above), and starts the binary with both variables set to it. The Docker
image already carries the files in `/cb-spider` and sets both variables to
`/cb-spider`. `/cb-spider/log` in the image is not on a volume, so the driver
logs are lost when the container is recreated.

When you launch the binary yourself, run `make cb-spider-root` first and point
both variables at the same directory. cb-log exits the server at startup when
- `CBLOG_ROOT` is set but `$CBLOG_ROOT/conf/log_conf.yaml` does not exist, or
- `log_conf.yaml` refers to `$CBSPIDER_ROOT` and the variable ends up empty
  (the default above could not be set, e.g. no home directory).

If `CBLOG_ROOT` is unset, cb-log falls back to its built-in settings.

1.3. Build and run the server binary
```shell
cd server
make run
```

Or, you can run it within Docker by this command.
 ```shell
 make run_docker
 ```

### 2. Register source group
Check your source group ID (sgID) after register.

A source group has a `type` field:
- `onprem` - on-premise sources. A connection is either a host reached over SSH
  (`resource_type: vm`) or a Kubernetes cluster registered by its kubeconfig
  (`resource_type: k8s`).
- `csp` - collects from cloud sources (VM / Kubernetes / Object Storage / managed database / load balancer) through the cb-spider drivers linked into the server.

`onprem`, `ssh` and an omitted `type` are all treated as on-premise, so payloads
written against earlier versions keep working unchanged. An omitted `type` is
still stored and returned as `ssh`; send `onprem` explicitly to store that value.

`GET /source_group/types` describes every source group type (`onprem`, `csp`,
`fs`, `db`, `minio`): the fields the group itself takes, the connection types it
holds and the fields each connection type takes (`auth_fields`, `fields`, each
with `required`, `secret`, `enum`, `default` and the like), so a client can build
its registration forms from it. The credential keys of a CSP come from
`GET /csp/{name}` (2.2).

`GET /source_group?type=<type>` lists the groups of one type. `type=onprem` also
matches the groups stored as `ssh` or without a type.

Every group response (`POST`, `GET`, `PUT` and the list) also carries
- `connection_count_by_type`: the number of connections per connection type. For
  `onprem` and `csp` groups that is the `resource_type` (an on-premise connection
  stored without one counts as `vm`); for `fs`, `db` and `minio` groups, whose
  connections take no `resource_type`, it is the group type.
- `capabilities`: which collections the group can run with the connections it holds
  now - `isCollectInfra`, `isCollectSoftware`, `isCollectKubernetes`,
  `isCollectObjectStorage`, `isCollectDb`, `isCollectFs`, `isCollectNlb`. A group
  without a connection a collection acts on reports `false` for it, so an empty
  group reports `false` for all of them.

Deleting a group or a connection also deletes everything the imports stored for
it (infra, software, kubernetes, helm, data, rdbms and nlb records).

#### 2.1 On-premise source group
- Request
```shell
curl -X 'POST' \
  'http://127.0.0.1:8081/honeybee/source_group' \
  -H 'accept: application/json' \
  -H 'Content-Type: application/json' \
  -d '{
  "description": "test migration group",
  "name": "test-group"
}'
```
- Reply
```json
{
 "id": "b9e86d53-9fbe-4a96-9e06-627f77fdd6b7",
 "name": "test-group",
 "description": "test migration group",
 "type": "ssh",
  "connection_info_status_count": {
    "count_connection_success": 0,
    "count_connection_failed": 0,
    "count_agent_success": 0,
    "count_agent_failed": 0,
    "connection_info_total": 0
  }
}
```

#### 2.2 CSP source group
A CSP group represents one CSP connection (provider + region + credential).
Resources under the group (VMs, Kubernetes clusters, object-storage buckets,
managed databases, load balancers) are
registered as individual `connection_info` entries.

Discover the credential keys required by the target CSP first:
```shell
curl 'http://127.0.0.1:8081/honeybee/csp'              # supported CSP names, lowercase
curl 'http://127.0.0.1:8081/honeybee/csp/aws'          # case-insensitive
```
The response includes `credential_keys`, read from the cb-spider driver metadata. Their
spelling differs per CSP: AWS uses lowercase snake_case
(`aws_access_key_id`, `aws_secret_access_key`) while Azure uses camelCase
(`clientId`, `clientSecret`, `tenantId`, `subscriptionId`).

`optional_credential_keys` lists the keys a group may take besides them. On
ibm, openstack, kt, gcp, nhn and azure the buckets are read through an S3 API
that the main credential does not reach, so these CSPs take `S3AccessKey` and
`S3SecretKey` (the Cloud Storage interoperability keys of cb-tumblebug's
`template.credentials.yaml`; on azure the storage account name and key, also
taken as `StorageAccountName` and `StorageAccountKey`). They are optional: a group without them registers, lists
regions and collects every other resource type, but its `object_storage`
connections are refused (3.3). Give both or neither; one alone is refused with
400 (`S3AccessKey and S3SecretKey must be given together`). The key names are
matched case-insensitively and stored as `S3AccessKey` and `S3SecretKey`. The
other CSPs read buckets with the main credential and have no optional keys;
an S3 key sent to them is refused like any unknown key. `credentials` lists
the required keys, then the optional ones with `optional: true` and
`resource_types: ["object_storage"]`.

It also includes `region_keys`, which is the shape of a region for that CSP
(e.g. `["Region", "Zone"]`) rather than a list of regions, and `default_region`.
The actual regions need a credential. Before the group exists, post the
credential to `POST /csp/{name}/region`; it is used for that one query and is
neither stored nor logged. After the group is created,
`GET /source_group/{sgID}/region` uses the stored one. Both return the same shape.
```shell
curl -X 'POST' 'http://127.0.0.1:8081/honeybee/csp/aws/region' \
  -H 'Content-Type: application/json' \
  -d '{ "credential": [
        {"key": "aws_access_key_id",     "value": "$AWS_ACCESS_KEY_ID"},
        {"key": "aws_secret_access_key", "value": "$AWS_SECRET_ACCESS_KEY"} ] }'
```

`GET /csp` also returns `supported_resource_types`, the `resource_type` values a
connection of each CSP can be collected as (3.3). It is read from the linked
driver's capabilities; `object_storage` is listed for every CSP whose buckets are
read through the S3 API, including the ones that need the optional S3 keys above.

Then create the group:
```shell
curl -X 'POST' \
  'http://127.0.0.1:8081/honeybee/source_group' \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "aws-prod-seoul",
    "type": "csp",
    "provider_name": "AWS",
    "region_name": "ap-northeast-2",
    "credential": [
      {"key": "aws_access_key_id",     "value": "$AWS_ACCESS_KEY_ID"},
      {"key": "aws_secret_access_key", "value": "$AWS_SECRET_ACCESS_KEY"}
    ]
  }'
```
honeybee stores the credential in OpenBao and keeps it there. For each call it
hands the credential to the CSP driver in memory; it is not registered or
written anywhere else. No connection name is kept on the source group.
### 3. Register connection info
Register the connection information to the source group. The body shape
depends on the group's `type`.

#### 3.1 On-premise SSH connection info
`resource_type` defaults to `vm` and may be omitted. A response always carries it:
an on-premise connection stored without one, registered before the field
existed, is returned as `vm`. Agent will be installed automatically.
- Request
```shell
curl -X 'POST' \
 'http://127.0.0.1:8081/honeybee/source_group/b9e86d53-9fbe-4a96-9e06-627f77fdd6b7/connection_info' \
 -H 'accept: application/json' \
 -H 'Content-Type: application/json' \
 -d '{ "description": "NFS Server", "ip_address": "192.0.2.123", "name": "cm-nfs", "password": "$SSH_PASSWORD", "private_key": "$SSH_PRIVATE_KEY", "ssh_port": "22", "user": "ubuntu" }'
```
- Reply
```json
{
  "id": "2f678139-e6e6-43e8-9722-33b834efc563",
  "name": "cm-nfs",
  "description": "NFS Server",
  "source_group_id": "b9e86d53-9fbe-4a96-9e06-627f77fdd6b7",
  "ip_address": "192.0.2.123",
  "ssh_port": "22",
  "user": "XXXXXXXX...=",
  "password": "XXXXXXXX...=",
  "private_key": "XXXXXXXX...=",
  "public_key": "",
  "connection_status": "success",
  "connection_failed_message": "",
  "agent_status": "success",
  "agent_failed_message": ""
}
```

#### 3.2 On-premise Kubernetes connection info
Register an on-premise cluster by its kubeconfig instead of an SSH host. The
kubeconfig is parsed and rejected if it is empty or malformed.
```shell
curl -X 'POST' \
 'http://127.0.0.1:8081/honeybee/source_group/{sgID}/connection_info' \
 -H 'Content-Type: application/json' \
 -d '{ "name": "onprem-k8s", "resource_type": "k8s", "kubeconfig": "apiVersion: v1\nkind: Config\n..." }'
```
There is no SSH host to reach, so the agent is not installed and the reply
reports that explicitly:
```json
{
  "resource_type": "k8s",
  "kubeconfig": "XXXXXXXX...=",
  "connection_status": "success",
  "connection_failed_message": "",
  "agent_status": "failed",
  "agent_failed_message": "agent-based collection is not applicable to on-prem k8s connections"
}
```
`POST .../import/*` collects through the in-guest agent over SSH, which such a
connection does not have. Calling it for this connection alone is refused with
the message above; calling it for the whole source group skips this member and
collects the rest. The stored kubeconfig is consumed by the downstream
migration modules.

#### 3.3 CSP connection info
For CSP groups, each `connection_info` points at one cloud resource by id.
You can list available resources first via the discovery endpoint:
```shell
curl 'http://127.0.0.1:8081/honeybee/source_group/{sgID}/discover?resource_type=vm'
curl 'http://127.0.0.1:8081/honeybee/source_group/{sgID}/discover?resource_type=k8s'
curl 'http://127.0.0.1:8081/honeybee/source_group/{sgID}/discover?resource_type=object_storage'
curl 'http://127.0.0.1:8081/honeybee/source_group/{sgID}/discover?resource_type=rdbms'
curl 'http://127.0.0.1:8081/honeybee/source_group/{sgID}/discover?resource_type=nlb'
```
When the CSP's driver has no handler for the `resource_type` (`k8s`, `nlb` or
`rdbms`; e.g. `k8s` and `rdbms` on kt), discovery answers 200 with empty `items`,
`unsupported: true` and the reason in `reason`, without asking the CSP. So does
`object_storage` on a group whose CSP needs the optional S3 keys (2.2) when the
group's credential has none.
Then register the picked resource:
```shell
curl -X 'POST' \
 'http://127.0.0.1:8081/honeybee/source_group/{sgID}/connection_info' \
 -H 'Content-Type: application/json' \
 -d '{ "name": "vm-app01", "resource_type": "vm", "resource_id": "i-0abc..." }'
```
`resource_type` is one of `vm`, `k8s`, `object_storage`, `rdbms` or `nlb`.
`GET /csp` tells which of them a CSP supports (`supported_resource_types`).

NCP Kubernetes clusters do not appear in `discover?resource_type=k8s`, as the
cb-spider NCP driver lists no clusters. Such a cluster is still registered and
collected by its system ID (the cluster UUID): when the listing does not hold
the cluster, honeybee reads it by that ID.

A `vm` connection may also take `ip_address`, `ssh_port`, `user` and
`password` or `private_key`; with them the agent is installed in the VM and
collects the guest as well (section 4). The other resource types have no host to
reach, so a request that sends any SSH field for them is refused with 400.

Registration checks the resource with the CSP:
- A `resource_type` the group's CSP is not collected as (not in its
  `supported_resource_types` of `GET /csp`) is refused with 400 before the CSP
  is asked, e.g. `resource_type 'rdbms' is not supported for kt (supported: vm,
  object_storage, nlb)`. Connections stored with such a type before are kept,
  and their refresh asks the CSP as before.
- An `object_storage` connection on a group whose CSP needs the optional S3 keys
  (2.2) and whose credential has none is refused with 400 before the CSP is
  asked, e.g. `object_storage on gcp needs S3AccessKey and S3SecretKey in the
  source group credential`. Add them with `PUT /source_group/{sgID}`, whose
  `credential` replaces the stored one and so must carry the required keys as well.
- A `resource_id` the CSP does not have is refused with 400. The resource counts
  as missing only when the CSP listing came back and holds no such resource; for
  `k8s`, reading the cluster by that ID must also fail or return another cluster.
- A bucket in another region than the group's is refused with 400 (see the table
  below for the CSPs where the region is known).
- When the CSP cannot be asked (timeout, credential, unreachable endpoint), the
  connection is registered with `connection_status: failed` and the reason in
  `connection_failed_message`, and is checked again on refresh.

`PUT .../connection_info/{connId}` on a csp connection:
- On a `vm` connection it takes the SSH fields above. A field left empty keeps its
  stored value, so a credential can be replaced but not cleared. An `ip_address`,
  stored or sent, needs `user` and `password` or `private_key`; `ssh_port` defaults
  to `22`.
- On the other resource types SSH fields are refused with 400. A `vm` connection
  changed into another resource type drops its SSH fields and secrets.
- A changed `resource_type` or `resource_id` is checked as on registration, with
  the same 400 for a `resource_type` the CSP is not collected as, a bucket on a
  group without the S3 keys its CSP needs, a missing resource or a bucket in
  another region.

`GET .../connection_info` takes `resource_type` (`vm`, `k8s`, `object_storage`,
`nlb`, `rdbms`) to list one type. On an on-premise group `resource_type=vm`
also matches the connections stored without one. `ssh_port` matches the port
exactly.

Bucket regions. A source group stands for one region, but the bucket listing
of most CSPs covers the whole account. Where the CSP tells a bucket's region,
discovery lists only the buckets of the group's region, with the region in
`region`; registration refuses a bucket of another region; and the bucket is read
at the endpoint of its region. Where it cannot, the region is left unknown, every
bucket is listed, and none is refused for its region.

| Provider | Region read | How, or why not |
|----------|-------------|-----------------|
| AWS | yes | `HEAD` on the bucket at the group region's endpoint. S3 serves it there only for a bucket of that region and names the bucket's region in `x-amz-bucket-region` otherwise. |
| Azure | no | A container has no region of its own; its storage account's location is an Azure Resource Manager property the Blob API does not expose to the account key. |
| GCP | no | `?location` answers a GCS location, which is a multi- or dual-region (`US`, `ASIA`, `NAM4`) as often as a region, in another spelling than the group's region. |
| IBM | no | The location constraint carries the storage class (`us-south-standard`), not the region alone. |
| Alibaba, Tencent, NCP, NHN | no | Each has its own location naming through its S3-compatible API, not verified against the region names the group uses. |
| KT, OpenStack | no | One endpoint for the whole account: there is no region to compare with. |

A discovered bucket whose region lookup failed is still listed, with the reason
in `extra.region_error`, since it cannot be told to be elsewhere.

`PUT .../refresh` only re-checks the connection status; it stores nothing.
Collection is done by `POST .../import/infra` on one connection, or by the
group imports of section 4. Either queries the CSP and writes to a table chosen
by `resource_type`:

| `resource_type` | Stored in | Read back with |
|-----------------|-----------|----------------|
| `vm` | `SavedInfraInfo.csp_data` | `GET .../infra` (the `csp` section) |
| `k8s` | `SavedKubernetesInfo` | `GET .../kubernetes` (the `csp` section holds the node groups) |
| `object_storage` | `SavedDataInfo` | `GET .../data` |
| `rdbms` | `SavedRDBMSInfo` | `GET .../rdbms` |
| `nlb` | `SavedNLBInfo` | `GET .../nlb` |

Note that `import/kubernetes` and `import/data` are the agent-over-SSH paths and
do not apply to CSP resources - use `import/infra`, or a group import by kind.

What is collected:
- `k8s`: the managed cluster's name (its name at the CSP, or its system ID when it
  has none), version, status, creation time and node groups (spec, image, root
  disk, desired/min/max size, autoscaling). The nodes and workloads are not
  collected: the cluster is read through the CSP's API, not through a kubeconfig.
  A cluster the CSP listing does not hold is read by its ID, which is how an NCP
  cluster is collected.
- `nlb`: the listener (protocol, IP, port, DNS name), the backend VM group
  (protocol, port, VMs) and the health check. `type` is `PUBLIC`, `INTERNAL`, or
  empty when the driver does not tell. On GCP the cb-spider driver leaves the
  type empty; it is then read from the listener's `LoadBalancingScheme`
  (`EXTERNAL` is `PUBLIC`, `INTERNAL` is `INTERNAL`). On AWS the cb-spider
  driver looks the target group up by the NLB's own name, the name it gives the
  target groups it creates. For an NLB created outside cb-spider whose target group has another
  name, the driver finds no target group and does not report it as an error, so
  the import succeeds with `vm_group` `{"protocol": "", "port": "", "vms": []}`
  and a zero `health_checker`. Such a record has `vm_group_unknown: true` (with a
  warning in the server log), so an unread target group is told apart from one
  that was read but has no targets, which has its protocol and port with an empty
  `vms` and `vm_group_unknown: false`. `GET .../nlb` of a connection and of the
  group carry the field, and so does discovery in `extra.vm_group_unknown`; a
  record stored before the field existed gets it from its empty `vm_group`. The
  refined NLBs (`nlbs`) have no place for it and show such an NLB with no
  backend servers.
- `object_storage`: the bucket's settings and, unless `count_objects=false`, its
  object count and total size. The region the bucket was found in is kept as well.

A collection that reads nothing is recorded as `failed`, not `success`:
- A bucket none of whose details could be read is stored with `status: failed`
  and its errors, and the import returns 400. Some details failing is still a
  success; the failures are kept in the stored errors. An item a CSP cannot
  answer for any bucket (versioning on Azure) does not count.
- When the CSP collection of a `k8s`, `object_storage`, `rdbms` or `nlb`
  connection fails, its record is marked `failed`. `GET` still shows the last data
  collected, while the refined APIs leave the record out.
- An import of a csp connection also sets `connection_status` (and
  `connection_failed_message`) from the outcome of the collection, not only
  registration and refresh do: a resource that can be found but not read is not a
  working connection.

### 4. Save current source information.
Below example is saving infrastructure information of all connection in the source group.
```shell
curl -X 'POST' \
 'http://127.0.0.1:8081/honeybee/source_group/b9e86d53-9fbe-4a96-9e06-627f77fdd6b7/import/infra' \
 -H 'accept: application/json'
```

- For software: POST http://X.X.X.X:8081/honeybee/source_group/{SourceGroupID}/import/software
- For Kubernetes: POST http://X.X.X.X:8081/honeybee/source_group/{SourceGroupID}/import/kubernetes
- For Helm: POST http://X.X.X.X:8081/honeybee/source_group/{SourceGroupID}/import/helm

To collect a whole group in one call, or one kind of resource of it, use the
imports and refined APIs by resource kind. All paths are under
`/honeybee/source_group/{sgID}`.

| Resource kind | Group import | Refined (group) | Refined (one connection) |
|---------------|--------------|-----------------|--------------------------|
| Everything an onprem or csp group holds | `POST import/all` | - | - |
| Hosts (onprem vm, csp vm) | `POST import/infra` | `GET infra/refined` | `GET connection_info/{connId}/infra/refined` |
| Software in hosts (agent) | `POST import/software` | `GET software/refined` | `GET connection_info/{connId}/software/refined` |
| CSP buckets (`object_storage`) | `POST import/bucket` | `GET bucket/refined` | `GET connection_info/{connId}/bucket/refined` |
| CSP managed databases (`rdbms`) | `POST import/rdbms` | `GET rdbms/refined` | `GET connection_info/{connId}/rdbms/refined` |
| CSP managed Kubernetes (`k8s`) | `POST import/k8s` | `GET k8s/refined` | `GET connection_info/{connId}/k8s/refined` |
| CSP load balancers (`nlb`) | `POST import/nlb` | `GET nlb/refined` | `GET connection_info/{connId}/nlb/refined` |
| fs / db / minio groups | `POST import/fs`, `import/db`, `import/objectstorage` | `GET fs/refined`, `db/refined`, `objectstorage/refined` | the same under `connection_info/{connId}/` |

- `import/all` collects every connection the way its type is collected: a csp
  resource other than a VM from the CSP, a VM as infra (the CSP side on a csp
  group, and through the agent when it has an `ip_address`), and a host the agent
  can reach also as software, kubernetes and helm. An on-premise `k8s` connection
  has nothing to collect and is left out. It takes the options of `import/infra`
  (`count_objects`) and of `import/software` (`show_default_packages`). An `fs`,
  `db` or `minio` group is refused with 400 and the import to use instead.
- `import/bucket`, `import/rdbms`, `import/k8s` and `import/nlb` collect only the
  csp connections of that `resource_type` and leave the rest alone. A group that
  is not `csp` is refused with 400. A Kubernetes cluster on VMs is collected
  through the agent by `import/kubernetes`.
- These five return one entry per connection: `connection_id`, `resource_type`
  and `results`, one `{kind, status, error_message}` per collection run, `kind`
  being `infra`, `software`, `kubernetes`, `helm`, `bucket`, `rdbms` or `nlb`.
- The older `import/infra` and `data/refined` stay as they were and give the same
  results: `import/infra` on a csp group collects every resource type, and
  `data/refined` is `bucket/refined`.

Every group import runs only on the connections it applies to and does not stop
at a failing one:
- `import/software`, `import/kubernetes`, `import/helm` and `import/data` go
  through the agent, so they run only on the connections the agent can be reached
  on: on-premise hosts, csp `vm` connections with an `ip_address`, and the
  connections of the other group types. The others are skipped, not reported as
  failed. Calling one of them on a single connection without an agent is refused.
- `import/infra` runs on every connection but an on-premise `k8s` one.
- A failing connection is listed with `status: failed` and the reason in
  `error_message`, and the rest go on. The request fails (400) only when no
  connection succeeded. A group with no connection to collect gives 200 and an
  empty list.

### 5. Get saved source information.
Below example is getting saved infrastructure information of all connection in the source group.
```shell
curl -X 'GET' \
 'http://127.0.0.1:8081/honeybee/source_group/b9e86d53-9fbe-4a96-9e06-627f77fdd6b7/infra' \
 -H 'accept: application/json'
```

- For software: GET http://X.X.X.X:8081/honeybee/source_group/{SourceGroupID}/software
- For Kubernetes: GET http://X.X.X.X:8081/honeybee/source_group/{SourceGroupID}/kubernetes
- For Helm: GET http://X.X.X.X:8081/honeybee/source_group/{SourceGroupID}/helm
- For CSP managed databases: GET http://X.X.X.X:8081/honeybee/source_group/{SourceGroupID}/rdbms
- For CSP load balancers: GET http://X.X.X.X:8081/honeybee/source_group/{SourceGroupID}/nlb

### 6. Get refined, saved source information.
Below example is getting refined, saved infrastructure information of all connection in the source group.
```shell
curl -X 'GET' \
 'http://127.0.0.1:8081/honeybee/source_group/b9e86d53-9fbe-4a96-9e06-627f77fdd6b7/infra/refined' \
 -H 'accept: application/json'
```

The refined APIs by resource kind are listed in section 4. A refined API reads
only what the last import collected successfully: a connection that has not
been collected, or whose last import failed, is left out of the group result.

`infra/refined` of a csp group:
- Only `vm` connections become nodes. A VM collected through the agent is refined
  from the agent's data, as an on-premise host is.
- A VM the agent did not collect (no `ip_address`, or only the agent call failed
  while the CSP collection succeeded) becomes a node from the CSP's data:
  hostname is the VM name, `machineId` the CSP's VM ID, the CPU is the vCPU count
  (1 CPU with as many cores and threads), memory, root and data disks, GPUs,
  firewall rules and one interface with the private IP. The public IP is not put
  in the interfaces: the CSP maps it outside the guest, and cm-beetle reads
  interface addresses as the node's own. The architecture is written as the agent
  writes it (`x86_64`, `aarch64`, not `amd64`, `arm64`).
- The OS of such a node is read from the image description, which is free text
  that differs per CSP. The distribution is found by name (Ubuntu, Debian, Rocky,
  AlmaLinux, CentOS, RHEL, Oracle Linux, Amazon Linux, SLES, Fedora, Windows) and
  the version is the first number after it, or the release codename for Ubuntu and
  Debian. When none of the image's OS description, name or ID holds a
  distribution's name, as with an Amazon Linux image known only as
  `al2023-ami-...`, the OS has no ID or version and `prettyName` keeps the
  description if the CSP gave one. Every SUSE image, openSUSE included, reads as
  `sles`.
- `k8sCluster` is the cluster the agent collected on a node; when the group has
  none, it is a CSP managed cluster with its name and version. The pod and service
  CIDRs of a managed cluster stay empty, as the CSP driver does not report them,
  and its nodes are not listed.
- `nlbs` holds the NLBs found in HAProxy on the nodes, then the CSP NLBs. A CSP NLB
  has `software` `"<provider>-nlb"` (e.g. `aws-nlb`) instead of `haproxy`, an
  empty `hostMachineId`, and listener address `*` when it is `PUBLIC`, its first
  IP (or DNS name) when it is `INTERNAL`, and empty when the type is not known. Its
  balancing algorithm is not reported. Its backend servers get the private IPs
  of the group's CSP VMs, matched by VM ID or name; a backend VM not registered in
  the group keeps an empty IP.

`k8s/refined` returns the same `k8sCluster`, with `managed` and, for a managed
cluster, a `csp` section with its node groups. `software/refined` has no
Kubernetes entry for a managed cluster, since its workloads are not collected.

`bucket/refined` (and `data/refined`) of one connection names the region the bucket
was found in as `sourceCloud.region`, or the group's region where the CSP cannot
tell. The group result names the group's region and leaves out a bucket found in
another region, since the model has one region for all of its buckets.

`rdbms/refined` of one connection names the zone the database runs in as
`sourceCloud.zone`, read from the driver's record (AWS `AvailabilityZone`, GCP
`GceZone`, Alibaba `ZoneId`, Tencent `Zone`), or the group's zone where the CSP
does not report one (Azure, IBM). The group result names the databases' zone
only when all of them report the same one, and the group's zone otherwise. The
spec (vCPU and memory) of a database is collected in its own zone as well:
Alibaba finds no spec when asked in another zone of the region.

## Health-check

### Server

Check if CM-Honeybee server is running

```bash
curl http://localhost:8081/honeybee/readyz

# Output if it's running successfully
# {"message":"CM-Honeybee API server is ready"}
```

### Agent

Check if CM-Honeybee agent is running. The agent listens on a kernel-chosen
loopback port and publishes it, so read the port instead of assuming one.

```bash
curl http://127.0.0.1:$(cat /etc/cloud-migrator/cm-honeybee-agent/port)/honeybee-agent/readyz

# Output if it's running successfully
# {"message":"CM-Honeybee Agent is ready"}
```

## Check out all APIs
* [Honeybee Agent APIs (Swagger Document)](https://cloud-barista.github.io/cb-tumblebug-api-web/?url=https://raw.githubusercontent.com/cloud-barista/cm-honeybee/main/agent/pkg/api/rest/docs/swagger.yaml)
* [Honeybee Server APIs (Swagger Document)](https://cloud-barista.github.io/cb-tumblebug-api-web/?url=https://raw.githubusercontent.com/cloud-barista/cm-honeybee/main/server/pkg/api/rest/docs/swagger.yaml)


## For Docker users
There are default private key and public key used for encrypt connection info's secret values (user, password, private key, kubeconfig) from the honeybee server.
(Located in server/_default_key)
For security, run these commands to generate new key files.

```bash
docker exec cm-honeybee rm /root/.cm-honeybee/honeybee.key
docker exec cm-honeybee rm /root/.cm-honeybee/honeybee.pub
docker restart cm-honeybee
```

If you want to use private key file with other modules like cm-grasshopper, run this command.
```bash
mkdir keys
docker cp cm-honeybee:/root/.cm-honeybee/honeybee.key keys/
docker cp cm-honeybee:/root/.cm-honeybee/honeybee.pub keys/
```

Now, mount the created folder to the honeybee server container.
For docker compose, add these lines.
```yaml
    volumes:
        - ./keys/honeybee.key:/root/.cm-honeybee/honeybee.key
        - ./keys/honeybee.pub:/root/.cm-honeybee/honeybee.pub
```

Now, you can copy ./keys/honeybee.key file to other module.

## For who develop modules with Honeybee

### About encrypted values of the connection info
Those encrypted values are always changes with each request by RSA algorithm.

### How to decrypt encrypted values in the connection info?
  1. Build and run the Honeybee server.
     ```shell
     cd server
     make run
     ```
  2. Copy `honeybee.key` file from `~/.cm-honeybee` or the path of 'CMHONEYBEE_ROOT' environment variable.
  3. See this commits to modify your source.
     * [cm-grasshopper: Decrypt passwords and private keys of Honeybee's connection info](https://github.com/cloud-barista/cm-grasshopper/commit/4c1c2c2224570d87296e24accca4b37e6ec7a81b)
     * [rsautil: Decrypt data with splited sizes](https://github.com/cloud-barista/cm-grasshopper/commit/08d6e90bd09e408e2ba1ddc026f07076e515f960)
     * [ssh: Apply honeybee changes](https://github.com/cloud-barista/cm-grasshopper/commit/0a412981c3005706544136e5f3b92d36e520bb5f)
