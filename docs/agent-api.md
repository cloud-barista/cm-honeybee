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
실측한 값은 세 가지입니다.

| 값 | 어떤 환경 | 같이 오는 것 |
|----|-----------|--------------|
| `"None"` | 물리 장비 | |
| `"Pass-Through"` | GPU 패스스루된 클라우드 VM | |
| `"Host VGPU"` | vGPU를 나눠 주는 호스트 | `host_vgpu_mode`(예: `"SR-IOV"`) |

vGPU 호스트에는 CUDA 런타임이 없어서 nvidia-smi가 CUDA 버전을 `Not Found`로 답합니다.
그것은 값이 아니라 **읽히지 않았다는 뜻**이라 `N/A`와 같이 취급되어 `cuda_version` 키가 빠집니다.

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

#### ECC를 켜면 총 메모리가 표기 용량보다 작게 나옵니다

`performance.fb_memory_total`은 드라이버가 보고하는 값을 그대로 옮긴 것이고, **카드에 적힌 용량이
아닙니다.** ECC가 켜진 카드는 패리티에 쓰는 만큼이 빠진 채 보고됩니다.

L40S(공칭 48GB, ECC on) 실측입니다.

```
Total    : 46068 MiB      <- 공칭 49152 MiB 보다 3084 MiB (6.27%) 작다
Reserved :  1104 MiB
Used     :  7233 MiB
Free     : 37733 MiB
```

두 가지를 주의하세요.

- **`reserved`는 `total` 안에 들어 있습니다.** `used + free + reserved`가 `total`과 맞습니다.
  `total` 옆에 따로 있는 양이 아닙니다.
- **`total + reserved`로도 공칭 용량이 복원되지 않습니다.** 위 예에서 47172 MiB로, 아직
  1980 MiB 모자랍니다. ECC가 가져간 몫과 `reserved`는 서로 다른 것입니다.

ECC가 얼마나 가져갔는지는 **이 응답만으로는 알 수 없습니다.** 공칭 용량을 담은 필드가 없어서,
차이를 구하려면 카드 사양을 밖에서 알아야 합니다. `ecc.mode`로 켜졌는지만 알 수 있습니다.

#### MIG

MIG가 켜진 카드는 `device_attribute.mig_mode`가 `"Enabled"`이고 `mig_devices[]`가 채워집니다.

```json
"mig_mode": "Enabled",
"mig_devices": [
  { "index": "0", "gpu_instance_id": "1", "compute_instance_id": "0",
    "fb_memory_total": 47744, "fb_memory_used": 47508, "fb_memory_free": 237 }
]
```

- **`mig_devices[].uuid`는 `-q -x`에 없으면 `nvidia-smi -L`로 채웁니다.** 스키마에는 `<uuid>`
  요소가 있는데 드라이버가 그것을 아예 내보내지 않는 구성이 있습니다(MIG-backed vGPU 호스트,
  드라이버 595.71.03에서 확인). 같은 호스트에서 `-L`은 같은 시점에 UUID를 내주므로, MIG 인스턴스가
  있는데 UUID가 빈 경우에만 `-L`을 한 번 더 불러 보충합니다. `-q -x`가 이미 준 값은 덮어쓰지
  않고, MIG가 없는 호스트에서는 `-L`을 부르지 않습니다. 보충에 실패해도 수집은 그대로 성공하고
  그 필드만 빕니다.
- MIG 카드는 ECC로 거의 깎이지 않습니다. 실측에서 공칭 96GB 카드가 97887 MiB를 보고해
  417 MiB 차이였습니다 (위 L40S의 3084 MiB와 대비됩니다).

```bash
curl http://127.0.0.1:$PORT/honeybee-agent/infra
```

#### 응답 예시 - `gpu` 절

물리 서버(GeForce GTX 1660, 드라이버 610.57.04 / CUDA 13.3)에서 받은 실제 응답입니다.
프로세스 이름의 긴 인자만 줄였고 나머지는 그대로입니다.

```json
{
  "nvidia": [
    {
      "device_attribute": {
        "gpu_uuid": "GPU-05548171-05c7-229a-e00e-59703ed40eb0",
        "driver_version": "610.57.04",
        "cuda_version": "13.3",
        "product_name": "NVIDIA GeForce GTX 1660",
        "product_brand": "GeForce",
        "product_architecture": "Turing",
        "nvml_version": "610.57",
        "index": 0,
        "minor_number": "0",
        "pci_bus_id": "00000000:01:00.0",
        "vbios_version": "90.16.25.00.C3",
        "compute_mode": "Default",
        "persistence_mode": "Disabled",
        "virtualization_mode": "None"
      },
      "performance": {
        "gpu_usage": 3,
        "memory_usage": 7,
        "encoder_usage": 0,
        "decoder_usage": 0,
        "fb_memory_used": 1302,
        "fb_memory_total": 6144,
        "fb_memory_free": 4446,
        "fb_memory_reserved": 397,
        "fb_memory_usage": 21,
        "bar1_memory_used": 21,
        "bar1_memory_total": 256,
        "bar1_memory_free": 235,
        "bar1_memory_usage": 8,
        "performance_state": "P8",
        "fan_speed": 40,
        "temperature_gpu": 39,
        "power_draw": 13.71,
        "power_limit": 120,
        "clock_graphics": 300,
        "clock_sm": 300,
        "clock_memory": 405,
        "clock_video": 540,
        "max_clock_graphics": 2100,
        "max_clock_sm": 2100,
        "max_clock_memory": 4001,
        "pcie_link_gen_current": 1,
        "pcie_link_width_current": 16,
        "pcie_replay_counter": 0,
        "clocks_event_reasons": 1
      },
      "processes": [
        { "pid": 99627,  "type": "G",   "name": "/usr/bin/gnome-shell",   "used_memory": 150 },
        { "pid": 100052, "type": "G",   "name": "/usr/bin/Xwayland",      "used_memory": 3 },
        { "pid": 109937, "type": "C+G", "name": "/opt/google/chrome/chrome", "used_memory": 460 }
      ]
    }
  ],
  "amd": [],
  "drm": [
    {
      "card": "card1",
      "pci_bus_id": "00000000:01:00.0",
      "driver_name": "nvidia-drm",
      "driver_version": "0.0.0",
      "driver_date": "0",
      "driver_description": "NVIDIA DRM driver"
    }
  ],
  "nvidia_smi_schema": "v13",
  "errors": [
    "AMD: rocm-smi command is not available"
  ]
}
```

