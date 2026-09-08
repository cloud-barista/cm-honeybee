# cm-honeybee 신규 작업 내역 2 - GPU 수집

v0.6.1(2026-08-28) 이후 main 에 들어간 **GPU 관련 작업**만 뽑았다.
[`cm-honeybee_new.md`](cm-honeybee_new.md) 는 v0.5.5~v0.6.x 의 OpenBao·패키지 수집 작업을 다루고,
이 문서는 그 다음에 들어온 GPU 갈래를 다룬다.

**전부 릴리즈에 들어가지 않았다.** 아래 커밋 9개는 어느 태그에도 없고 main/edge 에만 있다.

| 커밋 | 날짜 | 제목 |
|------|------|------|
| `5209d06` | 09-03 | agent: Parse nvidia-smi per XML schema version, add AMD collection and bound every GPU command |
| `24b78c0` | 09-03 | agent, server: Regenerate swagger for the expanded GPU model |
| `e5c473e` | 09-03 | agent: Collect the GPU extended resources a Kubernetes node advertises |
| `7965dd2` | 09-03 | docs: Cover the GPU fields the collectors gained |
| `235e780` | 09-04 | server: Bump the agent module to the commit that carries the GPU model |
| `58b62a1` | 09-04 | agent: Read an older nvidia-smi schema with the parser that fits it |
| `e24a40a` | 09-04 | server: Move the agent pin to the schema fix so the image matches HEAD |
| `72e960c` | 09-04 | docs: Correct what nvidia_smi_schema holds when nvidia-smi did not run |
| `48783bb` | 09-04 | docs: Record what the GPU fields mean once measured on real hardware |

---

## 1. nvidia-smi XML 을 스키마 버전별 파서로 읽는다

nvidia-smi 는 드라이버 세대마다 XML 요소 이름을 바꾼다. 이전에는 구조체 하나로 전부 받아서,
이름이 바뀐 요소는 **아무 말 없이 빠진 채** 수집됐다.

- 출력의 DOCTYPE(`nvsmi_device_<버전>.dtd`)을 읽어 v11 / v12 / v13 파서로 나눠 보낸다
  (`agent/gpu/nvidia/schema.go:59` detectSchema, `:96` parse).
- **파서가 없는 버전은 범위의 어느 쪽인지로 가른다** (`58b62a1`).
  v13 보다 새 문서는 v13 파서로(새 버전은 요소를 추가하지 이름을 바꾸지 않는다),
  v11 보다 낡은 문서는 v11 파서로 읽는다. v13 이 `power_readings` 를 `gpu_power_readings` 로
  바꿨기 때문에, v9 문서를 v13 파서로 읽으면 전력 값이 조용히 사라진다
  (`schema.go:20` latestSchema, `:26` oldestSchema, `:118`).
- **대체 파서로 읽은 것은 그렇다고 적는다.** 응답의 `nvidia_smi_schema` 가
  `"v9 (read as v11)"` 형태로 나온다 (`schema.go:47` readAs).
- DOCTYPE 이 아예 없는 옛 드라이버 출력은 v11 로 읽는다 (`schema.go:90`).
- nvidia-smi 가 돌지 않았으면 `nvidia_smi_schema` 는 **빈 값이고 omitempty 로 키째 빠진다**
  (`agent/pkg/api/rest/model/onprem/infra/gpu.go:170`).

## 2. "N/A" 를 0 으로 읽던 것을 고쳤다

nvidia-smi 가 `N/A` / `Not Supported` 로 답한 값이 0 으로 파싱돼 **실제 측정값처럼** 보고됐다.
메모리 사용률은 그 0 으로 나눗셈까지 했다.

- 숫자 필드를 전부 포인터로 바꿨다. `nil` = 보고 안 됨, `0` = 0 으로 보고됨이 구분된다
  (`gpu.go:30` NVIDIAPerformance, `:69` NVIDIAECCErrors).
- 문자열 필드도 `N/A` / `Not Supported` 는 빈 값으로 두고 omitempty 로 뺀다 (`gpu.go:3-5` 주석).
- 소비자 쪽 주의: **미지원 장비에서는 필드가 `null` 이 아니라 키째 사라진다.**
  MIG·ECC·vGPU 가 그렇다.

## 3. NVIDIA 수집 필드가 늘었다

`device_attribute` 6필드 → **15필드**, `performance` 는 클럭·전력·PCIe·ECC까지 확장.
`ecc` / `mig_devices` / `processes` 블록이 새로 생겼다 (`gpu.go:6-111`).

