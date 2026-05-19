# Changelog

이 프로젝트의 주요 변경 사항을 기록합니다. [Keep a Changelog](https://keepachangelog.com/ko/1.0.0/) 형식을 따릅니다.

## [Unreleased] - 2026-05-19

### 보안 (Security)
- **JWT 시크릿 프로덕션 가드** — `Config.Validate()`가 `ENV=production`에서 빈 값/기본 placeholder/32자 미만을 거부한다. 잘못된 배포가 공개된 placeholder로 토큰을 서명하던 위험 차단.
- placeholder 문자열을 조립식(`strings.Join`)으로 변경해 리터럴 형태로 소스에 노출되지 않게 함.

### 추가 (Added)
- **Health/readiness 분리**:
  - `/health` (기존) + `/healthz` (k8s 컨벤션 alias) — liveness, 의존성 무관 200.
  - `/readyz` — readiness, 백킹 스토어 ping 실패 시 503 + `store_unreachable`.
- `Store` 인터페이스에 `Ping(ctx) error` — `InMemoryStore`는 no-op, `PostgresStore`는 `db.PingContext`(2s 타임아웃), `CachedStore`는 위임.
- `InMemoryStore.Close()` + cleanup goroutine의 `ctx.Done()` 처리 — graceful shutdown에서 누수 없음.
- LICENSE (MIT), SECURITY.md, CONTRIBUTING.md, PR 템플릿, Dependabot (gomod/actions/docker).
- OpenAPI 스펙(`docs/openapi.yaml`)에 `/healthz`, `/readyz` 반영.
- 운영 런북: `docs/operations/backup-recovery.md` (pg_dump 백업/복구 + 시나리오별 절차 + RPO/RTO 목표).

### 수정 (Fixed)
- `ctx.Err() == context.DeadlineExceeded/Canceled` → `errors.Is(...)`. 래핑된 컨텍스트 에러에서도 timeout/cancellation 분기가 정상 동작.
- `err == sql.ErrNoRows` → `errors.Is(err, sql.ErrNoRows)` (`postgres_store.go` 3곳).
- `service/migrate.go`: `m.Close()` 미호출 사유 코멘트화 — golang-migrate v4의 postgres 드라이버 `Close()`가 공유 풀을 닫는 버그성 동작 회피.

### 운영/인프라 (Ops)
- `.env.example` 정비 — JWT placeholder 제거(빈 값), 누락된 env vars (`REDIS_URL`/`CACHE_TTL_*`/`LOG_LEVEL`/`MAX_OUTPUT_BYTES`) 추가, 각 변수에 운영 영향 주석.
- 신규 endpoint는 OpenAPI에 동시 반영.

### 보류 (Deferred)
- Sentry/관측 통합 — 개인 서버 환경 확인 후 결정 (SaaS / 셀프호스트 / OpenTelemetry / 빌트인만).
