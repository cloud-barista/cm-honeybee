#!/bin/bash

BUSYBOX_PATH="/tmp/busybox"

# Repo
AGENT_REPO="https://raw.githubusercontent.com/cloud-barista/cm-honeybee/main/agent"

get_latest_release() {
    curl --silent "https://api.github.com/repos/$1/releases/latest" | # Get latest release from GitHub api
    grep '"tag_name":' |                                            # Get tag line
    sed -E 's/.*"([^"]+)".*/\1/'                                    # Pluck JSON value
}

check_new_version() {
    LATEST_RELEASE=$(get_latest_release "cloud-barista/cm-honeybee")
    CURRENT_VERSION=$(cm-honeybee-agent version 2>&1)

    if [ "$LATEST_RELEASE" = "$CURRENT_VERSION" ]; then
        echo 0
    else
        echo 1
    fi
}

is_root() {
    [[ "$EUID" -ne 0 ]] && return 1 || return 0
}

root_check() {
    if ! is_root; then
        echo "Please run as root!"
        exit 1
    fi
}

Copy() {
    RESULT=$(check_new_version)
    if [ "$RESULT" = "0" ]; then
        echo "Latest version already installed."
    elif [ "$RESULT" = "1" ] || [ ! -f "/usr/bin/cm-honeybee-agent" ]; then
        systemctl stop cm-honeybee-agent > /dev/null 2>&1
        rm -rf /usr/bin/cm-honeybee-agent
        LATEST_RELEASE=$(get_latest_release "cloud-barista/cm-honeybee")
        DOWNLOAD_URL=https://github.com/cloud-barista/cm-honeybee/releases/download/${LATEST_RELEASE}/cm-honeybee-agent
        $BUSYBOX_PATH wget --no-check-certificate --quiet "$DOWNLOAD_URL" -P /usr/bin
        chmod a+x /usr/bin/cm-honeybee-agent
    fi

    if [ ! -f "/etc/cloud-migrator/cm-honeybee-agent/conf/cm-honeybee-agent.yaml" ]; then
        mkdir -p /etc/cloud-migrator/cm-honeybee-agent/conf
        $BUSYBOX_PATH wget --no-check-certificate --quiet "${AGENT_REPO}/conf/cm-honeybee-agent.yaml" -P /etc/cloud-migrator/cm-honeybee-agent/conf
    fi

    if [ ! -f "/lib/systemd/system/cm-honeybee-agent.service" ]; then
        $BUSYBOX_PATH wget --no-check-certificate --quiet "${AGENT_REPO}/service_file/systemd/cm-honeybee-agent.service" -P /lib/systemd/system
    fi
}

Start() {
    local status
    status=$(systemctl is-active cm-honeybee-agent)

    if [[ "$status" != "active" ]]; then
        systemctl daemon-reload
        systemctl enable cm-honeybee-agent
        systemctl start cm-honeybee-agent
    fi
}

# Main Script
((
    root_check
    Copy
    Start
) 2>&1) | tee -a /tmp/honeybee-agent-install.log
