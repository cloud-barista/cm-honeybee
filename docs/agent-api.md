# CM-Honeybee Agent API

**Agent**(`cm-honeybee-agent`)는 **각 소스 호스트에서 실행**되며 해당 호스트 한 대의 정보를 수집합니다.
상태를 저장하지 않는 읽기 전용 수집기로, 모든 호출은 요청 시점에 로컬 머신을 조사해 결과를 반환합니다.
Server가 이 엔드포인트들로부터 데이터를 가져갑니다(에이전트가 푸시하지 않음).

| 항목 | 값 |
|------|-----|
| 모듈명 | `HONEYBEE-AGENT` |
| Base path | `/honeybee-agent` |
| 리슨 주소 | `127.0.0.1` (루프백 전용) |
| 리슨 포트 | **기동할 때마다 커널이 고르는 빈 포트** (아래 "포트 확인" 참고) |
| Swagger UI | `http://127.0.0.1:<port>/honeybee-agent/api/index.html` |
| 인증 | 없음 |

> 아래 모든 경로는 base path 기준 상대 경로입니다. 전체 URL 예시:
> `http://127.0.0.1:<port>/honeybee-agent/infra`

## 포트 확인

에이전트는 고정 포트를 쓰지 않습니다. 설정의 `listen.port` 기본값이 `0`이고, 이는
**커널이 비어 있는 포트를 골라준다**는 뜻입니다. 소스 호스트에서 이미 쓰고 있는 포트와
충돌하지 않게 하기 위한 것이며, 재기동하면 포트 번호가 바뀝니다.

고른 포트는 에이전트가 파일로 남깁니다. 확인하는 방법은 셋입니다.

```bash
# 1. 포트 파일 (권장). 일반 계정으로 읽을 수 있습니다.
cat /etc/cloud-migrator/cm-honeybee-agent/port

# 2. 리슨 중인 소켓에서 직접 (root 필요)
sudo ss -lntp | grep cm-honeybee-age

# 3. 기동 로그
sudo journalctl -u cm-honeybee-agent | grep "http server started"
```

포트 파일 경로는 `CMHONEYBEE_AGENT_ROOT` 아래입니다. systemd 유닛이 이 값을
`/etc/cloud-migrator/cm-honeybee-agent/`로 지정하므로 기본 설치에서는 위 경로가 맞습니다.

이 문서의 예시는 아래처럼 포트를 변수에 담아 두었다고 가정합니다.

```bash
PORT=$(cat /etc/cloud-migrator/cm-honeybee-agent/port)
```

**포트를 고정해야 한다면** `conf/cm-honeybee-agent.yaml`의 `listen.port`에 번호를 적고
에이전트를 재기동하십시오. 이 경우에도 포트 파일은 그대로 갱신됩니다.

**Server는 이 포트를 알 필요가 없습니다.** cm-honeybee는 소스 호스트에 SSH로 접속한 뒤
그 안에서 `curl http://localhost:<port>`를 실행하며, 포트는 요청할 때마다 위 포트 파일을
읽어서 알아냅니다. 포트 파일이 없으면(구버전 에이전트) `cm-honeybee.agent.port` 설정값으로
넘어갑니다.

**루프백 전용입니다.** 에이전트는 `127.0.0.1`에만 바인드하므로 다른 호스트에서 직접
접근할 수 없습니다. 원격에서 Swagger UI를 봐야 하면 SSH 포트 포워딩을 쓰십시오.

```bash
ssh -L 8082:127.0.0.1:$PORT <user>@<source-host>
# 이후 브라우저에서 http://127.0.0.1:8082/honeybee-agent/api/index.html
```

## 엔드포인트 요약

| 메서드 | 경로 | 설명 |
|--------|------|------|
| GET | `/readyz` | 준비 상태(readiness) 확인. |
| GET | `/infra` | 호스트의 인프라 수집 (CPU, 메모리, 디스크, 네트워크, 라우팅, 방화벽, OS). |
| GET | `/software` | 설치된 소프트웨어 수집 (패키지, 바이너리, 컨테이너). |
| GET | `/kubernetes` | 쿠버네티스 클러스터/노드 정보 수집 (호스트가 접근 가능한 컨트롤 플레인일 때만). |
| GET | `/helm` | 설치된 Helm 릴리스 수집. |
| GET | `/data` | 데이터 마이그레이션 관련 정보 수집 (필수 필드만). |

---

## Admin

### `GET /readyz` — 준비 상태 확인

에이전트가 기동되어 요청을 처리할 수 있으면 `200 OK`를 반환합니다.

```bash
curl http://127.0.0.1:$PORT/honeybee-agent/readyz
```

---

## Infra

### `GET /infra` — 통합 인프라 정보 조회

호스트의 전체 인프라 구성을 수집합니다: 컴퓨트(CPU/메모리), 루트 + 데이터 디스크, 네트워크 인터페이스,
라우팅 테이블, 방화벽 규칙, OS 메타데이터, GPU.

GPU는 `nvidia`(nvidia-smi XML), `amd`(rocm-smi JSON), 커널이 붙인 `drm` 세 갈래로 나뉩니다.
세 갈래를 각각 독립으로 수집하므로 실패한 갈래만 빈 배열이 되고 이유가 `errors[]`에 남습니다 -
**`errors`가 비어 있지 않다고 수집 전체가 실패한 것은 아닙니다.** 예를 들어 AMD GPU가 없는
호스트는 `amd`가 빈 배열이고 `errors`에 `"AMD: rocm-smi command is not available"` 한 줄이
남지만 `nvidia` 수집은 정상입니다.

