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
  * Go: 1.24.0

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
        - port : Port of the agent's API.
- Configuration file example
  ```yaml
  cm-honeybee:
      listen:
          port: 8081
      agent:
          port: 8082
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
  "connection_info_status_count": {
    "count_connection_success": 0,
    "count_connection_failed": 0,
    "count_agent_success": 0,
    "count_agent_failed": 0,
    "connection_info_total": 0
  }
}
```
### 3. Register connection info
Register the connection information to the source group.
Agent will be installed automatically.
- Request
```shell
curl -X 'POST' \
 'http://127.0.0.1:8081/honeybee/source_group/b9e86d53-9fbe-4a96-9e06-627f77fdd6b7/connection_info' \
 -H 'accept: application/json' \
 -H 'Content-Type: application/json' \
 -d '{ "description": "NFS Server", "ip_address": "172.16.0.123", "name": "cm-nfs", "password": "some_pass", "private_key": "-----BEGIN RSA PRIVATE KEY-----\n******\n-----END RSA PRIVATE KEY-----", "ssh_port": 22, "user": "ubuntu" }'
```
- Reply
```json
{
  "id": "2f678139-e6e6-43e8-9722-33b834efc563",
  "name": "cm-nfs",
  "description": "NFS Server",
  "source_group_id": "b9e86d53-9fbe-4a96-9e06-627f77fdd6b7",
  "ip_address": "172.16.0.123",
  "ssh_port": "XXXXXXXX...=",
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

Check if CM-Honeybee agent is running

```bash
curl http://localhost:8081/honeybee/readyz

# Output if it's running successfully
# {"message":"CM-Honeybee Agent API server is ready"}
```

## Check out all APIs
* [Honeybee Agent APIs (Swagger Document)](https://cloud-barista.github.io/cb-tumblebug-api-web/?url=https://raw.githubusercontent.com/cloud-barista/cm-honeybee/main/agent/pkg/api/rest/docs/swagger.yaml)
* [Honeybee Server APIs (Swagger Document)](https://cloud-barista.github.io/cb-tumblebug-api-web/?url=https://raw.githubusercontent.com/cloud-barista/cm-honeybee/main/server/pkg/api/rest/docs/swagger.yaml)


## For Docker users
There are default private key and public key used for encrypt connection info's secret values (ssh port, user, password, private key) from the honeybee server.
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