| 블록 | 새로 들어온 것 |
|------|----------------|
| `device_attribute` | nvml_version, minor_number, serial, pci_bus_id, vbios_version, compute_mode, persistence_mode, mig_mode, **virtualization_mode**, host_vgpu_mode, vgpu_license_status |
| `performance` | FB/BAR1 메모리 4종씩, 온도(GPU·메모리), 전력(draw·limit), 클럭 4종 + 최대 클럭 3종, PCIe gen/width/replay, clocks_event_reasons(스로틀 사유 비트마스크) |
| `ecc` | mode + volatile/aggregate 각각 SRAM·DRAM correctable/uncorrectable. pre-Volta 스키마는 single_bit/double_bit |
| `mig_devices` | MIG 인스턴스별 index, gpu/compute instance id, uuid, FB 메모리 |
| `processes` | pid, type, name, used_memory, gpu_instance_id |

`virtualization_mode` 는 **응답만으로 물리/VM 을 가른다** - 실측에서 물리는 `"None"`,
GPU 패스스루된 클라우드 VM 은 `"Pass-Through"` 였다.

## 4. AMD(rocm-smi) 수집 신규

이전에는 AMD GPU 를 아예 수집하지 않았다.

- `rocm-smi --json` 으로 수집하고, 플래그를 거부하는 릴리즈는 `-a --json` 으로 폴백
  (`agent/gpu/amd/rocm-smi.go:38` queryArgs, `:58` fallbackArgs, `agent/gpu/amd/stats.go:64`).
- PATH 에 없으면 `/opt/rocm/bin/rocm-smi`, `/usr/bin/rocm-smi` 를 본다 - ROCm 프로필 스크립트를
  source 하지 않은 호스트를 위한 보강 (`rocm-smi.go:29`).
- `device_attribute` 13필드(card, gpu_uuid, unique_id, product_name, serial, pci_bus_id, vbios,
  driver_version, vendor/device id, **compute_partition / memory_partition**),
  `performance` 12필드 (`gpu.go:115-146`).
- AMD 가 없는 호스트는 `errors` 에 `"AMD: rocm-smi command is not available"` 한 줄이 들어가고
  수집 전체는 성공한다.

## 5. 모든 GPU 명령에 타임아웃, 3수집기 병렬

이전 구현은 셸을 통해 명령을 부르면서 **바운드가 전혀 없었고** stderr 를 stdout 에 섞었다.
드라이버가 물리면 그 블록이 `/infra` 요청까지 그대로 올라온다.

- `nvidia-smi -q -x` 30초, `nvidia-smi --version` 10초, `rocm-smi` 30초.
  프로세스를 죽인 뒤 파이프 대기도 5초로 바운드
  (`agent/gpu/nvidia/nvidia-smi.go:17-22`, `agent/gpu/amd/rocm-smi.go:17-21`).
- stdout 을 따로 받는다. nvidia-smi 가 stderr 로 내는 경고가 XML 에 섞이면 파싱이 깨진다.
- 로케일을 고정해 숫자 표기가 agent 환경을 따라가지 않게 했다.
- NVIDIA / AMD / DRM 세 수집기를 goroutine 으로 함께 돌린다. 총 대기가 합이 아니라
  **가장 느린 하나**로 묶인다 (`agent/driver/infra/gpu.go:38-55`).
- `GetGPUInfo` 는 항상 nil 에러를 반환한다. GPU 수집 실패가 `/infra` 전체를 죽이지 않고
  `errors` 배열로만 올라온다 (`agent/driver/infra/gpu.go:62-70`).

## 6. DRM 수집기 재작성

- `drm.ListDevices()` 대신 `/dev/dri` 를 직접 훑는다. 카드 번호가 결과에 남고 디바이스 파일이
  다시 닫힌다 (`agent/gpu/drm/drm_unix.go:31` GetDRMInfo, `:95` listCards).
- `card` 번호와 `pci_bus_id` 를 채워서 nvidia/amd 배열의 GPU 와 짝지을 수 있다
  (`drm_unix.go:126` cardPCIBusID, `gpu.go:153` DRM).
- 카드 열기에 읽기·쓰기 권한이 필요해 agent 사용자로 못 여는 카드가 있을 수 있고, 그 경우
  건너뛴다.
- **Windows 는 여전히 빈 스텁이다** (`agent/gpu/drm/drm_windows.go:9`).

## 7. 쿠버네티스 노드의 GPU 확장 리소스 수집 (`e5c473e`)

