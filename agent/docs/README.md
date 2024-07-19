# Agent 서비스 구동 방법

## 1. Honeybee 소스 클론

```shell
git clone git@github.com:cloud-barista/cm-honeybee.git
```

## 2. Agent 소스 빌드

```shell
cd cm-honeybee/agent
make
```

## 3. Agent 바이너리 복사

```shell
sudo cp cmd/cm-honeybee-agent/cm-honeybee-agent /usr/bin
```

## 4. Agent 데이터 폴더 생성 및 설정파일 복사

```shell
sudo mkdir -p /etc/cloud-migrator/cm-honeybee-agent/conf
sudo cp conf/cm-honeybee-agent.yaml /etc/cloud-migrator/cm-honeybee-agent/conf/
```

## 5. 서비스 스크립트 복사

```shell
sudo cp scripts/systemd/cm-honeybee-agent.service /etc/systemd/system/
```

## 6. 서비스 등록 및 활성화

```shell
sudo systemctl daemon-reload
sudo systemctl enable cm-honeybee-agent
```

## 7. Agent 서비스 시작

```shell
sudo systemctl start cm-honeybee-agent
```

## 8. Agent 서비스 상태 확인

```shell
sudo systemctl status cm-honeybee-agent
```

### 9. Agent 접속
http://127.0.0.1:8082
