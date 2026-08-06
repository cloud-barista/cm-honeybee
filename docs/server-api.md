# CM-Honeybee Server API

**Server**(`cm-honeybee`)는 컨트롤 플레인입니다. **SourceGroup**과 **ConnectionInfo**를 관리하고,
원시 소스 데이터를 수집(에이전트 풀, SSH, 또는 CSP 소스의 경우 cb-spider 경유)·저장하며, `cm-beetle`이
마이그레이션에 사용하는 **정제된 소스 모델**을 제공합니다.

| 항목 | 값 |
|------|-----|
| 모듈명 | `HONEYBEE` |
| Base path | `/honeybee` |
| 기본 포트 | `8081` |
| 의존성 | `cm-honeybee-agent` (`:8082`), `cb-spider` (엔드포인트 설정 가능) |
| Swagger UI | `http://<host>:8081/honeybee/api/index.html` |
| 인증 | 없음 |

> 아래 모든 경로는 base path 기준 상대 경로입니다. 전체 URL 예시:
> `http://localhost:8081/honeybee/source_group`

## 핵심 개념

- **SourceGroup** — 함께 마이그레이션할 소스 머신들의 논리적 그룹. 온프레미스 그룹과 `csp`(클라우드)
  그룹 등의 타입이 있습니다. SourceGroup은 등록된 *target* 정보(마이그레이션 후 cm-beetle이 돌려준
  결과)도 함께 가질 수 있습니다.
- **ConnectionInfo** — SourceGroup에 속한 개별 소스 노드의 연결 정보(IP, SSH 포트, 사용자/자격 증명).
  Server는 이를 사용해 호스트/에이전트에 접근하여 데이터를 수집합니다.
- **원시(Raw) vs 정제(Refined)** — `/infra`, `/software`, `/kubernetes`, `/helm`, `/data`는 수집된
  원시 데이터를 반환합니다. `/.../refined` 엔드포인트는 다운스트림에서 사용하는 정규화된 모델
  (`github.com/cloud-barista/cm-beetle/imdl/on-premise-model`)을 반환합니다.

---

## 엔드포인트 요약

### System
| 메서드 | 경로 | 설명 |
|--------|------|------|
| GET | `/readyz` | 준비 상태 확인. |

### SourceGroup
| 메서드 | 경로 | 설명 |
|--------|------|------|
| POST | `/source_group` | SourceGroup 등록. |
| GET | `/source_group` | SourceGroup 목록 (페이징·필터 가능). |
| GET | `/source_group/{sgId}` | SourceGroup 단건 조회. |
| PUT | `/source_group/{sgId}` | SourceGroup 수정. |
| DELETE | `/source_group/{sgId}` | SourceGroup 삭제. |
| PUT | `/source_group/{sgId}/refresh` | 그룹 전체의 연결 정보 상태 갱신. |
| POST | `/source_group/{sgId}/target` | 그룹에 target 정보(cm-beetle 마이그레이션 결과) 등록. |
| GET | `/source_group/{sgId}/discover` | 그룹이 접근 가능한 CSP 리소스 디스커버리. |

### ConnectionInfo
| 메서드 | 경로 | 설명 |
|--------|------|------|
| POST | `/source_group/{sgId}/connection_info` | ConnectionInfo 생성. |
| GET | `/source_group/{sgId}/connection_info` | ConnectionInfo 목록 (페이징·필터 가능). |
| GET | `/source_group/{sgId}/connection_info/{connId}` | ConnectionInfo 조회. |
| PUT | `/source_group/{sgId}/connection_info/{connId}` | ConnectionInfo 수정. |
| DELETE | `/source_group/{sgId}/connection_info/{connId}` | ConnectionInfo 삭제. |
| PUT | `/source_group/{sgId}/connection_info/{connId}/refresh` | 단일 연결 상태 갱신. |
| GET | `/connection_info/{connId}` | sgId 없이 ConnectionInfo 직접 조회. |
| PUT | `/connection_info/{connId}/refresh` | sgId 없이 단일 연결 상태 직접 갱신. |

