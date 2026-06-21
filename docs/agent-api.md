# CM-Honeybee Agent API

**Agent**(`cm-honeybee-agent`)는 **각 소스 호스트에서 실행**되며 해당 호스트 한 대의 정보를 수집합니다.
상태를 저장하지 않는 읽기 전용 수집기로, 모든 호출은 요청 시점에 로컬 머신을 조사해 결과를 반환합니다.
Server가 이 엔드포인트들로부터 데이터를 가져갑니다(에이전트가 푸시하지 않음).

| 항목 | 값 |
|------|-----|
| 모듈명 | `HONEYBEE-AGENT` |
| Base path | `/honeybee-agent` |
| 기본 포트 | `8082` |
| Swagger UI | `http://<host>:8082/honeybee-agent/api/index.html` |
| 인증 | 없음 |

> 아래 모든 경로는 base path 기준 상대 경로입니다. 전체 URL 예시:
> `http://localhost:8082/honeybee-agent/infra`

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
curl http://localhost:8082/honeybee-agent/readyz
```

---

## Infra

### `GET /infra` — 통합 인프라 정보 조회

호스트의 전체 인프라 구성을 수집합니다: 컴퓨트(CPU/메모리), 루트 + 데이터 디스크, 네트워크 인터페이스,
라우팅 테이블, 방화벽 규칙, OS 메타데이터.

```bash
curl http://localhost:8082/honeybee-agent/infra
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
curl "http://localhost:8082/honeybee-agent/software"

# OS 기본/베이스 패키지까지 포함
curl "http://localhost:8082/honeybee-agent/software?show_default_packages=true"
```

---

## Kubernetes

### `GET /kubernetes` — 쿠버네티스 정보 조회

쿠버네티스 **클러스터** 메타데이터(이름, 버전, CNI 플러그인, Pod/Service CIDR, NodePort 범위)와
**노드** 정보(`control-plane`/`worker` 등 역할, 노드 스펙, machine ID)를 수집합니다.

> **중요:** 쿠버네티스(및 Helm) 수집은 호스트가 *접근 가능한 쿠버네티스 컨트롤 플레인*일 때만
> 동작합니다. 컨트롤 플레인이 아니면 에이전트가 수집을 건너뜁니다(커밋 `66c7305`). kubeconfig 경로는
> `KUBECONFIG` 환경 변수가 설정되어 있으면 그 값을, 없으면 기본 위치를 사용합니다(커밋 `1d73b05`).

```bash
curl http://localhost:8082/honeybee-agent/kubernetes
```

Server는 이 출력을 정제 소스 모델로 매핑합니다 —
[Server API → 쿠버네티스 소스 모델](./server-api.md#쿠버네티스-소스-모델-etri-요청) 참고.

---

## Helm

### `GET /helm` — Helm 정보 조회

호스트가 속한 클러스터에 설치된 Helm 릴리스 목록을 수집합니다. 쿠버네티스 엔드포인트와 마찬가지로
접근 가능한 컨트롤 플레인 호스트에서만 의미가 있습니다.

```bash
curl http://localhost:8082/honeybee-agent/helm
```

---

## Data

### `GET /data` — 데이터 마이그레이션 정보 조회

데이터 마이그레이션에 필요한 필드로 한정하여 관련 정보를 수집합니다.

```bash
curl http://localhost:8082/honeybee-agent/data
```
