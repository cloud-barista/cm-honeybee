# CM-Honeybee Server API

**Server**(`cm-honeybee`)는 컨트롤 플레인입니다. **SourceGroup**과 **ConnectionInfo**를 관리하고,
원시 소스 데이터를 수집(에이전트 풀, SSH, 또는 CSP 소스의 경우 서버에 포함된 cb-spider 드라이버 경유)·저장하며, `cm-beetle`이
마이그레이션에 사용하는 **정제된 소스 모델**을 제공합니다.

| 항목 | 값 |
|------|-----|
| 모듈명 | `HONEYBEE` |
| Base path | `/honeybee` |
| 기본 포트 | `8081` |
| 의존성 | `cm-honeybee-agent` (소스 호스트의 루프백, 포트 자동), cb-spider 드라이버 (서버 바이너리에 포함, 별도 cb-spider 서버 불필요. `CBSPIDER_ROOT`/`CBLOG_ROOT` 필요, 아래 참고) |
| Swagger UI | `http://<host>:8081/honeybee/api/index.html` |
| 인증 | 없음 |

> 아래 모든 경로는 base path 기준 상대 경로입니다. 전체 URL 예시:
> `http://localhost:8081/honeybee/source_group`

> **cb-spider 드라이버 실행 환경:** CSP 조회는 서버에 링크된 cb-spider 드라이버가 CSP를 직접 호출합니다.
> 이전 버전의 `cm-honeybee.spider`(endpoint/username/password) 설정과 `HONEYBEE_SPIDER_*` 환경 변수는
> 더 이상 읽지 않습니다. 설정 파일에 `spider` 블록이 남아 있어도 무시될 뿐 기동은 실패하지 않습니다.
> 드라이버는 두 디렉터리를 읽습니다.
>
> - `CBSPIDER_ROOT`: `cloud-driver-libs/cloudos.yaml`·`cloudos_meta.yaml`·`region/*_region_meta.yaml`,
>   `conf/calllog_conf.yaml`. 드라이버가 `meta_db/`와 `log/`도 여기에 씁니다. 비어 있으면
>   `$CMHONEYBEE_ROOT/cb-spider`(없으면 `~/.cm-honeybee/cb-spider`)를 씁니다.
> - `CBLOG_ROOT`: cb-log 설정 `conf/log_conf.yaml`.
>
> Docker 이미지는 파일을 `/cb-spider`에 담고 두 변수를 `/cb-spider`로 지정합니다. 이미지의
> `/cb-spider/log`는 볼륨이 아니라서 컨테이너를 다시 만들면 드라이버 로그가 사라집니다.
> 로컬에서는 `cd server && make cb-spider-root`가 `go.mod`에 고정된 cb-spider 버전에서 파일을 복사하고,
> `make run`은 이를 먼저 실행한 뒤 두 변수를 지정해 바이너리를 띄웁니다.
> 바이너리를 직접 띄울 때 다음 경우에는 cb-log가 기동 중에 프로세스를 종료합니다.
> `CBLOG_ROOT`는 지정했는데 `$CBLOG_ROOT/conf/log_conf.yaml`이 없을 때, 그리고 `log_conf.yaml`이
> `$CBSPIDER_ROOT`를 참조하는데 그 값이 끝내 비어 있을 때(위 기본값도 정하지 못한 경우, 예: 홈 디렉터리 없음).
> `CBLOG_ROOT`를 지정하지 않으면 cb-log는 내장 기본 설정을 씁니다.

## 핵심 개념

- **SourceGroup** — 함께 마이그레이션할 소스 머신들의 논리적 그룹. 온프레미스 그룹과 `csp`(클라우드)
  그룹 등의 타입이 있습니다. SourceGroup은 등록된 *target* 정보(마이그레이션 후 cm-beetle이 돌려준
  결과)도 함께 가질 수 있습니다.
- **ConnectionInfo** - SourceGroup에 속한 개별 소스의 연결 정보. SSH 대상이면 IP/SSH 포트/사용자·자격
  증명을, 온프렘 k8s 클러스터면 `kubeconfig`를, CSP 리소스면 `resource_type`/`resource_id`를 담습니다.
  Server는 이를 사용해 호스트/에이전트/CSP에 접근하여 데이터를 수집합니다.
- **원시(Raw) vs 정제(Refined)** - `/infra`, `/software`, `/kubernetes`, `/helm`, `/data`, `/rdbms`,
  `/nlb`는 수집된 원시 데이터를 반환합니다. `/.../refined` 엔드포인트는 다운스트림에서 사용하는 정규화된 모델
  (`github.com/cloud-barista/cm-beetle/imdl/on-premise-model`)을 반환합니다.
  **정제 모델은 GPU를 노드의 `gpuCards[]`로 냅니다.** 물리 카드 하나가 항목 하나이고,
  모델이 섞인 노드도 카드별로 그대로 표현됩니다. 메모리는 드라이버가 보고한 값을 그대로
  넣습니다 - ECC가 켜진 카드는 표기 용량보다 작게 보고되는데, 그 차이를 여기서 보정하지 않고
  `eccEnabled`와 `memoryReservedGB`를 함께 실어 소비하는 쪽이 판단하게 합니다.
  온도·전력·클럭·프로세스 같은 텔레메트리와 `drm`, 그리고 **MIG 인스턴스 목록**은 정제 모델에
  담기지 않으므로, 그 값이 필요하면 `/.../infra`를 호출하세요.

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
| GET | `/source_group` | SourceGroup 목록 (페이징·필터 가능, `type` 필터). |
| GET | `/source_group/types` | SourceGroup 타입별 연결 타입과 입력 항목(등록 폼 구성용). |
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
| GET | `/source_group/{sgId}/connection_info` | ConnectionInfo 목록 (페이징·필터 가능, `resource_type` 필터). |
| GET | `/source_group/{sgId}/connection_info/{connId}` | ConnectionInfo 조회. |
| PUT | `/source_group/{sgId}/connection_info/{connId}` | ConnectionInfo 수정. |
| DELETE | `/source_group/{sgId}/connection_info/{connId}` | ConnectionInfo 삭제. |
| PUT | `/source_group/{sgId}/connection_info/{connId}/refresh` | 단일 연결 상태 갱신. |
| DELETE | `/source_group/{sgId}/connection_info/{connId}/agent` | **소스 호스트에 설치된 에이전트를 제거**(SSH로 uninstall 실행). |
| DELETE | `/source_group/{sgId}/agent` | **그룹 내 모든 호스트의 에이전트를 제거.** |
| GET | `/connection_info/{connId}` | sgId 없이 ConnectionInfo 직접 조회. |
| PUT | `/connection_info/{connId}/refresh` | sgId 없이 단일 연결 상태 직접 갱신. |

### CSP / Discovery
| 메서드 | 경로 | 설명 |
|--------|------|------|
| GET | `/csp` | 서버에 포함된 cb-spider 드라이버가 지원하는 CSP 목록과 CSP별 수집 가능한 `resource_type`. |
| GET | `/csp/{name}` | CSP 메타데이터(자격 증명 키+예시, 리전 키 등) 조회. 대소문자 무시, 미지원 시 지원목록 안내. |
| POST | `/csp/{name}/region` | 바디의 credential로 **그룹 없이** 실제 리전/존 목록 조회. credential은 저장하지 않음. |
| GET | `/source_group/{sgId}/region` | `csp` SourceGroup의 **실제 리전/존 목록**(저장 credential로 live 조회). |
| GET | `/source_group/{sgId}/discover` | `csp` SourceGroup의 VM / K8s 클러스터 / 오브젝트 스토리지 / NLB / 관리형 DB 디스커버리. |

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
| GET | `/source_group/{sgId}/connection_info/{connId}/rdbms` | 연결 1개의 DB 정보 (CSP 관리형 DB 또는 VM에 설치된 DB). |
| GET | `/source_group/{sgId}/rdbms` | 그룹 전체의 DB 정보. |
| GET | `/source_group/{sgId}/connection_info/{connId}/nlb` | csp `nlb` 연결 1개의 NLB 정보. |
| GET | `/source_group/{sgId}/nlb` | 그룹 전체의 csp NLB 정보 (마지막 import가 성공한 연결만). |