### CSP / Discovery
| 메서드 | 경로 | 설명 |
|--------|------|------|
| GET | `/csp` | 연결된 cb-spider가 지원하는 CSP 목록. |
| GET | `/csp/{name}` | CSP 메타데이터(자격 증명 키+예시, 리전 키 등) 조회. 대소문자 무시, 미지원 시 지원목록 안내. |
| GET | `/source_group/{sgId}/region` | `csp` SourceGroup의 **실제 리전/존 목록**(저장 credential로 live 조회). |
| GET | `/source_group/{sgId}/discover` | `csp` SourceGroup의 VM / K8s 클러스터 / 오브젝트 스토리지 디스커버리. |

### 원시 소스 정보 수집
| 메서드 | 경로 | 설명 |
|--------|------|------|
| GET | `/source_group/{sgId}/connection_info/{connId}/infra` | 노드 1개의 인프라 정보. |
| GET | `/source_group/{sgId}/infra` | 그룹 전체의 인프라 정보. |
| GET | `/source_group/{sgId}/connection_info/{connId}/software` | 노드 1개의 소프트웨어 정보. |
| GET | `/source_group/{sgId}/software` | 그룹 전체의 소프트웨어 정보. |
| GET | `/source_group/{sgId}/connection_info/{connId}/kubernetes` | 노드 1개의 쿠버네티스 정보. |
| GET | `/source_group/{sgId}/kubernetes` | 그룹 전체의 쿠버네티스 정보. |
| GET | `/source_group/{sgId}/connection_info/{connId}/helm` | 노드 1개의 Helm 정보. |
| GET | `/source_group/{sgId}/helm` | 그룹 전체의 Helm 정보. |
| GET | `/source_group/{sgId}/connection_info/{connId}/data` | 노드 1개의 데이터 정보. |
| GET | `/source_group/{sgId}/data` | 그룹 전체의 데이터 정보. |

### 정제된 소스 모델
| 메서드 | 경로 | 설명 |
|--------|------|------|
| GET | `/source_group/{sgId}/connection_info/{connId}/infra/refined` | 노드 1개의 정제 인프라. |
| GET | `/source_group/{sgId}/infra/refined` | 그룹 전체의 정제 인프라. |
| GET | `/source_group/{sgId}/connection_info/{connId}/software/refined` | 노드 1개의 정제 소프트웨어. |
| GET | `/source_group/{sgId}/software/refined` | 그룹 전체의 정제 소프트웨어. |

### Import (수집 + 저장)
| 메서드 | 경로 | 설명 |
|--------|------|------|
| POST | `/source_group/{sgId}/connection_info/{connId}/import/infra` | 노드 1개의 인프라 수집·저장. |
| POST | `/source_group/{sgId}/import/infra` | 그룹 전체의 인프라 import. |
| POST | `/source_group/{sgId}/connection_info/{connId}/import/software` | 노드 1개의 소프트웨어 import. |
| POST | `/source_group/{sgId}/import/software` | 그룹 전체의 소프트웨어 import. |
| POST | `/source_group/{sgId}/connection_info/{connId}/import/kubernetes` | 노드 1개의 쿠버네티스 import. |
| POST | `/source_group/{sgId}/import/kubernetes` | 그룹 전체의 쿠버네티스 import. |
| POST | `/source_group/{sgId}/connection_info/{connId}/import/helm` | 노드 1개의 Helm import. |
| POST | `/source_group/{sgId}/import/helm` | 그룹 전체의 Helm import. |
| POST | `/source_group/{sgId}/connection_info/{connId}/import/data` | 노드 1개의 데이터 import. |
| POST | `/source_group/{sgId}/import/data` | 그룹 전체의 데이터 import. |

### Benchmark
| 메서드 | 경로 | 설명 |
|--------|------|------|
| GET | `/bench/{connId}` | 연결의 벤치마크 정보 조회. |
| POST | `/bench/{connId}/run` | 벤치마크 실행 (벤치마크 에이전트가 없으면 자동 설치). |
| POST | `/bench/{connId}/stop` | 실행 중인 벤치마크 중지. |

---

## SourceGroup 타입 (ssh / csp)

모든 SourceGroup은 `type` 필드로 수집 방식을 구분합니다.

| type | 설명 | 연결 정보 입력 방식 |
|------|------|---------------------|
| `ssh` (기본값) | 온프레미스/단순 호스트. SSH로 직접 접속해 수집. | ConnectionInfo에 `ip_address`, `ssh_port`, `user`, `password`/`private_key` 입력. |
| `csp` | cb-spider 기반 클라우드 소스. credential/region으로 CSP에서 VM 메타데이터를 수집. VM이면 **SSH 접속 정보를 추가로 주면 게스트 내부까지 에이전트로 수집**. | SourceGroup에 `provider_name`/`region_name`/`credential[]`, ConnectionInfo에 `resource_type`/`resource_id` (+ 선택적으로 `zone`, `ip_address`/`ssh_port`/`user`/`password`\|`private_key`) 입력. |

