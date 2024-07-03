# Collecting and Aggregating Information From Source Computing
This repository provides a features of collection and aggregation for all source computing information. This is a sub-system on Cloud-Barista platform and utilizes CM-Beetle to migrate a multi-cloud.

## Overview

Collecting and Aggregating Information From Source Computing framework (codename: cm-honeybee) is going to support:

* collect and aggregate information from source computing about intrastructure, software, data
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
  * Go: 1.21.6

## How to run

### 1. Build and run agent

1.1. Write the configuration file.
  - Configuration file name is 'cm-honeybee-agent.yaml'
  - The configuration file must be placed in one of the following directories.
    - .cm-honeybee-agent/conf directory under user's home directory
    - 'conf' directory where running the binary
    - 'conf' directory where placed in the path of 'CMHONEYBEE_AGENT_ROOT' environment variable
  - Configuration options
    - listen
      - port : Listen port of the agent's API.
  - Configuration file example
    ```yaml
    cm-honeybee-agent:
        listen:
            port: 8082
    ```

1.2. Build and run the agent binary
```shell
cd agent
make run
```

### 2. Build and run server

2.1. Write the configuration file.
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

2.2. Build and run the server binary
```shell
cd server
make run
```

### 3. Register source group
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
 "id": "group-01",
 "name": "test-group",
 "description": "test migration group"
}
```
### 4. Register connection info
Register the connection information to the source group.
- Request
```shell
curl -X 'POST' \
 'http://127.0.0.1:8081/honeybee/source_group/group-01/connection_info' \
 -H 'accept: application/json' \
 -H 'Content-Type: application/json' \
 -d '{ "description": "NFS Server", "ip_address": "172.16.0.123", "name": "cm-nfs", "password": "some_pass", "private_key": "-----BEGIN RSA PRIVATE KEY-----\n******\n-----END RSA PRIVATE KEY-----", "ssh_port": 22, "user": "ubuntu" }'
```
- Reply
```json
{
  "id": "connection-01",
  "name": "cm-nfs",
  "description": "NFS Server",
  "source_group_id": "group-01",
  "ip_address": "172.16.0.123",
  "ssh_port": 22,
  "user": "ubuntu",
  "password": "some_pass",
  "private_key": "-----BEGIN RSA PRIVATE KEY-----\n******\n-----END RSA PRIVATE KEY-----",
  "public_key": "",
  "status": "",
  "failed_message": ""
}
```

### 5. Save current source information.
Below example is saving infrastructure information of all connection in the source group.
```shell
curl -X 'POST' \
 'http://127.0.0.1:8081/honeybee/source_group/group-01/import/infra' \
 -H 'accept: application/json'
```

### 6. Get saved source information.
Below example is getting saved infrastructure information of all connection in the source group.
```shell
curl -X 'GET' \
 'http://127.0.0.1:8081/honeybee/source_group/group-01/infra' \
 -H 'accept: application/json'
```

## Health-check

### Agent

Check if CM-Honeybee agent is running

```bash
curl http://localhost:8082/honeybee-agent/readyz

# Output if it's running successfully
# {"message":"CM-Honeybee Agent API server is ready"}
```

### Server

Check if CM-Honeybee server is running

```bash
curl http://localhost:8081/honeybee/readyz

# Output if it's running successfully
# {"message":"CM-Honeybee API server is ready"}
```

## Check out all APIs
* [Honeybee APIs (Swagger Document)](https://cloud-barista.github.io/cb-tumblebug-api-web/?url=https://raw.githubusercontent.com/cloud-barista/cm-honeybee/main/server/pkg/api/rest/docs/swagger.yaml)