# Documentation

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