- **`type`을 생략하면 `ssh`로 동작**하며, CSP 관련 필드는 모두 optional입니다. 따라서 기존
  (SSH 전용) 클라이언트 페이로드는 수정 없이 그대로 동작합니다(하위 호환).

- **CSP 타입은 두 가지 정보원을 함께 씁니다.**
  - **CSP(cb-spider) 수집** — VM 스펙·이미지·리전·공인/사설 IP·디스크·태그·VPC/서브넷/SG 이름 등
    *클라우드 바깥에서 보이는* 정보. `csp` 섹션으로 노출됩니다.
  - **에이전트(SSH) 수집** — OS·커널·CPU·메모리·디스크 사용량·소프트웨어 등 *게스트 내부* 정보.
    ConnectionInfo에 SSH 접속 정보를 준 경우에만 수집됩니다(에이전트를 설치해 조회).
  - 두 정보는 **분리 저장**되어 서로 덮어쓰지 않습니다(아래 "CSP 수집 동작 상세" 참고).

---

## 전형적인 워크플로우 (SSH 타입 — v0.6.0 권장)

```bash
BASE=http://localhost:8081/honeybee

# 1. SourceGroup 등록 (type 생략 → ssh 기본값)
SG=$(curl -s -X POST $BASE/source_group \
  -H 'Content-Type: application/json' \
  -d '{"name":"on-prem-k8s","description":"소스 k8s 클러스터"}' | jq -r '.id')

# 2. 컨트롤 플레인 노드의 ConnectionInfo 추가 (SSH 접속 정보)
curl -s -X POST $BASE/source_group/$SG/connection_info \
  -H 'Content-Type: application/json' \
  -d '{"name":"cp-1","ip_address":"10.0.0.10","ssh_port":"22","user":"ubuntu", ...}'

# 3. 수집 데이터 import (서버에 저장)
curl -s -X POST $BASE/source_group/$SG/import/infra
curl -s -X POST $BASE/source_group/$SG/import/kubernetes

# 4. 정제 소스 모델 조회 (cm-beetle이 사용하는 결과물)
curl -s $BASE/source_group/$SG/infra/refined | jq

# 5. cm-beetle 마이그레이션 후, 결과 target 정보를 다시 등록
curl -s -X POST $BASE/source_group/$SG/target -d '{ ... }'
```

> `import/*`는 먼저 수집(원시 `GET` 엔드포인트와 동일)한 뒤 결과를 **저장**하여 요청 간에 유지됩니다.
> 반면 `GET /.../infra` 등은 실시간 수집 데이터를 반환하며 반드시 저장되지는 않습니다.

---

## 전형적인 워크플로우 (CSP 타입)

CSP별로 credential 입력 항목이 다른 문제는 **cb-spider가 CSP마다 필요한 credential 키 목록을 알려주는
방식**으로 해결합니다. 클라이언트는 이 키 목록으로 입력 폼을 동적으로 구성하고, 값은 제네릭
`credential: [{key, value}]` 배열로 제출합니다(CSP별 하드코딩 불필요).

> **credential 보관 정책:** credential은 **honeybee가 암호화하여 보관**하며, **cb-spider에는 영구
> 등록하지 않습니다.** 조회가 필요한 시점에만 honeybee가 cb-spider에 credential/region/connection을
> **임시로 등록 → 조회 → 즉시 해제(unregister)** 합니다. 따라서 spider 측에는 credential이 남지
> 않으며, 영구 `ConnectionName` 바인딩도 두지 않습니다.

