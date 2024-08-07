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
 "id": "b9e86d53-9fbe-4a96-9e06-627f77fdd6b7",
 "name": "test-group",
 "description": "test migration group"
}
```
### 4. Register connection info
Register the connection information to the source group.
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
  "ssh_port": 22,
  "user": "ubuntu",
  "password": "O6fiNHqV71q5cXbJ31Y7i5xefELacROcugMz8rdo42vbJVHsN3Geh+5iqQqYJlT+gFGY2DoH8EgftrI3jWFbofUIhEe0gJWQakIO+1T3mVNb458ZFg9agoqZucAf2JJlCQFw5Wddswd88KegFcE3nqTXalQX1rspV2v2M/rJ/d7DHVh7Ej2sMxn+7ZKSdtnk3tSthJ5Z6zAcLlaequ210UZHcwGk58ByP6A+2Ga08pxoqd++z+OTkXCWCLMRpd85LBo0VHc2qDLrWhkxZDv4OBqTeT3RpgCTX9PDyjNXt7/4srSBOb7Al9DNx6ITCme+rcBRUSCmeulECCBr9CZFQ==",
  "private_key": "CNTS7NvcwUj09/ZFL43GotzE68x/l6pesSRvp6/hv85ISDe1ynCxy/V8SxRIvzji2jPjcg2AwLEViPCi5vSFT5LTFQneFAXwtgJj9MdLQB4LBJVl8Bq/8MOfUsM/zltV98BX/XErzQZHrKipYmjchl1u90/Kka2zt6Ko7MugZqmmvpSy9ILOlxMPRTDdmLreW2toaFeAIfIT6NbrsYhLq+Je2FRqeET9tsabDmooQiMFIAo+t7J3vbvYuRQeEjdj66hlGzxrde/sCV8aA7hLsupiXOoJKxLTLfiha2oGOWtF9ofvEoQulX1f8M98zMl+VXFpYgx2SSxgpWFx0iTfhA==",
  "public_key": "",
  "status": "",
  "failed_message": ""
}
```

### 5. Save current source information.
Below example is saving infrastructure information of all connection in the source group.
```shell
curl -X 'POST' \
 'http://127.0.0.1:8081/honeybee/source_group/b9e86d53-9fbe-4a96-9e06-627f77fdd6b7/import/infra' \
 -H 'accept: application/json'
```

### 6. Get saved source information.
Below example is getting saved infrastructure information of all connection in the source group.
```shell
curl -X 'GET' \
 'http://127.0.0.1:8081/honeybee/source_group/b9e86d53-9fbe-4a96-9e06-627f77fdd6b7/infra' \
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
* [Honeybee Agent APIs (Swagger Document)](https://cloud-barista.github.io/cb-tumblebug-api-web/?url=https://raw.githubusercontent.com/cloud-barista/cm-honeybee/main/agent/pkg/api/rest/docs/swagger.yaml)
* [Honeybee Server APIs (Swagger Document)](https://cloud-barista.github.io/cb-tumblebug-api-web/?url=https://raw.githubusercontent.com/cloud-barista/cm-honeybee/main/server/pkg/api/rest/docs/swagger.yaml)

## For who develop modules with Honeybee

### About passwords and private keys
Those encrypted values are always changes with each request by RSA algorithm.

### How to decrypt the password and the private key in the connection info?
  1. Build and run the Honeybee server.
     ```shell
     cd server
     make run
     ```
  2. Copy `honeybee.key` file from `~/.cm-honeybee` or the path of 'CMHONEYBEE_ROOT' environment variable.
  3. See this commit to modify your source.
     * [cm-grasshopper: Decrypt passwords and private keys of Honeybee's connection info](https://github.com/cloud-barista/cm-grasshopper/commit/4c1c2c2224570d87296e24accca4b37e6ec7a81b)