### 정제된 소스 모델
| 메서드 | 경로 | 설명 |
|--------|------|------|
| GET | `/source_group/{sgId}/connection_info/{connId}/infra/refined` | 노드 1개의 정제 인프라. |
| GET | `/source_group/{sgId}/infra/refined` | 그룹 전체의 정제 인프라. |
| GET | `/source_group/{sgId}/connection_info/{connId}/software/refined` | 노드 1개의 정제 소프트웨어. |
| GET | `/source_group/{sgId}/software/refined` | 그룹 전체의 정제 소프트웨어. |
| GET | `/source_group/{sgId}/connection_info/{connId}/bucket/refined` | csp 버킷 1개의 정제 오브젝트 스토리지. `data/refined`와 같음. |
| GET | `/source_group/{sgId}/bucket/refined` | 그룹 전체 csp 버킷의 정제 오브젝트 스토리지. `data/refined`와 같음. |
| GET | `/source_group/{sgId}/connection_info/{connId}/data/refined` | `bucket/refined`의 이전 이름. 호환용으로 남아 있음. |
| GET | `/source_group/{sgId}/data/refined` | `bucket/refined`의 이전 이름. 호환용으로 남아 있음. |
| GET | `/source_group/{sgId}/connection_info/{connId}/rdbms/refined` | 연결 1개의 정제 DB. |
| GET | `/source_group/{sgId}/rdbms/refined` | 그룹 전체의 정제 DB. |
| GET | `/source_group/{sgId}/connection_info/{connId}/k8s/refined` | 연결 1개의 정제 K8s 클러스터 (`infra/refined`의 `k8sCluster`). |
| GET | `/source_group/{sgId}/k8s/refined` | 그룹의 정제 K8s 클러스터. |
| GET | `/source_group/{sgId}/connection_info/{connId}/nlb/refined` | 연결 1개의 정제 NLB 목록 (`infra/refined`의 `nlbs`). |
| GET | `/source_group/{sgId}/nlb/refined` | 그룹의 정제 NLB 목록 (HAProxy NLB + csp NLB). |

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
| POST | `/source_group/{sgId}/import/all` | onprem·csp 그룹의 모든 연결을 각 타입에 맞게 수집. |
| POST | `/source_group/{sgId}/import/bucket` | csp 그룹의 `object_storage` 연결만 CSP에서 수집. |
| POST | `/source_group/{sgId}/import/rdbms` | csp 그룹의 `rdbms` 연결만 CSP에서 수집. |
| POST | `/source_group/{sgId}/import/k8s` | csp 그룹의 `k8s`(관리형 클러스터) 연결만 CSP에서 수집. |
| POST | `/source_group/{sgId}/import/nlb` | csp 그룹의 `nlb` 연결만 CSP에서 수집. |

