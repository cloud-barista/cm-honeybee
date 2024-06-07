#!/bin/bash

pid_file="/tmp/milkyway.pid"

# 각 결과를 JSON 형식으로 변환하는 함수
convert_to_json() {
    status_code=${1-"200"}
    type="$2"
    desc="$3"
    elapsed="$4"
    result="$5"
    specid="$6"
    unit="$7"

    # {
    #     "status_code": "%s",
    #     "type": "%s",
    #     "data": {
    #         "desc": "%s",
    #         "elapsed": "%s",
    #         "result": "%s",
    #         "specid": "%s",
    #         "unit": "%s"
    #     }
    # }
    # JSON 포맷으로 변환하여 출력
    printf '{"status_code": "%s","type": "%s","data": {"desc": "%s","elapsed": "%s","result": "%s","specid": "%s","unit": "%s"}}\n' \
        "$status_code" "$type" "$desc" "$elapsed" "$result" "$specid" "$unit"
}

# 각 결과를 가져오고 JSON 형식으로 변환하여 출력하는 함수
Initializer() {    
    # milkway 가 있을 경우
    if [ -f /tmp/milkyway ]; then
        # echo "[Download] --PASS"
        echo ""
    else
        # milkyway 가 없을 경우
        echo "[download milkyway binary]"
        wget https://github.com/ish-hcc/cb-milkyway/raw/master/src/milkyway -O /tmp/milkyway
        chmod a+x /tmp/milkyway
    fi

    Process_Start

    if [ -f /tmp/benchFirst ]; then
        # 첫 실행이 아닐 경우
        # echo "[Install] --PASS"
        echo ""
    else
        # 첫 실행인 경우
        touch /tmp/benchFirst

        echo "[Call Install]"
        curl -sX GET http://localhost:1324/milkyway/install | json_pp || return 1
        echo "#-----------------------------"
    fi
}

Process_Start() {
    # 현재 실행 중인 경우
    if [ -f /tmp/milkyway.pid ]; then
        # echo "[restart milkyway]"
        
        # PID 파일에서 PID 값 읽기
        pid=$(cat "$pid_file")

        # PID를 사용하여 프로세스 종료
        if [ -n "$pid" ]; then
            # echo "Milkyway 프로세스를 종료합니다. PID: $pid"
            kill "$pid"
            rm "$pid_file"
        else
            echo "PID 파일에서 PID를 읽어오는 데 실패했습니다."
        fi

        nohup /tmp/milkyway > /dev/null 2>&1 & echo $! > /tmp/milkyway.pid
    else
        # 현재 실행 중이 아닌 경우
        # echo "[start milkyway]"
        nohup /tmp/milkyway > /dev/null 2>&1 & echo $! > /tmp/milkyway.pid
    fi
}

Process_Kill() {    
    # PID 파일이 있는 경우
    if [ -f "$pid_file" ]; then
        # PID 파일에서 PID 값 읽기
        pid=$(cat "$pid_file")

        # PID를 사용하여 프로세스 종료
        if [ -n "$pid" ]; then
            # echo "Milkyway 프로세스를 종료합니다. PID: $pid"
            kill "$pid"
            rm "$pid_file"
        else
            echo "PID 파일에서 PID를 읽어오는 데 실패했습니다."
        fi
    else
        echo "PID 파일이 존재하지 않습니다."
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
    echo "인자를 입력해주세요. 예: milkyway.sh cpus"
    convert_to_json 400 "" "" "인자를 입력해주세요. 예: milkyway.sh cpus" "" ""
    exit 1
fi

# curl -sX GET http://localhost:1324/milkyway/rtt -H 'Content-Type: application/json' -d '{ "host": "localhost"}' |json_pp || return 1
# curl -sX GET http://localhost:1324/milkyway/clean | json_pp || return 1

Initializer
Collecting_Data $1
Process_Kill