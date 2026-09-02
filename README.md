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
  * Go: 1.26.2

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
    - spider
        - endpoint : cb-spider REST endpoint, used by CSP-type source groups.
        - username : Basic auth user for cb-spider (default: `default`).
        - password : Basic auth password for cb-spider (default: `default`).
- Configuration file example
  ```yaml
  cm-honeybee:
      listen:
          port: 8081
      agent:
          # Fallback only; see the option description above.
          port: 8082
      spider:
          endpoint: http://localhost:1024/spider
          username: default
          password: default
  ```

1.2. Build and run the server binary
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
- `csp` - collects from cloud sources (VM / Kubernetes / Object Storage) through cb-spider.

`onprem`, `ssh` and an omitted `type` are all treated as on-premise, so payloads
written against earlier versions keep working unchanged. An omitted `type` is
still stored and returned as `ssh`; send `onprem` explicitly to store that value.

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
A CSP group represents one cb-spider connection (provider + region + credential).
Resources under the group (VMs, Kubernetes clusters, object-storage buckets) are
registered as individual `connection_info` entries.

Discover the credential keys required by the target CSP first:
```shell
curl 'http://127.0.0.1:8081/honeybee/csp'              # supported CSP names, lowercase
curl 'http://127.0.0.1:8081/honeybee/csp/aws'          # case-insensitive
```
The response includes `credential_keys`, queried live from cb-spider. Their
spelling differs per CSP: AWS uses lowercase snake_case
(`aws_access_key_id`, `aws_secret_access_key`) while Azure uses camelCase
(`clientId`, `clientSecret`, `tenantId`, `subscriptionId`).

It also includes `region_keys`, which is the shape of a region for that CSP
(e.g. `["Region", "Zone"]`) rather than a list of regions, and `default_region`.
The actual regions need a stored credential - get them from
`GET /source_group/{sgID}/region` after the group is created.

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
honeybee stores the credential in OpenBao and keeps it there. cb-spider never
holds it: for each call honeybee registers a credential, region and connection
config under a per-call unique name, uses it, then tears all three down in
reverse order. No spider connection name is kept on the source group.
### 3. Register connection info
Register the connection information to the source group. The body shape
depends on the group's `type`.

#### 3.1 On-premise SSH connection info
`resource_type` defaults to `vm` and may be omitted. Agent will be installed automatically.
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
Do not call `POST .../import/*` on such a connection - those endpoints collect
through the in-guest agent over SSH. The stored kubeconfig is consumed by the
downstream migration modules.

#### 3.3 CSP connection info
For CSP groups, each `connection_info` points at one cloud resource by id.
You can list available resources first via the discovery endpoint:
```shell
curl 'http://127.0.0.1:8081/honeybee/source_group/{sgID}/discover?resource_type=vm'
curl 'http://127.0.0.1:8081/honeybee/source_group/{sgID}/discover?resource_type=k8s'
curl 'http://127.0.0.1:8081/honeybee/source_group/{sgID}/discover?resource_type=object_storage'
```
Then register the picked resource:
```shell
curl -X 'POST' \
 'http://127.0.0.1:8081/honeybee/source_group/{sgID}/connection_info' \
 -H 'Content-Type: application/json' \
 -d '{ "name": "vm-app01", "resource_type": "vm", "resource_id": "i-0abc..." }'
```
`resource_type` is one of `vm`, `k8s`, or `object_storage`.

`PUT .../refresh` only re-checks the connection status; it stores nothing.
Collection is done by `POST .../import/infra`, which queries cb-spider and
writes to a table chosen by `resource_type`:

| `resource_type` | Stored in | Read back with |
|-----------------|-----------|----------------|
| `vm` | `SavedInfraInfo.csp_data` | `GET .../infra` (the `csp` section) |
| `k8s` | `SavedKubernetesInfo` | `GET .../kubernetes` |
| `object_storage` | `SavedDataInfo` | `GET .../data` |

Note that `import/kubernetes` and `import/data` are the agent-over-SSH paths and
do not apply to CSP resources - use `import/infra` for all three.

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

### 6. Get refined, saved source information.
Below example is getting refined, saved infrastructure information of all connection in the source group.
```shell
curl -X 'GET' \
 'http://127.0.0.1:8081/honeybee/source_group/b9e86d53-9fbe-4a96-9e06-627f77fdd6b7/infra/refined' \
 -H 'accept: application/json'
```

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