> `fs`·`db`·`minio` 그룹은 각자의 경로를 씁니다: `import/fs`·`import/db`·`import/objectstorage`
> (수집), `fs`·`db`·`objectstorage` (원시), `fs/refined`·`db/refined`·`objectstorage/refined` (정제).
> 연결 1개 단위는 같은 경로가 `connection_info/{connId}/` 아래에 있습니다.
> 종류별 수집·정제 API의 동작은 아래 [자원 종류별 수집·정제](#자원-종류별-수집정제) 절을 참고하세요.

### Benchmark
| 메서드 | 경로 | 설명 |
|--------|------|------|
| GET | `/bench/{connId}` | 연결의 벤치마크 정보 조회. |
| POST | `/bench/{connId}/run` | 벤치마크 실행 (벤치마크 에이전트가 없으면 자동 설치). |
| POST | `/bench/{connId}/stop` | 실행 중인 벤치마크 중지. |

---

## SourceGroup 타입 (onprem / csp)

모든 SourceGroup은 `type` 필드로 수집 방식을 구분합니다.

| type | 설명 | 연결 정보 입력 방식 |
|------|------|---------------------|
| `onprem` | 온프레미스 소스. SSH로 직접 접속하거나, kubeconfig로 온프렘 k8s 클러스터를 등록. | ConnectionInfo의 `resource_type`에 따라 갈립니다(아래 표). |
| `csp` | cb-spider 기반 클라우드 소스. credential/region으로 CSP에서 VM·관리형 K8s·버킷·관리형 DB·NLB 메타데이터를 수집. VM이면 **SSH 접속 정보를 추가로 주면 게스트 내부까지 에이전트로 수집**. | SourceGroup에 `provider_name`/`region_name`/`credential[]`, ConnectionInfo에 `resource_type`/`resource_id` (+ 선택적으로 `zone`, `vm`이면 `ip_address`/`ssh_port`/`user`/`password`\|`private_key`) 입력. `vm`이 아닌 연결에 SSH 필드를 보내면 400. |

- 타입별로 받는 필드는 `GET /source_group/types`가 알려줍니다(아래 [SourceGroup](#sourcegroup) 절).

- **`onprem`, `ssh`, 빈 값은 모두 온프레미스로 같게 취급**됩니다(`server/common/csp.go`의
  `IsOnpremType`). 기존에 `"type":"ssh"`로 등록했거나 `type`을 생략한 클라이언트 페이로드는
  수정 없이 그대로 동작합니다(하위 호환).
- 다만 **`type`을 생략하면 저장·응답되는 값은 `ssh`** 입니다(`controller/sourceGroup.go:120-121`).
  `onprem`으로 저장하고 싶으면 명시적으로 보내야 합니다.

- **CSP 타입은 두 가지 정보원을 함께 씁니다.**
  - **CSP(cb-spider) 수집** - VM 스펙·이미지·리전·공인/사설 IP·디스크·태그·VPC/서브넷/SG 이름 등
    *클라우드 바깥에서 보이는* 정보. `csp` 섹션으로 노출됩니다.
  - **에이전트(SSH) 수집** - OS·커널·CPU·메모리·디스크 사용량·소프트웨어 등 *게스트 내부* 정보.
    ConnectionInfo에 SSH 접속 정보를 준 경우에만 수집됩니다(에이전트를 설치해 조회).
  - 두 정보는 **분리 저장**되어 서로 덮어쓰지 않습니다(아래 "CSP 수집 동작 상세" 참고).

### 온프레미스 ConnectionInfo의 `resource_type`

온프레미스 그룹도 `resource_type`으로 연결의 성격을 나눕니다. 허용 값은 `vm`과 `k8s` 둘뿐이며,
다른 값을 주면 `"resource_type for on-prem must be one of vm | k8s"` 오류로 거부됩니다.

| resource_type | 필수 입력 | 수집 방식 |
|---------------|-----------|-----------|
| `vm` (기본값, 생략 가능) | `ip_address`, `ssh_port`, `user`, `password`\|`private_key` | SSH로 접속해 에이전트를 설치하고 조회. |
| `k8s` | `kubeconfig` | **SSH 호스트가 없습니다.** 등록 시 kubeconfig의 구조만 검증하고 저장하며, 실제 수집은 다운스트림 마이그레이션 모듈이 이 kubeconfig로 수행합니다. |

- `kubeconfig`는 등록·수정 시 **구조를 파싱해 검증**합니다. 비었거나 형식이 깨졌으면 등록이 거부됩니다.
- 온프렘 `k8s` 커넥션의 상태 필드는 다음으로 고정됩니다. 에이전트 실패로 보이지만 **정상 상태**입니다.
  - `connection_status`: `success`
  - `agent_status`: `failed`, `agent_failed_message`:
    `"agent-based collection is not applicable to on-prem k8s connections"`
- `POST /import/*`는 에이전트 경로라 온프렘 `k8s`에는 해당하지 않습니다.
  - **단건 호출**(`.../connection_info/{connId}/import/*`)은 위와 같은 문구로 거부합니다.
  - **그룹 호출**(`/source_group/{sgId}/import/*`)은 그 커넥션만 건너뛰고 나머지를 수집합니다.
    응답 배열에 그 커넥션의 항목이 없는 것으로 확인합니다.

---

## 비밀 정보 저장 (OpenBao 필수)

커넥션 시크릿(비밀번호/개인키/kubeconfig)과 CSP credential은 **OpenBao**(KV v2)에만 저장됩니다.
SQLite에는 비밀 정보를 저장하지 않습니다.

| 종류 | OpenBao 경로 (KV v2, mount `secret`) |
|------|--------------|
| CSP credential | `secret/honeybee/csp/{sgId}` |
| 커넥션 시크릿(password/private_key/kubeconfig) | `secret/honeybee/ssh/{connId}` |

> 경로의 `ssh` 세그먼트는 kubeconfig 지원 이전에 쓰던 이름 그대로입니다. 기존에 저장된 항목을
> 계속 읽을 수 있도록 바꾸지 않았습니다.

- **자립형(self-managed)**: cm-honeybee가 전용 OpenBao의 init/unseal/KV-enable을 **스스로 수행**하고,
  unseal key와 root token은 `honeybee.key`로 **RSA 암호화해 DB(`open_bao_inits`)에 저장**합니다.
  init 사이드카·수동 unseal·평문 열쇠 파일이 없으며, 재부팅/볼륨 유실에도 스스로 복구합니다.
- **설정**: `cm-honeybee.yaml`의 `openbao.address`(또는 `HONEYBEE_VAULT_ADDR`)만 지정하면 됩니다
  (토큰·언실 키·사이드카 설정 불필요). 값은 `${VAR}`로 주입 가능.
- **미설정 시**: 비밀 정보가 필요한 작업(자격증명/SSH 접속정보를 가진 소스그룹·커넥션 생성)은
  `"OpenBao is required ..."` 오류로 실패합니다. (비밀 정보가 없는 조회 등은 정상 동작)
- CSP credential 키 이름은 cb-tumblebug의
  [template.credentials.yaml](https://github.com/cloud-barista/cb-tumblebug/blob/main/init/template.credentials.yaml)
  규약(= cb-spider `credentialcsp`)을 따릅니다. 예: Azure `clientId/clientSecret/tenantId/subscriptionId`,
  AWS `aws_access_key_id/aws_secret_access_key`. (`GET /csp/{name}`가 키+예시 제공)
- 구조·부팅 흐름·복구 시나리오·프로덕션(KMS auto-unseal) 등 자세한 내용은
  [`docs/openbao.md`](openbao.md) 참고. OpenBao 포함 전체 컨테이너 구성은
  [`server/docker-compose.yaml`](../server/docker-compose.yaml).

---

## 전형적인 워크플로우 (온프레미스 SSH 타입, 권장)

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

CSP별로 credential 입력 항목이 다른 문제는 **cb-spider 드라이버 메타정보가 CSP마다 필요한 credential 키 목록을 알려주는
방식**으로 해결합니다. 클라이언트는 이 키 목록으로 입력 폼을 동적으로 구성하고, 값은 제네릭
`credential: [{key, value}]` 배열로 제출합니다(CSP별 하드코딩 불필요).

> **credential 보관 정책:** credential은 **honeybee가 암호화하여 보관**하며, 다른 곳에 등록하지 않습니다.
> 조회가 필요한 시점에만 honeybee가 credential을 메모리에서 CSP 드라이버에 넘겨 조회합니다.
> 디스크에 따로 쓰지 않으며, 영구 `ConnectionName` 바인딩도 두지 않습니다.

```bash
BASE=http://localhost:8081/honeybee

# 1. 지원 CSP 목록 조회
curl -s $BASE/csp | jq
#   → { "csp": ["aws", "gcp", "azure", ...] }   이름은 소문자로 내려옵니다

# 2. 선택한 CSP의 credential 키 / 리전 키 조회 (CSP마다 다름)
curl -s $BASE/csp/azure | jq
#   → { "name":"azure",
#       "credential_keys":["clientId","clientSecret","tenantId","subscriptionId"],
#       "region_keys":["Region","Zone"], "default_region":"koreacentral" }
#   클라이언트는 credential_keys로 입력 폼을 동적 렌더링.
#   키 표기는 CSP마다 다릅니다 (AWS=aws_access_key_id 소문자, Azure=clientId camelCase).
#   region_keys는 "리전을 어떤 키로 표현하는가"일 뿐 리전 목록이 아닙니다(아래 3-1 참고).

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
          {"key":"clientId","value":"$AZURE_CLIENT_ID"},
          {"key":"clientSecret","value":"$AZURE_CLIENT_SECRET"},
          {"key":"tenantId","value":"$AZURE_TENANT_ID"},
          {"key":"subscriptionId","value":"$AZURE_SUBSCRIPTION_ID"}
        ],
        "connection_info":[{
          "name":"vm-1",
          "resource_type":"vm",
          "resource_id":"/subscriptions/<sub>/resourceGroups/<rg>/providers/Microsoft.Compute/virtualMachines/<vm>",
          "ip_address":"40.82.136.176", "user":"ubuntu", "password":"...", "ssh_port":"22"
        }]
      }' | jq -r '.id')
#   등록 시 honeybee는 상태만 확인합니다: (1) CSP 드라이버가 VM을 식별하는지(connection_status),
#   (2) SSH 정보가 있으면 게스트에 에이전트 설치(agent_status). 이 단계에선 데이터를 저장하지 않음.
#   응답의 connection_info_status_count로 connection/agent 성공 수를 확인.

# 4. 수집 + 저장 (csp_data + infra_data) — import 단계에서 수행
curl -s -X POST $BASE/source_group/$SG/import/infra      # CSP 메타 + (SSH 있으면)게스트 내부
curl -s -X POST $BASE/source_group/$SG/import/software    # 게스트 소프트웨어(SSH 필요)
#    또는 위 둘과 버킷·DB·K8s·NLB 연결까지 한 번에:
curl -s -X POST $BASE/source_group/$SG/import/all

# 5. 통합 조회: compute/network(에이전트) + csp(CSP 드라이버)가 함께 반환됨
curl -s $BASE/source_group/$SG/infra | jq '.servers[0] | {compute, csp}'

# 6. 정제 소스 모델(cm-beetle용) / target 등록은 SSH 흐름과 동일
curl -s $BASE/source_group/$SG/infra/refined | jq
```

> **디스커버리(선택):** resource_id를 모르면 `GET /source_group/{sgId}/discover?resource_type=vm|k8s|object_storage|nlb|rdbms`로
> 해당 CSP의 리소스 목록을 조회해 `resource_id`를 얻을 수 있습니다(이 호출도 저장된 credential로 CSP를 직접 조회).

---

## CSP 수집 동작 상세

CSP 타입 소스에서 등록/갱신(`POST /source_group`, `PUT .../refresh`) 시 honeybee가 하는 일과,
정보가 어디서 수집되어 어디에 저장·노출되는지의 전체 과정입니다.

### 1) CSP 연결 (credential/region/zone)

honeybee는 조회 때마다 SourceGroup의 provider·region과 OpenBao에 저장된 credential로 접속 정보를 만들어
cb-spider 드라이버에 바로 넘깁니다. credential/region/connection을 어디에도 등록하지 않습니다.

- **Zone precedence**: ConnectionInfo의 `zone`(명시) > `region_name`에 `"<region>/<zone>"`로 임베드 >
  빈 값. 즉 `region_name`은 `"koreacentral"` 또는 `"koreacentral/2"`를, ConnectionInfo는
  `"zone":"2"`를 지원하며, 둘 다 있으면 ConnectionInfo의 `zone`이 우선합니다.
  선택 가능한 zone 목록은 `GET /source_group/{sgId}/region`의 각 리전 `zones`에서 확인합니다.
- **Azure 주의:** cb-spider Azure 드라이버는 `RegionInfo.Region`을 **리소스 그룹**으로 사용합니다.
  따라서 Azure에서는 `region_name`에 **VM이 속한 리소스 그룹 이름**을 넣어야 합니다.

### 2) CSP에서 수집하는 것

**연결 1개의 수집·저장은 `POST /import/infra` 하나가 담당합니다.** 리소스 종류에 관계없이 이
엔드포인트를 부르며, `resource_type`에 따라 저장 위치와 조회 경로가 갈립니다. 그룹 단위로는
`import/infra`(모든 타입), `import/all`, 또는 종류별 `import/bucket`·`import/rdbms`·`import/k8s`·`import/nlb`를
씁니다([자원 종류별 수집·정제](#자원-종류별-수집정제)).

| resource_type | 드라이버 호출 | 저장 위치 | 조회 경로 |
|---------------|----------------|-----------|-----------|
| `vm` | VM 핸들러 `GetVM` (CSP 리소스 ID) | `SavedInfraInfo.csp_data` | `GET /.../infra`의 `csp` 섹션 |
| `k8s` | 클러스터 핸들러 `ListCluster` 후 ID 매칭 | `SavedKubernetesInfo` | `GET /.../kubernetes` (`csp` 섹션에 노드 그룹) |
| `object_storage` | S3 버킷 목록 후 이름 매칭 + 버킷 리전 판정(아래 표), 버킷 리전의 엔드포인트에서 상세 조회 | `SavedDataInfo` (버킷 리전을 `region`으로 함께 저장) | `GET /.../data` |
| `rdbms` | RDBMS 핸들러 목록 후 ID 매칭 | `SavedRDBMSInfo` | `GET /.../rdbms` |
| `nlb` | NLB 핸들러 `GetNLB`, 맞지 않으면 목록 후 ID 매칭 | `SavedNLBInfo` | `GET /.../nlb` |

- **`k8s`**: 관리형 클러스터의 이름(CSP의 이름, 없으면 system ID)·버전·상태·생성 시각·노드 그룹(스펙,
  이미지, 루트 디스크, desired/min/max 크기, 오토스케일링)을 저장합니다. 노드와 워크로드는 수집하지
  않습니다(kubeconfig가 아니라 CSP API로 읽기 때문). Pod/Service CIDR은 cb-spider가 주지 않아 비어 있습니다.
- **`nlb`**: 리스너(프로토콜, IP, 포트, DNS 이름), 백엔드 VM 그룹(프로토콜, 포트, VM 목록), 헬스 체크를
  저장합니다. `type`은 `PUBLIC`·`INTERNAL`, 드라이버가 알려주지 않으면 빈 값입니다.
  AWS의 cb-spider 드라이버는 대상 그룹을 **NLB 자신의 이름**으로 찾습니다(cb-spider가 NLB를 만들 때 대상 그룹에
  붙이는 이름). cb-spider 밖에서 만든 NLB의 대상 그룹 이름이 NLB 이름과 다르면 드라이버가 대상 그룹을 못 찾고
  이를 에러로 내지 않아, import는 성공하고 `vm_group`은 `{"protocol": "", "port": "", "vms": []}`,
  `health_checker`는 0으로 저장됩니다. 이런 레코드는 **`vm_group_unknown: true`**가 되고 서버 로그에 경고가 남습니다.
  대상 그룹을 읽었는데 대상이 0대면 프로토콜·포트가 있고 `vms`만 비며 `vm_group_unknown: false`라서 둘이 구별됩니다.
  연결 1개와 그룹의 `GET .../nlb` 응답, 디스커버리의 `extra.vm_group_unknown`에 이 값이 나옵니다. 이 필드가 생기기
  전에 저장된 레코드는 비어 있는 `vm_group`을 보고 같은 값을 채워 돌려줍니다. 정제 NLB(`nlbs`)에는 담을 자리가
  없어 이런 NLB는 백엔드 서버가 없는 것으로 나옵니다.
- **`object_storage`**: 버킷 설정과, `count_objects=false`가 아니면 객체 수·전체 크기를 저장합니다.

> **주의:** CSP 소스의 k8s·오브젝트 스토리지 정보를 채우려면 `POST /import/kubernetes`나
> `POST /import/data`가 아니라 **`POST /import/infra`**(또는 그룹의 종류별 import)를 불러야 합니다.
> `import/kubernetes`와 `import/data`는 게스트 안의 에이전트에 SSH로 붙는 경로라 CSP 리소스에는
> 해당하지 않으며, 그런 연결 하나에 부르면 거부되고 그룹으로 부르면 그 연결을 건너뜁니다.

#### 버킷 리전

SourceGroup은 리전 하나를 나타내지만, 대부분 CSP의 버킷 목록은 계정 전체를 돌려줍니다. 그래서 CSP가
버킷 리전을 알려주는 경우에만 리전을 판정하고, 판정한 리전이 그룹 리전과 다른 버킷은 아래처럼 다룹니다.
리전을 판정하지 못하면 리전을 비워 두고 버킷을 리전 때문에 거부하지 않습니다.

| Provider | 리전 판정 | 방법, 또는 못 하는 이유 |
|----------|-----------|--------------------------|
| AWS | 함 | 그룹 리전 엔드포인트에 버킷 `HEAD`. S3는 그 리전 버킷만 그 엔드포인트에서 응답하고, 다른 리전 버킷이면 `x-amz-bucket-region`에 실제 리전을 담아 돌려줍니다. |
| Azure | 못 함 | 컨테이너에는 자체 리전이 없고, 스토리지 계정의 location은 계정 키로 쓰는 Blob API가 아니라 Azure Resource Manager 속성입니다. |
| GCP | 못 함 | `?location`이 리전만큼 자주 멀티·듀얼 리전(`US`, `ASIA`, `NAM4`)을 돌려주고, 표기도 그룹 리전과 다릅니다. |
| IBM | 못 함 | location constraint가 리전이 아니라 스토리지 클래스를 포함합니다(`us-south-standard`). |
| Alibaba, Tencent, NCP, NHN | 못 함 | S3 호환 API의 location 표기가 각자 다르고, 그룹이 쓰는 리전 이름과 맞춰 검증하지 않았습니다. |
| KT, OpenStack | 못 함 | 계정 전체에 엔드포인트가 하나라 비교할 리전이 없습니다. |

- **디스커버리**(`discover?resource_type=object_storage`): 판정되는 CSP는 그룹 리전의 버킷만 돌려주고
  응답 `region`을 채웁니다. 판정이 실패한 버킷은 다른 리전이라고 단정할 수 없어서 목록에 남기고
  `extra.region_error`에 사유를 답니다. 판정 못 하는 CSP는 모든 버킷을 리전 없이 돌려줍니다.
- **등록·수정**: 다른 리전으로 판정된 버킷은 400으로 거부합니다. 리전을 모르면 거부하지 않습니다.
- **정제**: 연결 1개의 `bucket/refined`(`data/refined`)는 버킷이 발견된 리전을 `sourceCloud.region`으로,
  모르면 그룹 리전을 씁니다. 그룹 `bucket/refined`(`data/refined`)는 모델에 리전이 하나뿐이라 그룹 리전을
  쓰고, **그룹 리전과 다른 리전으로 저장된 버킷은 결과에서 뺍니다**(등록·import가 거부하기 전에 저장된
  기록에 남아 있을 수 있음). 그룹에 리전이 없으면 모든 버킷을 넣습니다.

아래는 `vm`의 경우입니다. VM 리소스는 드라이버의 `GetVM`으로 CSP에 직접 질의하므로 cb-spider로 만들지 않은(기존) VM도
정보를 돌려줍니다. 드라이버는 VM을 이름으로 식별하므로 honeybee는 전체 ARM ID 대신 **VM 이름(리소스 ID의 마지막
세그먼트)** 을 넘깁니다. 이 **수집·저장은 `POST /import/infra`에서** 이뤄지며(등록/`refresh`는
연결 상태만 확인하고 저장하지 않음), 결과는 `SavedInfraInfo.csp_data`에 저장되어 `GET /.../infra`의
`csp` 섹션으로 노출됩니다:

```jsonc
"csp": {
  "provider": "azure", "region": "koreacentral", "zone": "1",
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
> 드라이버의 `ListVPC`·`ListSecurity`로 해당 연결의 **모든** VPC/SG를 상세 포함해 조회한 뒤
> VM의 `VpcIID`/`SecurityGroupIIds`와 **이름으로 매칭**해 `cidr`·`subnets`·`rules`를 채웁니다.
> 이 목록은 CSP를 live 질의하므로 cb-spider로 만들지 않은(기존) VPC/SG도 상세가 반환됩니다(별도 등록 불필요).

### 3) 에이전트로 수집하는 것 (→ `compute`/`network.host`/소프트웨어 등)

ConnectionInfo에 SSH 접속 정보(`ip_address`/`user`/`password`\|`private_key`)가 있으면, honeybee는
SSH로 접속해 **에이전트(`cm-honeybee-agent`)를 설치·기동**하고, `import/*` 시 게스트 안에서
`http://localhost:<port>/honeybee-agent/...`를 호출해 OS·커널·CPU·메모리·디스크·소프트웨어를 수집합니다.
에이전트는 기동할 때마다 커널이 고른 빈 포트에 루프백으로 리슨하고 그 번호를
`/etc/cloud-migrator/cm-honeybee-agent/port`에 남기므로, honeybee는 요청 직전에 SSH로 이 파일을
읽어 포트를 알아냅니다. 파일이 없으면(구버전 에이전트) `cm-honeybee.agent.port` 설정값을 씁니다.
이 결과는 `SavedInfraInfo.infra_data`에 저장됩니다.

### 4) 연결/에이전트 상태의 의미

| 필드 | CSP 타입에서의 의미 |
|------|----------------------|
| `connection_status` | 등록·`refresh`에서는 **CSP 드라이버가 해당 리소스를 식별했는지**(CSP 도달성), `import`에서는 **CSP 수집이 성공했는지**. 찾을 수는 있어도 읽지 못하면 `failed`이고 사유는 `connection_failed_message`. SSH와 무관. |
| `agent_status` | **실제 SSH 접속 + 에이전트 설치 결과.** SSH 정보가 없으면 성공으로 위장하지 않고 `"no SSH access configured..."`로 실패 표기. |

### 5) 저장 분리 (덮어쓰기 없음)

아래는 `resource_type: vm`인 CSP 소스 얘기입니다. `k8s`/`object_storage`/`rdbms`/`nlb`는 `SavedInfraInfo`가 아니라
각각 `SavedKubernetesInfo`/`SavedDataInfo`/`SavedRDBMSInfo`/`SavedNLBInfo`에 통째로 쓰이므로 칼럼이 갈리지 않습니다(위 2) 표 참고).

`SavedInfraInfo`는 한 레코드에 두 칼럼을 둡니다.

| 칼럼 | 채우는 주체 | 노출 위치 |
|------|-------------|-----------|
| `csp_data` | CSP 드라이버 수집(`import/infra` 시) | `infra.csp` |
| `infra_data` | 에이전트 수집(`import/infra` 시) | `infra.compute`/`infra.network.host` 등 |

- **등록 / `refresh`**: 연결 상태만 확인합니다(저장 없음). `connection_status`는 CSP 드라이버가 리소스를
  식별하는지, `agent_status`는 실제 SSH+에이전트 설치 결과.
- **`POST /import/infra`**: 실제 수집·저장 단계. CSP 소스면 **csp_data(CSP 드라이버)와 infra_data(에이전트)를
  함께 갱신**합니다(SSH 정보가 없는 CSP 소스는 csp_data만 갱신하고 성공 처리).
  `connection_status`·`connection_failed_message`도 이 수집 결과로 갱신합니다.

#### 수집 실패는 `failed`로 남김

아무것도 읽지 못한 수집을 `success`로 남기지 않습니다.

- **버킷**: 상세 항목을 하나도 읽지 못하면 저장은 하되 `status: failed`와 오류 목록을 남기고 import는
  400을 돌려줍니다. 일부 항목만 실패하면 성공이고 실패한 항목은 저장된 오류 목록에 남습니다. 어느 버킷에서나
  실패하는 항목(Azure의 versioning)은 계산에서 뺍니다.
- **CSP 재수집 실패**: `k8s`·`object_storage`·`rdbms`·`nlb` 연결의 CSP 수집이 실패하면 그 자원 테이블
  (`SavedKubernetesInfo`·`SavedDataInfo`·`SavedRDBMSInfo`·`SavedNLBInfo`)의 레코드를 `failed`로 바꿉니다.
  기록이 없으면 `failed` 레코드를 만듭니다. 원시 `GET`은 마지막으로 수집한 데이터를 그대로 보여주고,
  정제 API는 그 레코드를 쓰지 않습니다. `vm`은 `SavedInfraInfo`에 같은 방식으로 남습니다.
- csp 리소스(`vm` 아닌 것, 또는 `ip_address` 없는 `vm`)에 에이전트 import(`software`·`kubernetes`·`helm`·`data`)를
  부르면 레코드를 건드리기 전에 거부하므로, CSP 수집이 남긴 레코드를 `failed`로 덮어쓰지 않습니다.
- 한쪽을 갱신해도 다른 칼럼은 보존되므로, **에이전트 import가 CSP 정보를 덮어쓰지 않고 함께 조회**됩니다.

### 6) 통합 조회 예시 (`GET /source_group/{sgId}/infra`)

SSH 정보를 함께 준 CSP(Azure) VM 소스의 실제 응답 예시입니다. `compute`/`network.host`는 **에이전트**가
게스트 내부에서 수집한 값, `csp`는 **CSP 드라이버**가 클라우드에서 수집한 값입니다(반복되는 배열은 `…`로 축약).

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
      // nvidia-smi 가 없는 소스라 nvidia_smi_schema 키가 아예 없다 (omitempty).
      "gpu": { "nvidia": [], "amd": [], "drm": [ { "card": "card0", "driver_name": "hyperv_drm", … } ],
               "errors": ["NVIDIA: nvidia-smi command is not available"] },
      "storage": { "mount_point": { "mounted_information": null } },
      "haproxy": { "version": "", … },
      "minio": { "version": "", "errors": ["… MinIO process not found", …] },

      // ── CSP 드라이버 수집: 클라우드 관점 (별도 저장, 덮어쓰이지 않음) ──
      "csp": {
        "provider": "azure", "region": "koreacentral", "zone": "1",
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
Query: `page`, `row`, `name`, `description`, `type` (모두 선택 필터).
`type`은 `onprem`·`csp`·`fs`·`db`·`minio` 중 하나이고, 그 밖의 값은 400입니다. `type=onprem`은
`ssh`로 저장됐거나 `type` 없이 저장된 그룹도 함께 찾습니다.

### `GET /source_group/types` - SourceGroup 타입 설명
타입마다 그룹이 받는 필드(`group_fields`), 담을 수 있는 연결 타입(`connection_types`)과 연결 타입별
입력 항목(`auth_fields`, `fields`)을 돌려줍니다. 각 항목에는 `name`, `required`, `secret`(OpenBao에 보관,
폼에서 가려야 함), `enum`, `default`, `required_when`, `one_of`, `values_source`, `credential_source`가
붙습니다. 클라이언트는 이것으로 등록 폼을 만들 수 있습니다. `resource_type_field`는 `required`(csp),
`optional`(onprem, 기본 `vm`), `none`(fs·db·minio, 연결 타입 이름이 그룹 타입)입니다.
CSP credential 키는 `GET /csp/{name}`이 알려줍니다.

### 그룹 응답의 `connection_count_by_type`·`capabilities`
등록·조회·수정·목록의 그룹 응답에 다음이 붙습니다.

- `connection_count_by_type`: 연결 타입별 연결 수. onprem·csp 그룹은 `resource_type`
  (`resource_type` 없이 저장된 온프레미스 연결은 `vm`), fs·db·minio 그룹은 그룹 타입으로 셉니다.
- `capabilities`: 지금 가진 연결로 그룹이 돌릴 수 있는 수집. `isCollectInfra`, `isCollectSoftware`,
  `isCollectKubernetes`, `isCollectObjectStorage`, `isCollectDb`, `isCollectFs`, `isCollectNlb`.
  수집이 다룰 연결이 없으면 `false`이므로 빈 그룹은 모두 `false`입니다.

### `GET /source_group/{sgId}` — SourceGroup 조회

### `PUT /source_group/{sgId}` — SourceGroup 수정
Body: `model.UpdateSourceGroupReq`.

### `DELETE /source_group/{sgId}` — SourceGroup 삭제
그룹의 모든 연결과, 각 연결에 import가 저장한 기록(infra, software, kubernetes, helm, data, rdbms, nlb)을
함께 지웁니다.

### `PUT /source_group/{sgId}/refresh` — 그룹 연결 상태 갱신
그룹 내 모든 ConnectionInfo의 도달성/상태를 갱신합니다.

### `POST /source_group/{sgId}/target` — TargetInfo 등록
Body: `model.RegisterTargetInfoReq` — cm-beetle을 통한 인프라 마이그레이션에서 돌려받은 target 정보.

---

## ConnectionInfo

### `POST /source_group/{sgId}/connection_info` — 생성
Body: `model.CreateConnectionInfoReq`. 필수 필드는 그룹의 `type`과 이 요청의 `resource_type`에 따라
갈립니다 - [SourceGroup 타입](#sourcegroup-타입-onprem--csp) 참고.

| 그룹 type | resource_type | 필수 필드 |
|-----------|---------------|-----------|
| `onprem` | `vm`(생략 가능) | `name`, `ip_address`, `ssh_port`, `user`, `password`\|`private_key` |
| `onprem` | `k8s` | `name`, `kubeconfig` |
| `csp` | `vm` | `name`, `resource_type`, `resource_id` (+ 선택 `zone`, SSH 접속 정보) |
| `csp` | `k8s` \| `object_storage` \| `rdbms` \| `nlb` | `name`, `resource_type`, `resource_id` (+ 선택 `zone`). SSH 필드를 보내면 400 |

응답의 `user`/`password`/`private_key`/`kubeconfig`는 `honeybee.pub`으로 **RSA 암호화된 값**입니다
(요청마다 값이 달라집니다). `ssh_port`는 암호화하지 않고 그대로 돌려줍니다.
응답에는 늘 `resource_type`이 있습니다. `resource_type` 없이 저장된 온프레미스 연결(필드가 생기기 전 등록)은
`vm`으로 돌려줍니다.

csp 연결은 등록할 때 CSP에 리소스를 확인합니다.

- 계정에 없는 `resource_id`는 **400으로 거부**합니다. CSP 목록 조회가 성공했는데 일치하는 리소스가 없을 때만
  없는 것으로 봅니다.
- 그룹 리전과 다른 리전으로 판정된 버킷도 **400으로 거부**합니다([버킷 리전](#버킷-리전)).
- CSP에 물어볼 수 없는 일시 장애(타임아웃, credential, 엔드포인트 도달 불가)는 등록하고
  `connection_status: failed`와 사유를 남깁니다. `refresh`에서 다시 확인합니다.

### `GET /source_group/{sgId}/connection_info` — 목록
Query 필터: `page`, `row`, `name`, `description`, `ip_address`, `ssh_port`, `user`, `resource_type`.

- `resource_type`: `vm`·`k8s`·`object_storage`·`nlb`·`rdbms` 중 하나, 그 밖의 값은 400. 온프레미스 그룹에서
  `vm`은 `resource_type` 없이 저장된 연결도 함께 찾습니다.
- `ssh_port`: 포트 번호가 정확히 같은 연결만 찾습니다. 1~65535가 아닌 값은 필터로 쓰지 않습니다.

### `GET /source_group/{sgId}/connection_info/{connId}` — 조회
`GET /connection_info/{connId}`로 그룹 없이도 조회 가능.

### `PUT /source_group/{sgId}/connection_info/{connId}` — 수정
Body: `model.CreateConnectionInfoReq`.

csp 연결에서는 다음과 같이 동작합니다.

- `vm` 연결은 SSH 필드(`ip_address`, `ssh_port`, `user`, `password`, `private_key`)를 받아 저장합니다
  (비밀값은 OpenBao). **빈 값으로 보낸 필드는 기존 값을 유지**하므로 자격 증명을 바꿀 수는 있어도 지울 수는
  없습니다. 저장됐거나 보낸 `ip_address`가 있으면 `user`와 `password`\|`private_key`가 있어야 하고,
  `ssh_port`는 기본 `22`입니다. 형식이 틀린 값은 건너뛰지 않고 400입니다.
- `vm`이 아닌 연결에 SSH 필드를 보내면 400입니다. `vm` 연결을 다른 `resource_type`으로 바꾸면 SSH 필드와
  비밀값을 지웁니다.
- `resource_type`이나 `resource_id`가 바뀌면 등록 때처럼 CSP에 확인하고, 없는 리소스·다른 리전 버킷은 400입니다.

### `DELETE /source_group/{sgId}/connection_info/{connId}` — 삭제
연결과, 그 연결에 import가 저장한 기록(infra, software, kubernetes, helm, data, rdbms, nlb)을 함께 지웁니다.

### `PUT /source_group/{sgId}/connection_info/{connId}/refresh` — 상태 갱신
`PUT /connection_info/{connId}/refresh`로 그룹 없이도 가능.
**상태만 갱신하고 수집 데이터는 저장하지 않습니다.** 수집·저장은 `POST /import/*`가 합니다.

### `DELETE /source_group/{sgId}/connection_info/{connId}/agent` - 에이전트 제거
소스 호스트에 SSH로 접속해 `uninstallAgent.sh`를 실행합니다. systemd 서비스를 중지·비활성화하고
유닛 파일, `/usr/bin/cm-honeybee-agent`, `/etc/cloud-migrator/cm-honeybee-agent`를 지웁니다.
ConnectionInfo 자체는 남습니다.

### `DELETE /source_group/{sgId}/agent` - 그룹 에이전트 일괄 제거
그룹에 속한 모든 호스트에서 위 작업을 수행합니다.

---

## CSP & Discovery

### `GET /csp` — 지원 CSP 목록
서버에 포함된 cb-spider 드라이버가 지원하는 CSP 목록을 반환합니다.

```bash
curl http://localhost:8081/honeybee/csp
#  { "csp": ["aws", "azure", ...],
#    "supported_resource_types": { "aws": ["vm","k8s","object_storage","nlb","rdbms"], ... } }
```

`supported_resource_types`는 CSP마다 그 CSP의 csp 연결이 수집될 수 있는 `resource_type` 목록입니다.
`vm`·`k8s`·`nlb`·`rdbms`는 링크된 드라이버의 핸들러 지원 여부로, `object_storage`는 honeybee가 S3 API로
버킷을 읽는 CSP인지로 정합니다. honeybee에 드라이버가 없는 CSP는 빈 목록입니다.

### `GET /csp/{name}` — CSP 메타데이터 조회
해당 CSP의 자격 증명 키/리전/기타 메타데이터를 반환합니다.

- **`name`은 대소문자 무시**(`aws` == `AWS`).
- **미지원 CSP**를 주면 400과 함께 **지원 목록을 안내**합니다:
  `"unsupported CSP: \"foo\". Supported CSPs: ALIBABA, AWS, AZURE, GCP, ..."`
- 응답의 `credentials[]`는 각 credential 키에 **예시값·설명**을 함께 제공(입력 폼 자동 구성용):

```bash
curl http://localhost:8081/honeybee/csp/azure
#  { "name":"azure",
#    "credential_keys":["clientId","clientSecret","tenantId","subscriptionId"],
#    "credentials":[ {"key":"clientId","example":"<AZURE_CLIENT_ID>","description":"Azure Client ID, a GUID"}, ... ],
#    "region_keys":["Region","Zone"], "default_region":"koreacentral" }
```

`default_region`은 cb-spider 드라이버 메타정보에 적힌 그 CSP의 기본 질의 리전입니다(비어 있으면 생략).

> **`credential_keys`의 표기는 CSP마다 다릅니다.** 하나로 뭉뚱그리지 마세요.
> AWS는 snake_case 소문자(`aws_access_key_id`), Azure는 camelCase(`clientId`)입니다.
> 이 값은 cb-spider 메타정보의 `credentialcsp`를 그대로 내보내므로 `go.mod`에 고정된 cb-spider 버전을 따릅니다.
> `credentials[]`의 예시·설명은 honeybee가 키 이름을 **정확히 일치**시켜 붙이므로
> (`controller/csp.go`의 `credentialExamples`), 이름이 어긋나면 `example`/`description`이
> 빈 값으로 내려옵니다.

> `region_keys` 필드는 리전 "키 구조"(`Region`/`Zone`)일 뿐 실제 리전 목록이 아닙니다.
> 실제 리전 목록은 credential이 필요하므로 그룹을 만들기 전에는 아래 `POST /csp/{name}/region`을,
> 만든 뒤에는 `GET /source_group/{sgId}/region`을 쓰세요.

### `POST /csp/{name}/region` - 그룹 없이 리전/존 목록 조회
바디의 credential로 CSP에 live 질의해 리전과 각 리전의 zone을 돌려줍니다. 응답 모양은
`GET /source_group/{sgId}/region`과 같습니다. credential은 이 조회에만 쓰고 **저장하거나 로그에 남기지
않습니다**. 키는 `GET /csp/{name}`의 `credential_keys`를 따릅니다. 질의에 쓰는 리전은 그 CSP의 목록 조회용
기본 리전입니다.

```bash
curl -X POST http://localhost:8081/honeybee/csp/aws/region \
  -H 'Content-Type: application/json' \
  -d '{"credential":[{"key":"aws_access_key_id","value":"$AWS_ACCESS_KEY_ID"},
                     {"key":"aws_secret_access_key","value":"$AWS_SECRET_ACCESS_KEY"}]}'
```

### `GET /source_group/{sgId}/region` — CSP 실제 리전/존 목록 (live)
`csp` SourceGroup의 **저장된 credential로 CSP에 live 질의**하여 그 CSP의 실제 리전 목록과
각 리전의 zone을 반환합니다(등록 후 사용). 등록 前 단계(`GET /csp/{name}`)에서는 credential이 없어
실제 리전을 못 주므로, 소스그룹 등록 후 이 API로 리전/zone 선택지를 채웁니다.

```bash
curl http://localhost:8081/honeybee/source_group/$SG/region
#  { "provider":"azure",
#    "regions":[ {"name":"koreacentral","display_name":"...","zones":["1","2","3"]}, ... ] }
```

> 여기서 얻은 `zones` 값이 ConnectionInfo의 `zone`(아래) 입력 후보가 됩니다.

### `GET /source_group/{sgId}/discover` — CSP 리소스 디스커버리
`type=csp` SourceGroup에 바인딩된 CSP 연결로 접근 가능한 VM / K8s 클러스터 / 오브젝트 스토리지 버킷 / NLB / 관리형 DB를
나열합니다(UI에서 ConnectionInfo 선택지를 채울 때 사용).

| 쿼리 파라미터 | 필수 | 설명 |
|---------------|------|------|
| `resource_type` | 예 | `vm` \| `k8s` \| `object_storage` \| `nlb` \| `rdbms` |

```bash
curl "http://localhost:8081/honeybee/source_group/$SG/discover?resource_type=k8s"
```

`object_storage`는 버킷 리전을 판정하는 CSP(AWS)에서 **그룹 리전의 버킷만** 돌려주고 각 항목의 `region`을
채웁니다([버킷 리전](#버킷-리전)).

---

## 소스 정보 수집

각 데이터 타입마다 두 가지 범위(연결 1개 / 그룹 전체)가 있습니다.

```bash
# 노드 1개
curl $BASE/source_group/$SG/connection_info/$CONN/infra
# 그룹 전체
curl $BASE/source_group/$SG/infra
```

`software`, `kubernetes`, `helm`, `data`, `rdbms`, `nlb`도 동일한 패턴을 따릅니다.

---

## 자원 종류별 수집·정제

그룹 하나를 한 번에, 또는 그 안의 한 종류만 수집·정제하는 API입니다. 경로는 모두
`/source_group/{sgId}` 아래입니다.

| 자원 종류 | 그룹 수집 | 그룹 정제 | 연결 1개 정제 |
|-----------|-----------|-----------|----------------|
| onprem·csp 그룹의 전부 | `POST import/all` | - | - |
| 호스트 (onprem vm, csp vm) | `POST import/infra` | `GET infra/refined` | `GET connection_info/{connId}/infra/refined` |
| 호스트 안의 소프트웨어 (에이전트) | `POST import/software` | `GET software/refined` | `GET connection_info/{connId}/software/refined` |
| csp 버킷 (`object_storage`) | `POST import/bucket` | `GET bucket/refined` | `GET connection_info/{connId}/bucket/refined` |
| csp 관리형 DB (`rdbms`) | `POST import/rdbms` | `GET rdbms/refined` | `GET connection_info/{connId}/rdbms/refined` |
| csp 관리형 K8s (`k8s`) | `POST import/k8s` | `GET k8s/refined` | `GET connection_info/{connId}/k8s/refined` |
| csp 로드밸런서 (`nlb`) | `POST import/nlb` | `GET nlb/refined` | `GET connection_info/{connId}/nlb/refined` |
| fs·db·minio 그룹 | `POST import/fs`·`import/db`·`import/objectstorage` | `GET fs/refined`·`db/refined`·`objectstorage/refined` | 같은 경로가 `connection_info/{connId}/` 아래 |

- **`import/all`**: 그룹의 모든 연결을 각 타입에 맞게 수집합니다. `vm`이 아닌 csp 리소스는 CSP에서,
  VM은 infra로(csp 그룹이면 CSP 쪽, `ip_address`가 있으면 에이전트까지), 에이전트가 닿는 호스트는
  software·kubernetes·helm까지 수집합니다. 온프레미스 `k8s` 연결은 수집할 것이 없어 빠집니다.
  `import/infra`의 옵션(`count_objects`)과 `import/software`의 옵션(`show_default_packages`)을 받습니다.
  `fs`·`db`·`minio` 그룹은 맞는 API를 알려주며 400으로 거부합니다.
- **`import/bucket`·`import/rdbms`·`import/k8s`·`import/nlb`**: csp 그룹에서 해당 `resource_type` 연결만
  CSP로 수집하고 나머지는 건드리지 않습니다. csp가 아닌 그룹은 400입니다. VM 위의 K8s 클러스터는
  에이전트 경로인 `import/kubernetes`로 수집합니다.
- 이 다섯 API의 응답은 연결마다 `connection_id`, `resource_type`, `results`를 담고, `results`는 실행한
  수집마다 `{kind, status, error_message}`입니다. `kind`는 `infra`·`software`·`kubernetes`·`helm`·`bucket`·
  `rdbms`·`nlb` 중 하나입니다.
- 기존 `import/infra`와 `data/refined`는 **호환용으로 그대로** 남아 있습니다. csp 그룹의 `import/infra`는
  모든 resource_type을 수집하고, `data/refined`는 `bucket/refined`와 같은 결과입니다.

### 그룹 수집은 목적에 맞는 연결만, 실패해도 멈추지 않음

- `import/software`·`import/kubernetes`·`import/helm`·`import/data`는 에이전트 경로라 **에이전트가 닿는
  연결만** 돕니다: 온프레미스 호스트, `ip_address`가 있는 csp `vm`, 그 밖의 그룹 타입의 연결. 나머지는
  실패로 남기지 않고 건너뜁니다. 그런 연결 하나에 단건으로 부르면 거부합니다.
- `import/infra`는 온프레미스 `k8s`를 뺀 모든 연결을 돕니다.
- 한 연결이 실패해도 나머지를 계속 수집합니다. 실패한 연결은 응답에 `status: failed`와 사유
  `error_message`로 남습니다(`error_message`는 응답에만 있고 저장하지 않습니다). **하나도 성공하지 못했을
  때만** 400입니다. 수집할 연결이 없으면 200과 빈 목록입니다.

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

정제 API는 **마지막 import가 성공한 기록만** 씁니다. 수집하지 않았거나 마지막 import가 실패한 연결은
그룹 결과에서 빠집니다. 종류별 정제 API는 [자원 종류별 수집·정제](#자원-종류별-수집정제) 절의 표를 보세요.

### csp 그룹의 `infra/refined`

- **노드는 `vm` 연결만** 됩니다. 버킷·DB·클러스터·NLB 연결은 노드가 아닙니다. 에이전트가 수집한 VM은
  온프레미스 호스트와 같이 에이전트 데이터로 정제합니다.
- **에이전트가 수집하지 않은 VM**(`ip_address` 없음, 또는 CSP 수집은 성공하고 에이전트 호출만 실패)은
  CSP 수집값으로 노드를 만듭니다.

  | 노드 필드 | 값 |
  |-----------|-----|
  | `hostname` | VM 이름 |
  | `machineId` | CSP의 VM ID |
  | `cpu` | vCPU 수를 `cpus: 1`, `cores`·`threads`: vCPU 수로. `architecture`는 에이전트 표기(`x86_64`, `aarch64`)로 맞춤 (`amd64`, `arm64` 아님) |
  | `memory` | VM 스펙의 메모리 |
  | `rootDisk`, `dataDisks` | CSP가 크기를 알려준 디스크 |
  | `interfaces` | 사설 IP 하나(서브넷 CIDR을 알면 그 prefix). **공인 IP는 넣지 않습니다** - CSP가 게스트 밖에서 매핑하는 주소라 게스트 NIC에 없고, cm-beetle은 인터페이스 주소를 노드 자신의 주소로 읽습니다 |
  | `gpuCards` | VM 스펙의 GPU |
  | `firewallTable` | 보안 그룹 규칙 |
  | `os` | 이미지 설명에서 판별(아래) |

- **OS 판별 한계**: 이미지 설명은 CSP마다 다른 자유 텍스트라서 배포판 이름(Ubuntu, Debian, Rocky,
  AlmaLinux, CentOS, RHEL, Oracle Linux, Amazon Linux, SLES, Fedora, Windows)을 찾고 그 뒤의 첫 숫자를
  버전으로 씁니다(Ubuntu·Debian은 코드네임도 봄). 이미지의 OS 설명·이름·ID 어디에도 배포판 이름이 없으면
  (예: `al2023-ami-...`로만 알려진 Amazon Linux 이미지) `id`·`versionId`가 비고 `prettyName`에 설명만
  남습니다. openSUSE를 포함한 SUSE 이미지는 모두 `sles`로 판별됩니다.
- **`k8sCluster`**: 에이전트가 노드에서 수집한 클러스터가 우선이고, 없으면 csp 관리형 클러스터의 이름·버전을
  씁니다. 관리형 클러스터의 Pod/Service CIDR은 비어 있고 노드는 나열하지 않습니다.
- **`nlbs`**: 노드의 HAProxy에서 찾은 NLB 뒤에 csp NLB(마지막 import가 성공한 `nlb` 연결)가 붙습니다.
  csp NLB는 `software`가 `haproxy` 대신 **`"<provider>-nlb"`**(예: `aws-nlb`), `hostMachineId`는 비고,
  리스너 주소는 `PUBLIC`이면 `*`, `INTERNAL`이면 첫 IP(없으면 DNS 이름), 타입을 모르면 빈 값입니다.
  balance 알고리즘은 CSP가 알려주지 않아 비어 있습니다. 백엔드 서버 IP는 그룹의 csp VM 사설 IP를 VM ID·이름으로
  맞춰 채우며, 그룹에 등록되지 않은 VM은 IP가 빕니다.

### `k8s/refined`·`nlb/refined`·`bucket/refined`

- `k8s/refined`: `infra/refined`의 `k8sCluster`와 같은 값에 `connection_id`, `managed`(csp 관리형이면
  `true`), 관리형이면 노드 그룹을 담은 `csp` 섹션을 붙입니다. 연결 1개는 클러스터가 없으면 400, 그룹은
  `k8sCluster: null`입니다.
- `software/refined`에는 **csp 관리형 클러스터의 kubernetes 항목이 없습니다**. 워크로드를 수집하지 않기
  때문입니다.
- `nlb/refined`: `infra/refined`의 `nlbs`와 같은 목록을 `nlbs`로 돌려줍니다. 그룹에 없으면 빈 목록입니다.
- `bucket/refined`(=`data/refined`): [버킷 리전](#버킷-리전) 절의 리전 규칙을 따릅니다.

### 노드의 GPU (`gpuCards`)

물리 GPU 카드 하나가 항목 하나입니다. 한 노드에 서로 다른 모델이 꽂혀 있어도 카드별로 그대로
실리므로, 요약 하나로 뭉개지지 않습니다.

```json
"gpuCards": [
  {
    "driverIndex": "0",
    "uuid": "GPU-1aefeed7-20f6-5f9f-a3a5-119cedf83094",
    "vendor": "NVIDIA",
    "model": "NVIDIA L40S",
    "type": "GPU",
    "architecture": "Ada Lovelace",
    "driverVersion": "595.71.03",
    "pciBusId": "00000000:61:00.0",
    "eccEnabled": true,
    "memoryTotalGB": 44.98828,
    "memoryReservedGB": 1.078125,
    "memoryFreeGB": 36.848633,
    "memoryUsedGB": 7.0634766
  }
]
```

| 필드 | 채워지는 값 |
|------|-------------|
| `driverIndex` | 드라이버가 쓰는 표기를 **그대로**. NVIDIA는 `"0"`, AMD는 `"card0"`·`"card10"` |
| `vendor` / `model` / `architecture` | 각 드라이버가 보고한 값 |
| `eccEnabled` | ECC가 켜져 있는지. 드라이버가 ECC를 보고하지 않으면 `false` |
| `memoryTotalGB` | 드라이버가 보고한 총량을 GB로. **보정하지 않습니다** |
| `memoryReservedGB` | 그 총량 **안에** 잡혀 있는 예약분 |

읽을 때 주의할 점.

- **`driverIndex`는 배열 안에서 유일하지 않습니다.** NVIDIA와 AMD가 같이 꽂힌 노드에서는
  nvidia-smi의 `0`과 rocm-smi의 `card0`이 각자 0번입니다. 노드 안에서 카드를 가리는 키로는
  `uuid`나 `pciBusId`를 쓰세요.
- **ECC가 켜진 카드의 `memoryTotalGB`는 카드 표기 용량보다 작습니다.** 위 L40S는 공칭 48GB인데
  44.99가 실립니다. 보정하지 않는 이유는 보정한 값과 날것을 구분할 수 없게 되기 때문이고,
  대신 `eccEnabled`와 `memoryReservedGB`를 같이 보냅니다. `memoryTotalGB + memoryReservedGB`로도
  표기 용량이 복원되지 않는다는 점에 주의하세요 - 둘은 서로 다른 것입니다.
- AMD는 rocm-smi가 free와 reserved를 주지 않아 `memoryFreeGB`·`memoryReservedGB`가 비고,
  ECC 상태도 보고하지 않습니다.
- 쿠버네티스 API로만 보이는 노드는 확장 리소스가 개수만 알려주므로, 장수만큼 항목이 생기고
  `uuid`·`pciBusId`는 빕니다.

---

### 쿠버네티스 소스 모델

정제 인프라 모델은 소스 K8s 클러스터를 하나의 단위로 마이그레이션할 수 있도록 쿠버네티스 소스 정보를
포함합니다. K8s 정보는 `/infra/refined` 응답의 `k8sCluster`에 실리고, 클러스터만 따로 보려면
`/k8s/refined`를 씁니다(`/kubernetes/refined`는 없습니다). 구현 위치는 `server/pkg/api/rest/controller/getRefined.go`이며 핵심은 다음과 같습니다.

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
| `types` | `cpus,cpum,memR,memW,fioR,fioW` | 콤마로 구분된 벤치마크 타입: `cpus, cpum, memR, memW, fioR, fioW, dbR, dbW`. |

> 생략하면 `dbR`/`dbW`를 뺀 6종이 전부 돕니다(`server/lib/ssh/ssh.go`의 `getBenchmarkTypes`).
> Swagger 어노테이션에는 `default(cpus,cpum)`으로 적혀 있으나 실제 동작은 위와 같습니다.

```bash
curl -X POST "http://localhost:8081/honeybee/bench/$CONN/run?types=cpus,memR,fioW"
```

### `POST /bench/{connId}/stop` — 벤치마크 중지
