# CM-Honeybee API 문서

CM-Honeybee는 Cloud-Migrator(cloud-barista) 도구 모음에서 **소스 환경 수집기** 역할을 하는 모듈입니다.
온프레미스(또는 이미 운영 중인 클라우드) 소스 호스트로부터 인프라, 소프트웨어, 쿠버네티스, Helm, 데이터
정보를 수집하고, 이 원시 정보를 `cm-beetle`이 마이그레이션 추천·실행에 사용하는 표준 **소스 모델**로
정제(refine)합니다.

CM-Honeybee는 독립적으로 배포 가능한 두 개의 REST 모듈로 구성됩니다.

| 모듈 | 바이너리 / 모듈명 | Base path | 기본 포트 | 역할 |
|------|------------------|-----------|-----------|------|
| **Agent** (`cm-honeybee-agent`) | `HONEYBEE-AGENT` | `/honeybee-agent` | `8082` | **각 소스 호스트에서 실행.** 로컬 인프라/소프트웨어/k8s/helm/데이터를 수집해 요청 시 반환합니다. |
| **Server** (`cm-honeybee`) | `HONEYBEE` | `/honeybee` | `8081` | 중앙 컨트롤 플레인. SourceGroup / ConnectionInfo를 관리하고, 에이전트(또는 SSH / CSP / cb-spider)로부터 데이터를 수집·저장하며 **정제된 소스 모델**을 제공합니다. |

```
                                       ┌─────────────────────────────┐
   마이그레이션 사용자 / cm-beetle ───▶ │  cm-honeybee SERVER  :8081  │
                                       │   /honeybee                 │
                                       └───────┬─────────────┬───────┘
                                  agent :8082  │             │  cb-spider / SSH
                                   (REST pull)  ▼             ▼  (CSP 디스커버리)
                                ┌──────────────────────┐  ┌──────────────────┐
                                │ cm-honeybee AGENT     │  │  클라우드/온프렘  │
                                │  /honeybee-agent      │  │  소스 호스트들    │
                                └──────────────────────┘  └──────────────────┘
```

## API 레퍼런스

- [Agent API](./agent-api.md) — `cm-honeybee-agent`가 노출하는 엔드포인트 (호스트별 수집기).
- [Server API](./server-api.md) — `cm-honeybee`가 노출하는 엔드포인트 (컨트롤 플레인, 정제 모델, K8s 소스 모델).

## 문서 공통 규칙

- **인증:** 없음. 두 모듈 모두 현재 인증 미들웨어를 적용하지 않으며(요청 로깅만 수행) 신뢰된 내부망에서
  실행하는 것을 전제로 합니다. **외부에 그대로 노출하지 마세요.**
- **콘텐츠 타입:** 요청/응답 본문 모두 `application/json`.
- **경로 접두사:** 아래 모든 경로는 모듈 base path 기준 상대 경로입니다
  (Agent는 `/honeybee-agent`, Server는 `/honeybee`).
  전체 URL 예시: `http://<host>:8081/honeybee/source_group`.
- **인터랙티브 문서 (Swagger UI):** 각 모듈은 `<base-path>/api/`에서 Swagger UI를 제공합니다
  (예: `http://localhost:8081/honeybee/api/index.html`). OpenAPI 명세는 `make swag`로 생성합니다
  (두 모듈에 대해 `swag init`을 실행해 `pkg/api/rest/docs/`에 출력).

## 빠른 시작

```bash
# 1. 소스 호스트에서 Agent 실행
cd agent && make run          # :8082 리슨, base path /honeybee-agent

# 2. 컨트롤 노드에서 Server 실행
cd server && make run         # :8081 리슨, base path /honeybee

# 3. 헬스 체크
curl http://localhost:8082/honeybee-agent/readyz
curl http://localhost:8081/honeybee/readyz
```

"소스 등록 → 수집 → 정제 → cm-beetle 전달"의 전체 흐름은 [Server API 문서](./server-api.md)의
전형적인 워크플로우 절(SSH 타입 / CSP 타입)을 참고하세요. v0.6.0에서는 **SSH 타입 등록을 권장**하며,
CSP(cb-spider) 타입은 실험적/예정 기능입니다.