이 응답에서 읽어야 할 것:

- `ecc`·`mig_devices` 키가 **없습니다.** GeForce가 ECC와 MIG를 지원하지 않아 nvidia-smi가
  `N/A`로 답했고, `omitempty`라 키째 빠집니다. `null`이 아니라 부재입니다.
  `serial`·`host_vgpu_mode`·`vgpu_license_status`·`temperature_memory`도 같은 이유로 없습니다.
- `amd`가 빈 배열이고 `errors`에 한 줄이 있지만 `nvidia` 수집은 정상입니다.
- `drm[0].pci_bus_id`가 `nvidia[0].device_attribute.pci_bus_id`와 같은 값이라 두 배열을
  짝지을 수 있습니다.
- `clocks_event_reasons`는 클럭 이벤트(스로틀) 사유 비트마스크입니다. `0`은 "보고됐고 활성 사유
  없음", 키 부재는 "사유를 아예 보고하지 않음"입니다.

가상화 환경에서는 같은 코드가 다르게 채웁니다. GPU 패스스루된 클라우드 VM(Tesla T4)에서 실측한
차이입니다.

| 필드 | 물리 (GeForce) | 패스스루 VM (Tesla) |
|------|----------------|---------------------|
| `virtualization_mode` | `"None"` | `"Pass-Through"` |
| `ecc` | 키 없음 | `{"mode": "Enabled", ...}` |
| `serial` | 키 없음 | 보고됨 |
| `fan_speed` | 보고됨 | 키 없음 (수동냉각) |
| `performance_state` | `"P8"` (절전) | `"P0"` 고정 |
| `processes[].type` | `G` / `C+G` | `C` |
| `drm` | `nvidia-drm`, PCI 주소 있음 | `simpledrm`, PCI 주소 없음 |

마지막 줄이 위에서 말한 `nvidia`와 `drm`의 역할 차이입니다. 그 VM 이미지에는 `nvidia_drm` 커널
모듈이 로드되지 않아 `/sys/class/drm`에 NVIDIA 노드가 없었고, 잡힌 `simpledrm`은 하이퍼바이저
프레임버퍼입니다.

#### `amd` 배열의 모양

AMD는 `rocm-smi --json` 출력을 파싱해 채웁니다. 카드 하나가 배열 항목 하나입니다.

```json
"amd": [
  {
    "device_attribute": {
      "card": "card0",
      "gpu_id": "0x738c",
      "product_name": "Instinct MI100",
      "serial_number": "0",
      "pci_bus_id": "0000:1E:00.0",
      "vbios_version": "113-D3430400-037",
      "driver_version": "6.2.4",
      "vendor_id": "Advanced Micro Devices, Inc. [AMD/ATI]",
      "device_id": "0x738c"
    },
    "performance": {
      "gpu_usage": 0,
      "memory_usage": 7,
      "vram_memory_used": 2294,
      "vram_memory_total": 32752,
      "performance_level": "auto",
      "fan_speed": 0,
      "temperature_gpu": 31,
      "temperature_memory": 30,
      "power_draw": 39,
      "power_cap": 290,
      "clock_sm": 300,
      "clock_memory": 1200
    }
  }
]
```

rocm-smi 출력을 그대로 옮기지 않고 변환하는 것이 있습니다.

| rocm-smi 가 주는 것 | 응답 필드 | 변환 |
|---------------------|-----------|------|
| `"VRAM Total Memory (B)": "205939376128"` | `vram_memory_total` | 바이트 → MiB |
| `"sclk clock speed:": "(132Mhz)"` | `clock_sm` | 괄호와 단위를 벗기고 숫자만 |
| `Temperature (Sensor edge)` | `temperature_gpu` | |
| `Temperature (Sensor memory)` | `temperature_memory` | `Sensor junction`은 쓰지 않습니다 |
| `Average Graphics Package Power (W)` | `power_draw` | |
| `Max Graphics Package Power (W)` | `power_cap` | |
| 최상위 `system` 항목의 `Driver version` | 각 카드의 `driver_version` | 카드마다 채워 넣습니다 |

주의할 점:

- **카드 순서는 rocm-smi 출력 순서가 아니라 카드 이름 순입니다.** 수집할 때마다 순서가
  흔들리지 않게 정렬합니다.
- **카드마다 필드 수가 다를 수 있습니다.** rocm-smi가 카드별로 다른 블록을 내면 그대로
  반영되고, 없는 항목은 키째 빠집니다.
- `"performance": {}`가 나올 수 있습니다. 카드는 인식됐는데 읽어낸 수치가 하나도 없는
  상태이며, `amd`가 빈 배열인 것(카드 없음)과 다릅니다.
- `compute_partition`·`memory_partition`은 파티셔닝을 지원하는 데이터센터 카드에서만 나옵니다.

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