```bash
BASE=http://localhost:8081/honeybee

# 1. 지원 CSP 목록 조회
curl -s $BASE/csp | jq
#   → { "csp": ["AWS", "GCP", "AZURE", ...] }

# 2. 선택한 CSP의 credential 키 / 리전 메타데이터 조회 (CSP마다 다름)
curl -s $BASE/csp/azure | jq
#   → { "name":"AZURE",
#       "credential_keys":["ClientId","ClientSecret","TenantId","SubscriptionId"],
#       "regions":[...], ... }
#   클라이언트는 credential_keys로 입력 폼을, regions로 리전 드롭다운을 동적 렌더링.

# 3. CSP 타입 SourceGroup 등록 + VM ConnectionInfo (한 번에)
#    connection_info에 resource_type/resource_id(=CSP 리소스 식별) + 선택적 SSH 접속 정보.
SG=$(curl -s -X POST $BASE/source_group \
  -H 'Content-Type: application/json' \
  -d '{
        "name":"azure-source",
        "type":"csp",
        "provider_name":"azure",
        "region_name":"koreacentral",
        "credential":[
          {"key":"ClientId","value":"..."},
          {"key":"ClientSecret","value":"..."},
          {"key":"TenantId","value":"..."},
          {"key":"SubscriptionId","value":"..."}
        ],
        "connection_info":[{
          "name":"vm-1",
          "resource_type":"vm",
          "resource_id":"/subscriptions/<sub>/resourceGroups/<rg>/providers/Microsoft.Compute/virtualMachines/<vm>",
          "ip_address":"40.82.136.176", "user":"ubuntu", "password":"...", "ssh_port":"22"
        }]
      }' | jq -r '.id')
#   등록 시 honeybee는 상태만 확인합니다: (1) cb-spider가 VM을 식별하는지(connection_status),
#   (2) SSH 정보가 있으면 게스트에 에이전트 설치(agent_status). 이 단계에선 데이터를 저장하지 않음.
#   응답의 connection_info_status_count로 connection/agent 성공 수를 확인.

# 4. 수집 + 저장 (csp_data + infra_data) — import 단계에서 수행
curl -s -X POST $BASE/source_group/$SG/import/infra      # CSP 메타 + (SSH 있으면)게스트 내부
curl -s -X POST $BASE/source_group/$SG/import/software    # 게스트 소프트웨어(SSH 필요)

# 5. 통합 조회: compute/network(에이전트) + csp(cb-spider)가 함께 반환됨
curl -s $BASE/source_group/$SG/infra | jq '.servers[0] | {compute, csp}'

# 6. 정제 소스 모델(cm-beetle용) / target 등록은 SSH 흐름과 동일
curl -s $BASE/source_group/$SG/infra/refined | jq
```

> **디스커버리(선택):** resource_id를 모르면 `GET /source_group/{sgId}/discover?resource_type=vm|k8s|object_storage`로
> 해당 CSP의 리소스 목록을 조회해 `resource_id`를 얻을 수 있습니다(이 호출도 임시 connection 등록→조회→해제).

---

## CSP 수집 동작 상세

CSP 타입 소스에서 등록/갱신(`POST /source_group`, `PUT .../refresh`) 시 honeybee가 하는 일과,
정보가 어디서 수집되어 어디에 저장·노출되는지의 전체 과정입니다.

### 1) cb-spider 임시 연결 (credential/region/connection)

honeybee는 조회 때마다 per-call 유니크 이름으로 **credential → region → connectionconfig**를 임시 등록하고,
끝나면 역순으로 해제합니다. 이때 region은 **CSP 메타(`meta.Region`)가 요구하는 키를 모두** 채웁니다.
예를 들어 Azure/AWS는 `Region`과 `Zone`을 모두 요구하므로 `Region`만 보내면 cb-spider가 거부합니다.

- **Zone precedence**: ConnectionInfo의 `zone`(명시) > `region_name`에 `"<region>/<zone>"`로 임베드 >
  provider 기본값 `1`. 즉 `region_name`은 `"koreacentral"` 또는 `"koreacentral/2"`를, ConnectionInfo는
  `"zone":"2"`를 지원하며, 둘 다 있으면 ConnectionInfo의 `zone`이 우선합니다. (리소스 ID 기반 단건
  조회에는 zone 값이 실제로 쓰이지 않지만, cb-spider가 zone 키를 요구하는 CSP가 있어 채워 보냅니다.)
  선택 가능한 zone 목록은 `GET /source_group/{sgId}/region`의 각 리전 `zones`에서 확인합니다.
- **Azure 주의:** cb-spider Azure 드라이버는 `RegionInfo.Region`을 **리소스 그룹**으로 사용합니다.
  따라서 Azure에서는 `region_name`에 **VM이 속한 리소스 그룹 이름**을 넣어야 합니다.

