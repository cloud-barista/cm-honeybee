#!/bin/bash

AGENT_REPO="https://raw.githubusercontent.com/cloud-barista/cm-honeybee/main/agent"

get_latest_release() {
  curl --silent "https://api.github.com/repos/$1/releases/latest" | # Get latest release from GitHub api
    grep '"tag_name":' |                                            # Get tag line
    sed -E 's/.*"([^"]+)".*/\1/'                                    # Pluck JSON value
}

Initializer() {
    if [ -f /tmp/agentFirst ]; then
        # 첫 실행이 아닐 경우
        # echo "[Install] --PASS"
        echo ""

        sleep 1
    else
        # 첫 실행인 경우
        touch /tmp/agentFirst

        if [ -f /etc/debian_version ]; then
          apt-get install -y curl wget > /tmp/honeybee-agent-install.log 2>&1
        elif [ -f /etc/redhat-release ]; then
          yum install -y curl wget > /tmp/honeybee-agent-install-install.log 2>&1
        fi
    fi
}

Copy() {
    if [ -f /usr/bin/cm-honeybee-agent ]; then
        # echo "[Binary Copy] --PASS"
        echo ""

        sleep 1
    else
        LATEST_RELEASE=$(get_latest_release "cloud-barista/cm-honeybee")
        DOWNLOAD_URL=https://github.com/cloud-barista/cm-honeybee/releases/download/${LATEST_RELEASE}/cm-honeybee-agent

        wget --no-check-certificate --continue --quiet $DOWNLOAD_URL -P /usr/bin
        chmod a+x /usr/bin/cm-honeybee-agent

        mkdir -p /etc/cloud-migrator/cm-honeybee-agent/conf
        wget --no-check-certificate --continue --quiet ${AGENT_REPO}/conf/cm-honeybee-agent.yaml -P /etc/cloud-migrator/cm-honeybee-agent/conf
        wget --no-check-certificate --continue --quiet ${AGENT_REPO}/scripts/systemd/cm-honeybee-agent.service -P /etc/systemd/system
    fi
}

Start() {
    status=$(service cm-honeybee-agent status | grep Active | awk '{print $3}')
    if [[ "$status" == "(running)" ]]; then
        # echo "[service start] --PASS"
        echo ""

        sleep 1
    else
        systemctl daemon-reload
        systemctl enable --now cm-honeybee-agent
    fi

    sleep 1
}

Initializer
Copy
Start