쿠버네티스에는 GPU 를 나타내는 기본 리소스가 없다. 클러스터가 GPU 를 가졌다고 말하는 곳은
`nvidia.com/gpu` 같은 벤더 확장 리소스뿐인데 이걸 안 봤다 - **GPU 클러스터와 아닌 클러스터가
응답상 같았다.**

- 노드 status 의 capacity/allocatable 에서 GPU 확장 리소스를 뽑는다
  (`agent/driver/kubernetes/gpu.go:27` parseNodeGPU, `agent/driver/kubernetes/node.go:26`).
- 벤더 도메인 `nvidia.com/`, `amd.com/`, `gpu.intel.com/` 를 인식하고, 이름에 `gpu` 가 들어가거나
  `nvidia.com/mig-` 로 시작하는 것을 GPU 로 판정한다 (`gpu.go:17-21`, `:58` isGPUResource).
- 장치 상세(product, driver_version, memory, mig_capable, mig_strategy)는 NFD/GFD 가 붙인
  **노드 레이블**에서 읽는다. 순수 device plugin 만 도는 클러스터에서는 비어 있고, 추측하지 않는다
  (`agent/pkg/api/rest/model/onprem/kubernetes/kubernetes.go` NodeGPU).
- 맵 순회 순서가 랜덤이라 resource_name 으로 정렬해 수집마다 순서가 흔들리지 않게 했다.

## 8. 서버 반영 (`235e780`, `e24a40a`, `24b78c0`)

server 는 agent 모델을 Go 모듈로 가져다 쓰는데, `server/Dockerfile` 이 `server/` 만 빌드 컨텍스트로
받아 `go.work` 를 못 본다. **즉 Docker 이미지는 항상 `server/go.mod` 의 핀을 쓴다.**
핀이 낡으면 agent 가 보낸 새 GPU 필드가 역직렬화에서 조용히 사라진다.

- 핀을 GPU 모델이 든 커밋으로 올렸고(`235e780`), 스키마 수정까지 반영해 HEAD 와 맞췄다
  (`e24a40a`, `server/go.mod:17` = `58b62a1`).
- agent·server 양쪽 swagger 재생성 (`24b78c0`).

**refined 에는 GPU 가 없다.** `server/` Go 코드에 `gpu` 문자열이 0건이고, 대상 모델
`cm-beetle/imdl@v0.1.12/on-premise-model` 에 GPU 필드 자체가 없다.
`infra.gpu` 와 쿠버네티스 노드의 `spec.gpu` 는 **원시 엔드포인트로만** 나온다
(`docs/server-api.md:30`).

## 9. 문서 (`7965dd2`, `72e960c`, `48783bb`)

`docs/agent-api.md` / `docs/server-api.md` 에 반영한 것:
- `nvidia` / `amd` / `drm` 세 배열의 역할이 다르다는 것. **GPU 유무는 `nvidia`/`amd` 로 판단한다**
  (`agent-api.md:101-104`).
- `errors` 가 비어 있지 않다고 수집 전체가 실패한 것이 아니라는 것 (`agent-api.md:97`).
- v13 의 `power_readings` → `gpu_power_readings` 개명과 대체 파서 표기 (`agent-api.md:132`).
- 쿠버네티스에 기본 GPU 리소스가 없다는 배경과 `gpu[]` 의 출처 (`agent-api.md:168-169`).
- 실측으로 확인한 `virtualization_mode` 값 (`agent-api.md:107`).

---

## 실제 수집 결과 예시

수집기가 만든 값이지 손으로 쓴 예시가 아니다. 각 예시마다 **어디서 어떻게 받았는지**를 같이 적는다.

### 예시 1 - 물리 서버 x Linux x NVIDIA