### 2) CSP에서 수집하는 것 (→ `csp` 섹션)

VM 리소스는 `GET /cspvm/{id}`로 조회합니다. cb-spider는 관리하지 않는(기존) VM도 CSP에 직접 질의해
정보를 돌려줍니다. 전체 ARM ID는 경로 인코딩 문제로 깨지므로 honeybee는 **VM 이름(리소스 ID의 마지막
세그먼트)** 을 넘깁니다. 이 **수집·저장은 `POST /import/infra`에서** 이뤄지며(등록/`refresh`는
연결 상태만 확인하고 저장하지 않음), 결과는 `SavedInfraInfo.csp_data`에 저장되어 `GET /.../infra`의
`csp` 섹션으로 노출됩니다:

```jsonc
"csp": {
  "provider": "AZURE", "region": "koreacentral", "zone": "1",
  "name": "vm-1", "id": "/subscriptions/.../virtualMachines/vm-1",
  "vm_spec": "Standard_D2s_v3", "image": "...", "platform": "LINUX/UNIX",
  "public_ip": "40.82.136.176", "private_ip": "10.0.0.4",
  "root_disk": { "type": "PremiumSSD", "size": 30 },
  "data_disks": [],
  "network": {
    "vpc": {
      "name": "vm-1-vnet", "cidr": "10.0.0.0/16",
      "subnets": [ { "name": "default", "cidr": "10.0.0.0/24", "zone": "" } ]
    },
    "subnet": "default",
    "security_groups": [ {
      "name": "vm-1-nsg",
      "rules": [ { "direction": "inbound", "protocol": "tcp", "from_port": "22", "to_port": "22", "cidr": "0.0.0.0/0" } ]
    } ]
  },
  "tags": {}
}
```

> **VPC/SG 상세 해석:** VM 조회는 VPC/SG의 **이름(IID)** 만 주므로, honeybee는
> `GET /allvpcinfo`·`GET /allsecuritygroupinfo`로 해당 연결의 **모든** VPC/SG를 상세 포함해 조회한 뒤
> VM의 `VpcIID`/`SecurityGroupIIds`와 **이름으로 매칭**해 `cidr`·`subnets`·`rules`를 채웁니다.
> 이 all-info 목록은 CSP를 live 질의하므로 cb-spider가 관리하지 않는(기존) VPC/SG도
> `OnlyCSPInfoList`로 상세가 반환됩니다(별도 등록 불필요).

### 3) 에이전트로 수집하는 것 (→ `compute`/`network.host`/소프트웨어 등)

ConnectionInfo에 SSH 접속 정보(`ip_address`/`user`/`password`\|`private_key`)가 있으면, honeybee는
SSH로 접속해 **에이전트(`cm-honeybee-agent`)를 설치·기동**하고, `import/*` 시 게스트 안에서
`http://localhost:8082/honeybee-agent/...`를 호출해 OS·커널·CPU·메모리·디스크·소프트웨어를 수집합니다.
이 결과는 `SavedInfraInfo.infra_data`에 저장됩니다.

### 4) 연결/에이전트 상태의 의미

| 필드 | CSP 타입에서의 의미 |
|------|----------------------|
| `connection_status` | **cb-spider가 해당 VM을 식별했는지**(CSP 도달성). SSH와 무관. |
| `agent_status` | **실제 SSH 접속 + 에이전트 설치 결과.** SSH 정보가 없으면 성공으로 위장하지 않고 `"no SSH access configured..."`로 실패 표기. |

### 5) 저장 분리 (덮어쓰기 없음)

`SavedInfraInfo`는 한 레코드에 두 칼럼을 둡니다.

| 칼럼 | 채우는 주체 | 노출 위치 |
|------|-------------|-----------|
| `csp_data` | cb-spider 수집(`import/infra` 시) | `infra.csp` |
| `infra_data` | 에이전트 수집(`import/infra` 시) | `infra.compute`/`infra.network.host` 등 |

- **등록 / `refresh`**: 연결 상태만 확인합니다(저장 없음). `connection_status`는 cb-spider가 리소스를
  식별하는지, `agent_status`는 실제 SSH+에이전트 설치 결과.
- **`POST /import/infra`**: 실제 수집·저장 단계. CSP 소스면 **csp_data(cb-spider)와 infra_data(에이전트)를
  함께 갱신**합니다(SSH 정보가 없는 CSP 소스는 csp_data만 갱신하고 성공 처리).
