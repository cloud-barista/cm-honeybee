#!/bin/bash

# Repo
AGENT_REPO="https://raw.githubusercontent.com/cloud-barista/cm-honeybee/main/agent"

get_latest_release() {
  curl --silent "https://api.github.com/repos/$1/releases/latest" | # Get latest release from GitHub api
    grep '"tag_name":' |                                            # Get tag line
    sed -E 's/.*"([^"]+)".*/\1/'                                    # Pluck JSON value
}

is_root() {
    if [[ "$EUID" -ne 0 ]]
    then
        return 1
    else
        return 0
    fi
}

root_check() {
if ! is_root
then
    echo "Root 계정으로 실행해주세요."
    exit 1
fi
}
Initializer() {
    if [ -x "$(command -v curl)" ] && [ -x "$(command -v wget)" ] && [ -x "$(command -v iptables)" ]; then
        # echo "[Initializer] --PASS"
        echo ""

        sleep 1

    else
        NEEDED_DEPS=(curl wget iptables)
        # echo "패키지 설치 :" "${NEEDED_DEPS[@]}"

        if [ -x "$(command -v apt-get)" ]
        then
            sudo apt-get install "${NEEDED_DEPS[@]}" -y
        elif [ -x "$(command -v yum)" ]
        then
            sudo yum install "${NEEDED_DEPS[@]}" -y
        else
            # echo "패키지 매니저를 찾을 수 없어 설치에 실패하였습니다. 수동으로 다음 패키지 설치 :" "${NEEDED_DEPS[@]}"
            exit 1
        fi
    fi
}

Copy() {
    if [ -f "/usr/bin/cm-honeybee-agent" ]; then
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
    status=$(systemctl status cm-honeybee-agent | grep Active | awk '{print $3}')
    if [[ "$status" == "(running)" ]]; then
        # echo "[service start] --PASS"
        echo ""

        sleep 1
    else
        # 서비스 활성화
        systemctl daemon-reload
        systemctl enable cm-honeybee-agent
        systemctl start cm-honeybee-agent
    fi

    sleep 1
}

((
# root 체크
root_check

# 초기 설정
Initializer

# Agent 복사
Copy

# Agent 실행
Start
) 2>&1) | tee -a /tmp/honeybee-agent-install.log
