#!/bin/bash

pid_file="/tmp/milkyway.pid"

Is_root() {
    if [[ "$EUID" -ne 0 ]]
    then
        return 1
    else
        return 0
    fi
}

Root_check() {
    if ! Is_root
    then
        echo "Root 계정으로 실행해주세요."
        exit 1
    fi
}

# 각 결과를 JSON 형식으로 변환하는 함수
convert_to_json() {
    status_code=${1-"200"}
    type="$2"
    desc="$3"
    elapsed="$4"
    result="$5"
    specid="$6"
    unit="$7"

    # JSON 포맷으로 변환하여 출력
    printf '{"status_code": "%s","type": "%s","data": {"desc": "%s","elapsed": "%s","result": "%s","specid": "%s","unit": "%s"}}\n' \
        "$status_code" "$type" "$desc" "$elapsed" "$result" "$specid" "$unit"
}

# 기본 패키지 설치 함수
Initializer() {
    if [ -x "$(command -v curl)" ] && [ -x "$(command -v wget)" ] && [ -x "$(command -v jq)" ] && [ -x "$(command -v sysbench)" ] && [ -x "$(command -v ping)" ]; then
        sleep 1
    else
        NEEDED_DEPS=(curl wget jq sysbench)
        if [ -x "$(command -v apt-get)" ]
        then
            sudo apt-get install "${NEEDED_DEPS[@]}" -y > /tmp/honeybee-agent-install.log 2>&1
        elif [ -x "$(command -v yum)" ]
        then
            sudo yum install "${NEEDED_DEPS[@]}" -y > /tmp/honeybee-agent-install.log 2>&1
        else
            exit 1
        fi

        if [ -x "$(command -v ping)" ]
        then
            if [ -x "$(command -v apt-get)" ]
            then
                sudo apt-get install "iputils-ping" -y > /tmp/honeybee-agent-install.log 2>&1
            elif [ -x "$(command -v yum)" ]
            then
                sudo yum install "iputils" -y > /tmp/honeybee-agent-install.log 2>&1
            fi
        else
            exit 1
        fi
    fi

    chmod a+x /tmp/milkyway
    Process_Start
}

Process_Start() {
    # 현재 실행 중인 경우
    if [ -f /tmp/milkyway.pid ]; then
        # PID 파일에서 PID 값 읽기
        pid=$(cat "$pid_file")

        # PID를 사용하여 프로세스 종료
        if ps -p $pid > /dev/null 2>&1; then
            kill -9 "$pid"
            rm "$pid_file"
        fi

        nohup /tmp/milkyway > /dev/null 2>&1 &
        echo $! > /tmp/milkyway.pid

        sleep 1
    else
        # 현재 실행 중이 아닌 경우
        nohup /tmp/milkyway > /dev/null 2>&1 &
        echo $! > /tmp/milkyway.pid

        sleep 1

        # milkyway 첫 실행 시, init 실행
        curl -X GET http://localhost:1324/milkyway/init > /tmp/milkyway-install.log 2>&1
    fi
}

Process_Kill() {
    # 현재 실행 중인 경우
    if [ -f /tmp/milkyway.pid ]; then
        # PID 파일에서 PID 값 읽기
        pid=$(cat "$pid_file")

        # PID를 사용하여 프로세스 종료
        if ps -p $pid > /dev/null 2>&1; then
            kill -9 "$pid" > /dev/null 2>&1
            rm "$pid_file"
        fi
    fi
}

Benchmarking_Kill() {
    # 관련 프로세스의 PID를 추출
    pids=$(ps -ef | grep "/bin/bash /tmp/milkyway.sh" | grep -v grep | awk '{print $2}')

    # PID가 존재하는지 확인하고 종료
    if [ -n "$pids" ]; then
        for pid in $pids; do
            kill -9 $pid > /dev/null 2>&1
        done
    fi
}

Collecting_Data() {
    # 입력된 인자에 따라 함수 실행
    case "$1" in
        cpus)
            get_and_convert_result "cpus"
            ;;
        cpum)
            get_and_convert_result "cpum"
            ;;
        memR)
            get_and_convert_result "memR"
            ;;
        memW)
            get_and_convert_result "memW"
            ;;
        fioR)
            get_and_convert_result "fioR"
            ;;
        fioW)
            get_and_convert_result "fioW"
            ;;
        dbR)
            get_and_convert_result "dbR"
            ;;
        dbW)
            get_and_convert_result "dbW"
            ;;
        *)
            echo "지원하지 않는 인자입니다."
            exit 1
            ;;
    esac
}

Clean() {
    curl -sX GET "http://localhost:1324/milkyway/clean"
}

# 각 결과를 가져오고 JSON 형식으로 변환하여 출력하는 함수
get_and_convert_result() {
    type="$1"
    endpoint="http://localhost:1324/milkyway/$type"

    # curl을 사용하여 결과 가져오기
    result=$(curl -sX GET "$endpoint" | json_pp)

    # curl 결과가 없을 경우 에러 메시지 출력 후 종료
    if [ -z "$result" ]; then
        echo "Error in executing the benchmark for $type" >&2
        return 1
    fi

    # JSON으로 변환하여 출력
    convert_to_json "200" "$type" \
        "$(echo "$result" | jq -r '.desc')" \
        "$(echo "$result" | jq -r '.elapsed')" \
        "$(echo "$result" | jq -r '.result')" \
        "$(echo "$result" | jq -r '.specid')" \
        "$(echo "$result" | jq -r '.unit')"
}

if [ -z "$1" ]; then
    echo "인자를 입력해주세요. 예: milkyway.sh --run cpus"
    exit 1
fi

while [[ $# -gt 0 ]]; do
    case $1 in
        --run)
            Root_check
            Initializer
            Collecting_Data $2
            Clean
            Process_Kill
            ;;
        --stop)
            Root_check
            Process_Kill
            Benchmarking_Kill
            ;;
    esac
    shift
done