- 한쪽을 갱신해도 다른 칼럼은 보존되므로, **에이전트 import가 CSP 정보를 덮어쓰지 않고 함께 조회**됩니다.

### 6) 통합 조회 예시 (`GET /source_group/{sgId}/infra`)

SSH 정보를 함께 준 CSP(Azure) VM 소스의 실제 응답 예시입니다. `compute`/`network.host`는 **에이전트**가
게스트 내부에서 수집한 값, `csp`는 **cb-spider**가 클라우드에서 수집한 값입니다(반복되는 배열은 `…`로 축약).

```jsonc
{
  "servers": [
    {
      // ── 에이전트(SSH) 수집: 게스트 내부 ──
      "compute": {
        "os": {
          "os": { "pretty_name": "Ubuntu 24.04.4 LTS", "name": "Ubuntu", "version_id": "24.04",
                  "id": "ubuntu", "id_like": "debian" },
          "kernel": { "release": "6.17.0-1021-azure", "architecture": "x86_64" },
          "node": { "hostname": "ish-test", "machineid": "ac427abe-…", "timezone": "UTC" }
        },
        "compute_resource": {
          "cpu": { "vendor": "GenuineIntel", "model": "Intel(R) Xeon(R) Platinum 8171M CPU @ 2.60GHz",
                   "max_speed": 2100, "cache": 36608, "cpus": 1, "cores": 1, "threads": 2 },
          "memory": { "type": "Unknown", "size": 8142, "used": 362, "available": 7270 },
          "root_disk": { "name": "", "type": "", "size": 0 },
          "data_disk": [ { "name": "sdb1", "type": "HDD", "size": 15, "available": 14 }, … ]
        }
      },
      "network": {
        "host": {
          "network_interface": [
            { "interface": "eth0", "address": ["10.0.0.4/24", "fe80::…/64"],
              "gateway": "10.0.0.1", "mac_address": "60:45:bd:45:65:a5", "mtu": 1500 }, …
          ],
          "dns": { "dns_server": ["168.63.129.16"] },
          "route": [ { "interface": "eth0", "destination": "0.0.0.0", "next_hop": "10.0.0.1", … }, … ],
          "firewall_rule": []
        },
        "csp": { "name": "", "vpc": null, "nlb": null, "security_group": null }
      },
      "gpu": { "nvidia": [], "drm": [ { "driver_name": "hyperv_drm", … } ],
               "errors": ["NVIDIA: nvidia-smi command is not available"] },
      "storage": { "mount_point": { "mounted_information": null } },
      "haproxy": { "version": "", … },
      "minio": { "version": "", "errors": ["… MinIO process not found", …] },

      // ── cb-spider 수집: 클라우드 관점 (별도 저장, 덮어쓰이지 않음) ──
      "csp": {
        "provider": "AZURE", "region": "koreacentral", "zone": "1",
        "name": "ish-test",
        "id": "/subscriptions/…/resourceGroups/koreacentral/providers/Microsoft.Compute/virtualMachines/ish-test",
        "vm_spec": "Standard_D2s_v3", "image": "canonical:ubuntu-24_04-lts:ubuntu-pro:latest",
        "platform": "LINUX/UNIX", "public_ip": "40.82.136.176", "private_ip": "10.0.0.4",
        "root_disk": { "name": "Not visible in Azure", "type": "PremiumSSD", "size": 30 },
        "data_disks": [],
        "network": {
          "vpc": { "name": "ish-test-vnet", "cidr": "10.0.0.0/16",
                   "subnets": [ { "name": "default", "cidr": "10.0.0.0/24", "zone": "" } ] },
          "subnet": "default",
          "security_groups": [ { "name": "ish-test-nsg",
            "rules": [ { "direction": "inbound", "protocol": "tcp",
                         "from_port": "22", "to_port": "22", "cidr": "0.0.0.0/0" } ] } ]
        },
        "tags": {},
        "start_time": "2026-08-05T10:22:47Z"
      }
    }
  ]
}
```

> 위 예시에서 `network.host.csp`(에이전트가 채우지 못하는 스텁)는 `null`이고, CSP 네트워크는
> 최상위 `csp.network`에 상세히 담깁니다. `csp` 섹션은 SSH 없이 CSP만으로도 채워지며,
> `compute`/`network.host`는 SSH+에이전트가 있어야 채워집니다.