**`nvidia` 배열과 `drm` 배열은 역할이 다릅니다.** `nvidia`는 nvidia-smi가 보고하는 GPU이고,
`drm`은 커널 DRM 서브시스템에 등록된 카드 목록입니다. `nvidia_drm` 모듈이 로드되지 않은
환경(일부 클라우드 이미지)에서는 GPU가 `nvidia`에는 잡히지만 `drm`에는 안 잡히고, 대신
하이퍼바이저 프레임버퍼(`simpledrm` 등)만 나옵니다. **GPU 유무는 `nvidia`/`amd`로 판단하세요.**

`device_attribute.virtualization_mode`로 물리 장비와 가상화 환경을 구분할 수 있습니다.
실측 예: 물리 GPU는 `"None"`, GPU 패스스루된 클라우드 VM은 `"Pass-Through"`.

`nvidia_smi_schema`에는 `nvidia-smi -q -x` 출력의 DOCTYPE에서 읽어낸 XML 스키마 버전이 담깁니다.

```
<!DOCTYPE nvidia_smi_log SYSTEM "nvsmi_device_v13.dtd">   →   "v13"
```

전용 파서가 없는 버전은 가장 가까운 파서로 읽고 **감지한 버전과 실제로 쓴 파서를 함께** 표기합니다.
대체해서 읽은 것을 이해한 것처럼 보고하지 않기 위함입니다.

| 감지된 버전 | 쓰는 파서 | 표기 |
|-------------|-----------|------|
| v10, v11 | v11 | `v10`, `v11` |
| v12 | v12 | `v12` |
| v13 | v13 | `v13` |
| v13보다 위 (예: v14) | v13 | `v14 (read as v13)` |
| v11보다 아래 (예: v9) | v11 | `v9 (read as v11)` |
| DOCTYPE 없음 (구형 드라이버) | v11 | `v11` |
| **nvidia-smi가 없거나 실행 실패** | 없음 | **키 자체가 응답에서 빠짐** |

마지막 줄이 중요합니다. `omitempty`라서 빈 문자열(`""`)로 오지 않고 **키가 사라집니다.**
그 경우 이유는 `errors[]`에서 확인합니다 (예: `"NVIDIA: nvidia-smi command is not available"`).

하위 버전을 v13 파서로 읽으면 안 되는 이유가 있습니다. v13이 `power_readings`를
`gpu_power_readings`로 바꿨기 때문에, v9 문서를 v13 파서로 읽으면 `power_draw`·`power_limit`·
`clocks_event_reasons`가 **에러 없이 사라집니다.**

```bash
curl http://127.0.0.1:$PORT/honeybee-agent/infra
```

---

## Software

### `GET /software` — 소프트웨어 정보 조회

설치된 소프트웨어를 수집합니다: OS 패키지, 독립 실행 바이너리, 실행 중인 컨테이너.

| 쿼리 파라미터 | 타입 | 기본값 | 설명 |
|---------------|------|--------|------|
| `show_default_packages` | bool | `false` | OS 기본/베이스 패키지를 결과에 포함. 기본적으로는 필터링되어 "의미 있는"(사용자 설치) 소프트웨어만 반환됩니다. |

```bash
# 사용자 설치 소프트웨어만
curl "http://127.0.0.1:$PORT/honeybee-agent/software"

# OS 기본/베이스 패키지까지 포함
curl "http://127.0.0.1:$PORT/honeybee-agent/software?show_default_packages=true"
```

---

## Kubernetes

### `GET /kubernetes` — 쿠버네티스 정보 조회

쿠버네티스 **클러스터** 메타데이터(이름, 버전, CNI 플러그인, Pod/Service CIDR, NodePort 범위)와
**노드** 정보(`control-plane`/`worker` 등 역할, 노드 스펙, machine ID)를 수집합니다.

노드 스펙에는 CPU·메모리·임시 스토리지와 함께 `gpu[]`가 들어갑니다. 쿠버네티스에는 GPU를 나타내는
기본 리소스가 없어서, 클러스터가 GPU를 가졌다고 말하는 곳은 `nvidia.com/gpu` 같은 **확장 리소스**뿐입니다.
`capacity`/`allocatable`은 코어가 아니라 **장치 개수**입니다. `vendor`·`product`·`driver_version`·
`memory`·`mig_capable`·`mig_strategy`는 벤더의 feature discovery가 노드에 붙인 레이블에서 가져오므로,
그것 없이 device plugin만 도는 클러스터에서는 비어 있습니다(추측해서 채우지 않습니다).

> **중요:** 쿠버네티스(및 Helm) 수집은 호스트가 *접근 가능한 쿠버네티스 컨트롤 플레인*일 때만
> 동작합니다. 컨트롤 플레인이 아니면 에이전트가 수집을 건너뜁니다(커밋 `66c7305`). kubeconfig 경로는
> `KUBECONFIG` 환경 변수가 설정되어 있으면 그 값을, 없으면 기본 위치를 사용합니다(커밋 `1d73b05`).

```bash
curl http://127.0.0.1:$PORT/honeybee-agent/kubernetes
```

Server는 이 출력을 정제 소스 모델로 매핑합니다 —
[Server API → 쿠버네티스 소스 모델](./server-api.md#쿠버네티스-소스-모델) 참고.

---

## Helm

### `GET /helm` — Helm 정보 조회

호스트가 속한 클러스터에 설치된 Helm 릴리스 목록을 수집합니다. 쿠버네티스 엔드포인트와 마찬가지로
접근 가능한 컨트롤 플레인 호스트에서만 의미가 있습니다.

```bash
curl http://127.0.0.1:$PORT/honeybee-agent/helm
```

---

## Data

### `GET /data` — 데이터 마이그레이션 정보 조회

데이터 마이그레이션에 필요한 필드로 한정하여 관련 정보를 수집합니다.

```bash
curl http://127.0.0.1:$PORT/honeybee-agent/data
```
