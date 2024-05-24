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
  * Go: 1.22.3

## How to run

### 1. Build and run agent

1.1. Write the configuration file.
  - Configuration file name is 'cm-honeybee-agent.yaml'
  - The configuration file must be placed in one of the following directories.
    - .cm-honeybee-agent/conf directory under user's home directory
    - 'conf' directory where running the binary
    - 'conf' directory where placed in the path of 'CMHONEYBEE_AGENT_ROOT' environment variable
  - Configuration options
    - server
      - address : Specify collection server's address ({IP or Domain}:{Port})
      - timeout : HTTP timeout value as seconds.
    - listen
      - port : Listen port of the agent's API.
  - Configuration file example
    ```yaml
    cm-honeybee-agent:
        server:
            address: 172.16.0.10:8081
            timeout: 10
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
        - port : Listen port of the agent's API.
- Configuration file example
  ```yaml
  cm-honeybee:
      listen:
          port: 8081
  ```

2.2. Build and run the server binary
```shell
cd server
make run
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