---

## SourceGroup

### `POST /source_group` — SourceGroup 등록
Body: `model.CreateSourceGroupReq`. 생성된 `model.SourceGroupRes`를 반환.

### `GET /source_group` — SourceGroup 목록
Query: `page`, `row`, `name`, `description` (모두 선택 필터).

### `GET /source_group/{sgId}` — SourceGroup 조회

### `PUT /source_group/{sgId}` — SourceGroup 수정
Body: `model.UpdateSourceGroupReq`.

### `DELETE /source_group/{sgId}` — SourceGroup 삭제

### `PUT /source_group/{sgId}/refresh` — 그룹 연결 상태 갱신
그룹 내 모든 ConnectionInfo의 도달성/상태를 갱신합니다.

### `POST /source_group/{sgId}/target` — TargetInfo 등록
Body: `model.RegisterTargetInfoReq` — cm-beetle을 통한 인프라 마이그레이션에서 돌려받은 target 정보.

---

## ConnectionInfo

### `POST /source_group/{sgId}/connection_info` — 생성
Body: `model.CreateConnectionInfoReq` (IP, SSH 포트, 사용자, 자격 증명 등).

### `GET /source_group/{sgId}/connection_info` — 목록
Query 필터: `page`, `row`, `name`, `description`, `ip_address`, `ssh_port`, `user`.

### `GET /source_group/{sgId}/connection_info/{connId}` — 조회
`GET /connection_info/{connId}`로 그룹 없이도 조회 가능.

### `PUT /source_group/{sgId}/connection_info/{connId}` — 수정
Body: `model.CreateConnectionInfoReq`.

### `DELETE /source_group/{sgId}/connection_info/{connId}` — 삭제

### `PUT /source_group/{sgId}/connection_info/{connId}/refresh` — 상태 갱신
`PUT /connection_info/{connId}/refresh`로 그룹 없이도 가능.

---

## CSP & Discovery

### `GET /csp` — 지원 CSP 목록
연결된 cb-spider가 지원하는 CSP 목록을 반환합니다.

```bash
curl http://localhost:8081/honeybee/csp
```

### `GET /csp/{name}` — CSP 메타데이터 조회
해당 CSP의 자격 증명 키/리전/기타 메타데이터를 반환합니다.

- **`name`은 대소문자 무시**(`aws` == `AWS`).
- **미지원 CSP**를 주면 400과 함께 **지원 목록을 안내**합니다:
  `"unsupported CSP: \"foo\". Supported CSPs: ALIBABA, AWS, AZURE, GCP, ..."`
- 응답의 `credentials[]`는 각 credential 키에 **예시값·설명**을 함께 제공(입력 폼 자동 구성용):

```bash
curl http://localhost:8081/honeybee/csp/azure
#  { "name":"AZURE",
#    "credential_keys":["ClientId","ClientSecret","TenantId","SubscriptionId"],
#    "credentials":[ {"key":"ClientId","example":"<AZURE_CLIENT_ID>","description":"Azure Client ID (clientId)"}, ... ],
#    "regions":["Region","Zone"], ... }
```

> `regions` 필드는 리전 "키 구조"(`Region`/`Zone`)일 뿐 실제 리전 목록이 아닙니다.
> 실제 리전 목록은 credential이 필요하므로 아래 `GET /source_group/{sgId}/region`을 쓰세요.

### `GET /source_group/{sgId}/region` — CSP 실제 리전/존 목록 (live)
`csp` SourceGroup의 **저장된 credential로 cb-spider에 live 질의**하여 그 CSP의 실제 리전 목록과
각 리전의 zone을 반환합니다(등록 후 사용). 등록 前 단계(`GET /csp/{name}`)에서는 credential이 없어
실제 리전을 못 주므로, 소스그룹 등록 후 이 API로 리전/zone 선택지를 채웁니다.

```bash
curl http://localhost:8081/honeybee/source_group/$SG/region
#  { "provider":"AZURE",
#    "regions":[ {"name":"koreacentral","display_name":"...","zones":["1","2","3"]}, ... ] }
```

> 여기서 얻은 `zones` 값이 ConnectionInfo의 `zone`(아래) 입력 후보가 됩니다.