| 항목 | 값 |
|------|-----|
| CSP / 리전 | 없음. 사내 물리 워크스테이션 |
| GPU | NVIDIA GeForce GTX 1660 (Turing), 6GB, 1장 |
| OS / 드라이버 | Ubuntu, 드라이버 610.57.04 / CUDA 13.3 / NVML 610.57 |
| agent | HEAD 빌드, md5 `8df9b5003a5f1aa841cd75cc6f81626b` |
| 수집 방법 | agent 를 systemd 로 띄운 뒤 `GET /honeybee-agent/infra` 를 loopback 으로 호출 |
| 응답 | HTTP 200, 27,442 B, 0.292s. 아래는 그 응답의 `gpu` 절만 잘라낸 것 |
| nvidia-smi 원본 | `<!DOCTYPE nvidia_smi_log SYSTEM "nvsmi_device_v13.dtd">` -> `nvidia_smi_schema` 가 `"v13"` 로 채워졌다 |

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
        {
          "pid": 99627,
          "type": "G",
          "name": "/usr/bin/gnome-shell",
          "used_memory": 150
        },
        {
          "pid": 99875,
          "type": "G",
          "name": "/opt/teamviewer/tv_bin/TeamViewer",
          "used_memory": 21
        },
        {
          "pid": 100052,
          "type": "G",
          "name": "/usr/bin/Xwayland",
          "used_memory": 3
        },
        {
          "pid": 100664,
          "type": "C+G",
          "name": "gjs",
          "used_memory": 83
        },
        {
          "pid": 109937,
          "type": "C+G",
          "name": "/opt/google/chrome/chrome ...(인자 생략)",
          "used_memory": 460
        },
        {
          "pid": 624393,
          "type": "C+G",
          "name": "/usr/share/code/code ...(인자 생략)",
          "used_memory": 111
        },
        {
          "pid": 838284,
          "type": "C+G",
          "name": "/usr/bin/nautilus",
          "used_memory": 60
        },
        {
          "pid": 987576,
          "type": "C+G",
          "name": "/usr/bin/gnome-text-editor",
          "used_memory": 28
        },
        {
          "pid": 1353631,
          "type": "C+G",
          "name": "/usr/bin/gnome-system-monitor",
          "used_memory": 28
        }
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
}```

이 예시에서 읽을 것:

- `nvidia_smi_schema: "v13"` - 괄호가 없으므로 **v13 문서를 v13 파서로 읽었다.** 대체 파서가 아니다.
- `virtualization_mode: "None"` - 물리 장비다.
- `ecc` / `mig_devices` 키가 **아예 없다.** GeForce 가 ECC·MIG 를 지원하지 않아 nvidia-smi 가 N/A 로
  답했고, omitempty 로 키째 빠졌다. `null` 이 아니라 부재다.
- `serial`, `host_vgpu_mode`, `vgpu_license_status`, `temperature_memory` 도 같은 이유로 없다.
- `amd: []` + `errors` 한 줄 - **AMD 가 없다는 것이지 수집이 실패한 것이 아니다.**
  `nvidia` 는 정상적으로 채워져 있다.
- `drm[0].card: "card1"`, `pci_bus_id: "00000000:01:00.0"` - `nvidia[0].device_attribute.pci_bus_id`
  와 같은 값이라 두 배열을 짝지을 수 있다. 이 짝짓기가 `5209d06` 에서 추가된 것이다.
- `clocks_event_reasons: 1` - 스로틀 사유 비트마스크. 0 이 아니라 1 이므로 "보고됐고 무언가 켜져 있다".
- `performance_state: "P8"`, `clock_sm: 300` (최대 2100) - 유휴 상태다.

### 예시 2 - 클라우드 VM x Ubuntu x NVIDIA (2026-09-04 실측 기록)

**이 장비는 검증 후 삭제해서 지금 다시 뽑을 수 없다.** 아래는 그때 남긴 측정 기록이고,
JSON 원문이 아니라 필드 단위 기록이다.

| 항목 | 값 |
|------|-----|
| CSP / 리전 | AWS `ap-northeast-2` (서울) |
| 스펙 | `g4dn.xlarge` - NVIDIA Tesla T4 x1, 4vCPU / 16GiB, $0.647/h |
| 이미지 | `ami-00ffffe72a332c865` (Deep Learning Base OSS Nvidia Driver GPU AMI, Ubuntu 22.04) |
| 루트 디스크 | 100GB (60GB 로 요청했다가 최소 75GB 미달로 생성 실패해서 다시 만들었다) |
| OS / 드라이버 | Ubuntu 22.04.5 LTS, 커널 6.8.0-1061-aws, 드라이버 595.91.07 / CUDA 13.2 |
| agent | HEAD 빌드를 scp 로 수동 반영. md5 `8df9b5003a5f1aa841cd75cc6f81626b` 로 전송본=설치본 일치 확인 |
| 수집 방법 | 사내 검증 노드의 honeybee server 에 소스그룹·커넥션을 등록하고 `POST /import/infra` -> `GET /infra` |
| 응답 | import 200 / 0.77s, GET 200 / 7,716 B |
| 정리 | `option=terminate` 로 삭제(force 미사용). 삭제 후 spider 로 VM·VPC·SG·키페어·디스크 전부 0 확인. 과금 약 32분 $0.35 |

**같은 코드가 물리와 VM 에서 다르게 채우는 것** - 이 대조가 이번 검증의 핵심이다.

| 필드 | 물리 GTX 1660 | VM Tesla T4 | 왜 다른가 |
|------|---------------|-------------|-----------|
| `virtualization_mode` | `"None"` | `"Pass-Through"` | **응답만으로 물리/VM 을 가른다** |
| `ecc` | 키 없음 | `{mode:"Enabled", volatile/aggregate 전부 0}` | Tesla 만 ECC 지원 |
| `serial` | 키 없음 | `"1325119103531"` | 컨슈머 카드는 미보고 |
| `fan_speed` | `40` | 키 없음 | T4 는 무팬 수동냉각이라 N/A |
| `performance_state` | `"P8"` (절전) | `"P0"` 고정 | VM 은 persistence_mode=Enabled |
| `processes[].type` | `G` / `C+G` (디스플레이) | `C` (순수 컴퓨트) | 패스스루 게스트에서도 프로세스 수집은 정상 |
| `product_brand` | `"GeForce"` | `"NVIDIA"` | |
| `fb_memory_total` / `power_limit` | `6144` / `120` | `15360` / `70` | |
| `device_attribute` 필드 수 | 14 | 15 | T4 는 `serial` 이 더 있다 |
| `drm` | `card1` / `nvidia-drm` / PCI 있음 | `card0` / `simpledrm` / PCI 없음 | 아래 참조 |

**VM 의 `drm` 이 NVIDIA 를 못 담은 것은 수집기 버그가 아니다.** 그 AMI 에 `nvidia_drm` 커널 모듈이
로드되지 않아 `/sys/class/drm` 에 NVIDIA 노드가 없었다 (`lsmod` 에 nvidia_drm 없음,
`/proc/driver/nvidia/gpus/0000:00:1e.0` 는 존재). 잡힌 `simpledrm` 은 하이퍼바이저 프레임버퍼다.
**GPU 유무는 `nvidia` / `amd` 배열로 판단해야 하고 `drm` 으로 판단하면 안 된다.**

부하를 걸어 값이 따라 움직이는 것도 확인했다 (고정값이 아니라는 증거):
`gpu_usage` 0 -> 100 -> 0, `fb_memory_used` 0 -> 2153 -> 0, `clock_sm` 1590 -> 1485 -> 1590,
`processes` 부재 -> 부하 프로세스 2150MiB -> 부재. 동시에 캡처한 nvidia-smi 와 값까지 일치했다.

### 예시 3 - AMD

**실장비 출력 예시가 없다.** rocm-smi 가 실제로 도는 장비를 아직 못 구했다.
지금 낼 수 있는 것은 **AMD 가 없을 때의 출력**뿐이고, 그것이 예시 1 의
`"amd": []` + `errors: ["AMD: rocm-smi command is not available"]` 다.

2026-09-08 기준 AWS 재확인 결과, `g4ad` 계열은 여전히 잡히지 않는다.

| 리전 | g4ad.xlarge / 2xlarge / 4xlarge | 대조군 (t3.small, g4dn.xlarge, g5.xlarge, g6.xlarge) |
|------|--------------------------------|------------------------------------------------------|
| us-east-1 | 셋 다 `Unavailable` | 넷 다 `Available` |
| us-east-2 | 셋 다 `Unavailable` | - |
| us-west-2 | 셋 다 `Unavailable` | - |

오류 문구는 셋 다 같다:
`instance type "g4ad.xlarge" is not offered in any AZ of region "us-east-1" (DescribeInstanceTypeOfferings returned no results)`.

**대조군이 전부 `Available` 이므로 조회 경로 자체는 정상이다.** g4ad 만 이 계정에 안 열려 있다.
인스턴스는 만들지 않았고 dry-run 만 돌렸다.

---

## 검증 상태

**실장비 2대에서 실측했다.** 단위테스트는 합성 픽스처 기반이라 별개로 본다.

| 조합 | 상태 | 근거 |
|------|------|------|
| 물리 x Linux x NVIDIA | **실측 완료** | GTX 1660, 드라이버 610.57.04 / CUDA 13.3, DTD v13. `strace` 로 `execve("/usr/bin/nvidia-smi",["-q","-x"])` 확인 - 목 아님. 부하 대조에서 gpu_usage 0→24, P8→P0, clock 300→1950 으로 값이 따라 움직임 |
| VM(패스스루) x Ubuntu x NVIDIA | **실측 완료** | AWS g4dn.xlarge Tesla T4, 드라이버 595.91.07 / CUDA 13.2. `device_attribute` 6→15필드, `ecc` 블록 신규, `virtualization_mode="Pass-Through"`. 부하 대조 gpu_usage 0→100→0 |
| server 통과 (end-to-end) | **실측 완료** | 커밋 `e24a40a` 이미지로 배포한 뒤 import 200/0.77s → GET 200/7716B. `gpu` 키 5개, `nvidia_smi_schema="v13"` 통과 확인 |
| 낡은 핀의 영향 | **반증으로 확정** | 같은 원시 JSON 을 낡은 핀 `06a9173` 으로 역직렬화하면 `gpu` 키 3개, `device_attribute` 6필드, `virtualization_mode`/`vbios_version`/`nvidia_smi_schema` 전부 nil |
| AMD (rocm-smi 실동작) | **미시험** | 픽스처 단위테스트 4건과 "AMD 부재 시 오류 경로"만 확인. rocm-smi 가 실제로 도는 장비에서 수집한 적 없음 |
| 쿠버네티스 GPU 확장 리소스 | **미시험** | 접근 가능한 클러스터에 GPU 노드가 없어 확장 리소스 0개. 코드 경로가 실제로 채우는 것을 못 봄 |
| Windows | **미시험** | 크로스 빌드만 통과(`GOOS=windows go build ./...` exit 0). DRM 은 빈 스텁이고 nvidia-smi/rocm-smi 를 OS 분기 없이 부른다 |
| MIG / 다중 GPU / 비 root 프로세스 가림 | **미시험** | 단일 GPU 장비 2대에서만 봤다 |

단위테스트 (2026-09-04 실행, `go test -count=1`):
`gpu/amd` ok 0.003s · `gpu/nvidia` ok 0.005s · `driver/kubernetes` ok 0.038s.
`gpu/drm`, `gpu/nvidia/common`, `schema_v11~v13` 은 테스트 파일 없음.

### AMD 실장비 확보 실패

클라우드로 AMD 장비를 잡으려 3회 시도해 전부 실패했고 원인이 셋 다 다르다.

| 시도 | 리전 | 원인 |
|------|------|------|
| 1 | azure-koreacentral | `OverconstrainedZonalAllocationRequest` (Azure 재고) |
| 2 | azure-eastus | `Security Group does not exist` (낡은 tumblebug 기록) |
| 3 | azure-eastus | `connection reset by peer` (Azure 관리 API 로 나가는 경로 문제, 431초 후) |

구조적 난관: 카탈로그의 AMD GPU 는 전부 그래픽용(MI25/V520/V620/V710)이고 ROCm 정식 지원
컴퓨트 카드(MI200/MI300)가 없다. AWS g4ad 는 ROCm 포함 이미지가 없고 드라이버가 S3 + IAM 역할을
요구하는데 tumblebug 이 만든 인스턴스에는 그 역할이 없다. Azure `microsoft-dsvm:ubuntu-hpc:2204-rocm`
이미지가 유일한 활로다.

---

## 남은 것

| 무엇 | 상태 |
|------|------|
| **릴리즈** | GPU 커밋 9개가 어느 태그에도 없다. 최신 태그는 v0.6.1 |
| **agent 자동 설치** | `server/lib/ssh/sourceFiles/copyAgent.sh` 가 releases/latest(=v0.6.1)에서 받는다. 릴리즈 전에는 소스 호스트에 **GPU 코드 없는 agent 가 깔린다** (md5 대조로 확인) |
| refined | GPU 미포함. 대상 모델 imdl 에 필드가 없어 cm-beetle 이 refined 로는 GPU 를 못 본다 |
| `/kubernetes` 응답 | "클러스터 없음" 과 "수집 실패" 를 구분할 수 없다 (200 + 빈 구조, `errors` 필드 없음). `gpu` 섹션에는 `errors` 가 있는데 여기는 없다 |
| omitempty 설명 | nvidia-smi 가 N/A 로 주는 값이 키째 사라진다는 것이 swagger 설명에 없다 |
| 검증 노드 정리 | 전송해 둔 이미지 tar(60MB) 삭제. compose 를 정식 이미지로 원복하는 것은 릴리즈 후 별건 |
| 롤백 태그 | `cloudbaristaorg/cm-honeybee:rollback-before-gpuchk` = `ef108965a4de` |
