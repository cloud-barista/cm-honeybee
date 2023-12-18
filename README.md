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
  * Ubuntu 23.10, Ubuntu 22.04, Ubuntu 18.04, Rocky Linux 9, Windows 11
* Language:
  * Go: 1.21.5

## How to run

1. Build the binary
  - Run on Linux.
    ```shell
    make
    ```
  - Run on Linux for build Windows binary or run on Windows where make command is available.
    ```shell
    make windows
    ```

2. Write the configuration file.
  - Configuration file name is 'cm-honeybee.yaml'
  - The configuration file must be placed in one of the following directories.
    - .cm-honeybee/conf directory under user's home directory
    - 'conf' directory where running the binary
    - 'conf' directory where placed in the path of 'CMHONEYBEE_ROOT' environment variable
  - Configuration options
    - server (Need to implementation.)
      - address : Specify collection server's address ({IP or Domain}:{Port})
      - timeout : HTTP timeout value as seconds.
    - listen
      - port : Listen port of the API.
  - Configuration file example
    ```yaml
    cm-honeybee:
        server:
            address: 172.16.0.10:8081
            timeout: 10
        listen:
            port: 8082
    ```

3. Run with privileges
  - Linux
    ```shell
    sudo ./cm-honeybee
    ```
  - Windows
    - Run cm-honeybee.exe
    - Click Yes when UAC window is appears.

#### Download source code

Clone CM-Honeybee repository

```bash
git clone https://github.com/cloud-barista/cm-honeybee.git ${HOME}/cm-honeybee
```

#### Build CM-Honeybee

Build CM-Honeybee source code

```bash
cd ${HOME}/cm-honeybee
make build
```

(Optional) Update Swagger API document
```bash
cd ${HOME}/cm-honeybee
make swag
```

Access to Swagger UI
(Default link) http://localhost:8056/beetle/swagger/index.html

#### Run CM-Honeybee binary

Run CM-Honeybee server

```bash
cd ${HOME}/cm-honeybee
make build
./cm-honeybee
```

#### Health-check CM-Honeybee

Check if CM-Honeybee is running

```bash
curl http://localhost:8056/honeybee/health

# Output if it's running successfully
# {"message":"CM-Honeybee API server is running"}
```