### `GET /source_group/{sgId}/discover` — CSP 리소스 디스커버리
`type=csp` SourceGroup에 바인딩된 CSP 연결로 접근 가능한 VM / K8s 클러스터 / 오브젝트 스토리지 버킷을
나열합니다(UI에서 ConnectionInfo 선택지를 채울 때 사용).

| 쿼리 파라미터 | 필수 | 설명 |
|---------------|------|------|
| `resource_type` | 예 | `vm` \| `k8s` \| `object_storage` |

```bash
curl "http://localhost:8081/honeybee/source_group/$SG/discover?resource_type=k8s"
```

---

## 소스 정보 수집

각 데이터 타입마다 두 가지 범위(연결 1개 / 그룹 전체)가 있습니다.

```bash
# 노드 1개
curl $BASE/source_group/$SG/connection_info/$CONN/infra
# 그룹 전체
curl $BASE/source_group/$SG/infra
```

`software`, `kubernetes`, `helm`, `data`도 동일한 패턴을 따릅니다.

---

## 정제된 소스 모델

이 엔드포인트들은 다운스트림 마이그레이션 도구가 사용하는 정규화 모델을 반환합니다. 정제 모델은
`github.com/cloud-barista/cm-beetle/imdl/on-premise-model`입니다.

```bash
# 노드 1개
curl $BASE/source_group/$SG/connection_info/$CONN/infra/refined
# 그룹 전체
curl $BASE/source_group/$SG/infra/refined

# 소프트웨어
curl $BASE/source_group/$SG/connection_info/$CONN/software/refined
curl $BASE/source_group/$SG/software/refined
```

### 쿠버네티스 소스 모델

정제 인프라 모델은 소스 K8s 클러스터를 하나의 단위로 마이그레이션할 수 있도록 쿠버네티스 소스 정보를
포함합니다. 별도의 `/kubernetes/refined` 엔드포인트는 없으며, K8s 정보는 `/infra/refined` 응답에
함께 실립니다. 구현 위치는 `server/pkg/api/rest/controller/getRefined.go`이며 핵심은 다음과 같습니다.

- **모델 출처 변경:** 더 이상 사용하지 않는 `cloud-barista/cm-model/infra/on-premise-model`에서
  `cloud-barista/cm-beetle/imdl/on-premise-model`로 전환(커밋 `a2f2b05`).
- **필드명 변경:** `OnpremInfra.Servers → OnpremInfra.Nodes`, `ServerProperty → NodeProperty`.
- **신규 필드:** `OnpremInfra.K8sCluster`(타입 `K8sClusterProperty`)와 `NodeProperty.Role`
  (예: `control-plane`, `worker`, `standalone`).
- **K8s 클러스터 필드:** name, version, 그리고 참조 필드 PodCIDR, ServiceCIDR, CNIPlugin,
  NodePortRange (`buildK8sCluster`).
- **노드 조립:** 정제 노드는 쿠버네티스 수집 결과로부터 구성되어 호스트 수집 노드와 **machine ID** 기준
  으로 병합됩니다(`mergeK8sNodes`, `buildNodeFromK8s`; 커밋 `3c1e076`).

K8s 소스 그룹의 정제 인프라 응답은 대략 다음과 같습니다.

```json
{
  "network": { "ipv4Networks": {}, "ipv6Networks": {} },
  "nodes": [
    {
      "hostname": "k8s-master.example.com",
      "machineId": "8bb703c5d378420db97f6636cf454530",
      "role": "control-plane",
      "cpu":  { "architecture": "amd64", "threads": 8 },
      "memory": { "totalSize": 15 },
      "rootDisk": { "totalSize": 77 }
    }
  ],
  "k8sCluster": { "name": "", "version": "1.32.3", "cniPlugin": "calico" }
}
```

---

## Benchmark

### `GET /bench/{connId}` — 벤치마크 정보 조회

### `POST /bench/{connId}/run` — 벤치마크 실행
연결된 서버에 벤치마크 에이전트가 없으면 자동으로 설치한 뒤 벤치마크를 실행합니다.

| 쿼리 파라미터 | 기본값 | 설명 |
|---------------|--------|------|
| `types` | `cpus,cpum` | 콤마로 구분된 벤치마크 타입: `cpus, cpum, memR, memW, fioR, fioW, dbR, dbW`. |

```bash
curl -X POST "http://localhost:8081/honeybee/bench/$CONN/run?types=cpus,memR,fioW"
```

### `POST /bench/{connId}/stop` — 벤치마크 중지
