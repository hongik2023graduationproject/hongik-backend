# PostgreSQL 백업 및 복구 절차

> 본 문서는 운영 환경 PostgreSQL의 일일 백업 + 장애 시 복구 절차를 정의한다.
> 클라우드 관리형 DB(예: GCP Cloud SQL, AWS RDS)를 쓰면 1-2절 대부분은 콘솔 설정으로 대체 가능하며, 본 문서는 자체 운영(VM/베어메탈) 기준이다.

---

## 1. 백업 전략

### 1.1 정책

| 항목 | 값 |
|---|---|
| 백업 주기 | 매일 1회 (UTC 18:00 = KST 03:00) |
| 보존 기간 | 일간 7일 / 주간 4주 / 월간 6개월 |
| 위치 | 동일 리전 오브젝트 스토리지 + 다른 리전 미러 1부 |
| 암호화 | 객체 스토리지 SSE-S3 (또는 동급) + 전송 시 TLS |
| 검증 | 매주 1회 무작위 백업 파일 1개 복원 리허설 (별도 임시 DB로) |

### 1.2 백업 명령

논리 백업(개발 환경 + 소규모 데이터에 적합):

```bash
# 환경 변수
export PGHOST="${DB_HOST}"
export PGPORT="${DB_PORT:-5432}"
export PGUSER="${DB_USER}"
export PGPASSWORD="${DB_PASSWORD}"
export PGDATABASE="${DB_NAME:-hongik}"

# pg_dump (custom format은 병렬 복원 + 선택적 복원 지원)
TS=$(date -u +%Y%m%dT%H%M%SZ)
BACKUP_FILE="hongik-${TS}.dump"

pg_dump \
  --format=custom \
  --compress=9 \
  --no-owner \
  --no-acl \
  --verbose \
  --file="${BACKUP_FILE}"

# 객체 스토리지 업로드 (예: AWS S3)
aws s3 cp "${BACKUP_FILE}" "s3://hongik-backups/postgres/daily/${BACKUP_FILE}" \
  --storage-class STANDARD_IA \
  --sse AES256

# 로컬 임시 파일 정리
rm "${BACKUP_FILE}"
```

물리 백업 (대용량 + PITR 필요 시):
- `pg_basebackup` + WAL archiving 조합. 운영 부하가 일정 수준 이상이면 이쪽으로 전환.

### 1.3 자동화

옵션 1 — cron + 운영 서버에서 직접:
```cron
0 18 * * * /opt/hongik/scripts/backup.sh >> /var/log/hongik/backup.log 2>&1
```

옵션 2 — GitHub Actions schedule (DB가 외부 접근 가능한 경우만):
```yaml
on:
  schedule:
    - cron: "0 18 * * *"
```

옵션 3 — 클라우드 관리형 DB의 내장 백업 기능 (권장). GCP Cloud SQL: "Automated backups" 토글, AWS RDS: "Backup retention period" 설정.

### 1.4 무결성 검증

```bash
# 백업 파일 자체 무결성
pg_restore --list "${BACKUP_FILE}" | head -20

# 주간 1회 — 임시 DB로 복원 후 row count 비교
createdb hongik_verify
pg_restore --dbname=hongik_verify --no-owner --no-acl "${BACKUP_FILE}"
psql -d hongik_verify -c "SELECT COUNT(*) FROM snippets;"
psql -d hongik_verify -c "SELECT COUNT(*) FROM users;"
dropdb hongik_verify
```

리허설 결과는 Slack #ops 채널에 기록한다.

---

## 2. 복구 절차

### 2.1 시나리오 A — 데이터 손상 (논리적, 일부 row)

```bash
# 1. 영향 범위 파악
psql -d hongik -c "SELECT COUNT(*) FROM snippets WHERE ...;"

# 2. 가장 최근 백업으로부터 영향받은 테이블만 복원
pg_restore \
  --dbname=hongik \
  --table=snippets \
  --data-only \
  --disable-triggers \
  /path/to/hongik-20260518T180000Z.dump

# 3. 정합성 확인
psql -d hongik -c "SELECT COUNT(*), MAX(created_at) FROM snippets;"
```

⚠️ 운영 중 복원은 lock contention을 일으킨다. 가능하면 일시적으로 readiness probe(/readyz)가 503을 내도록 일시 정지한 뒤 진행.

### 2.2 시나리오 B — 전체 DB 손상 / 클러스터 손실

```bash
# 1. 새 DB 인스턴스 프로비저닝 + readiness probe로 트래픽 차단
# 2. 가장 최근 백업 다운로드
aws s3 cp s3://hongik-backups/postgres/daily/hongik-20260518T180000Z.dump ./

# 3. 빈 DB 생성
createdb hongik

# 4. 마이그레이션 도구가 아닌 pg_restore 사용 (스키마 + 데이터 한 번에)
pg_restore \
  --dbname=hongik \
  --no-owner \
  --no-acl \
  --verbose \
  --jobs=4 \
  hongik-20260518T180000Z.dump

# 5. 백엔드 설정에서 DATABASE_URL을 새 인스턴스로 갱신
# 6. /readyz가 200을 반환할 때까지 대기, 트래픽 복구
```

복구 직후 발생할 수 있는 일:
- 백업 시점 ~ 장애 시점 사이의 데이터 손실(RPO = 백업 주기 = 24h). 손실 데이터가 크면 신청자에게 재제출 안내.
- 캐시는 자동 무효화 (Redis TTL). 첫 트래픽은 cache-miss로 약간 느림.

### 2.3 시나리오 C — 잘못된 마이그레이션

```bash
# 1. 새 트래픽 차단 (readiness probe 일시 정지)
# 2. golang-migrate down으로 직전 버전으로 되돌림
DATABASE_URL=... ./bin/migrate -path migrations -database "$DATABASE_URL" down 1

# 또는 백업에서 스키마만 복원
pg_restore --schema-only --clean --if-exists --dbname=hongik latest.dump

# 3. 코드도 한 버전 롤백 (GHCR :sha 태그로)
docker pull ghcr.io/.../hongik-backend:<previous-sha>
docker compose up -d backend

# 4. 데이터 정합성 확인 후 트래픽 복구
```

마이그레이션은 항상 **down 스크립트 + 별도 DB에서 리허설** 후 적용한다 (`migrations/000001_init.down.sql` 같은 파일 존재 확인).

---

## 3. RPO / RTO 목표

| 지표 | 목표 | 비고 |
|---|---|---|
| RPO (복구 가능한 최대 데이터 손실) | 24시간 | 일일 백업 주기 |
| RTO (복구까지 걸리는 시간) | < 2시간 | 시나리오 B 기준; A/C는 < 30분 |

RPO를 줄이려면 WAL archiving + PITR로 전환. RTO를 줄이려면 hot standby 또는 관리형 DB의 다중 가용영역 옵션.

---

## 4. 런북 — 장애 발생 시 첫 15분

1. **확인**: `curl https://api.hong-ik.dev/readyz` — DB 문제면 `not_ready` + `store_unreachable` 메시지
2. **로그**: 백엔드 슬로그에서 `failed to connect to PostgreSQL` 또는 store 관련 에러 검색
3. **상태**: DB 인스턴스의 시스템 상태(CPU/메모리/디스크) 점검
4. **분류**:
   - 일시적 (네트워크/CPU 스파이크) → 재시도 + 모니터링
   - 영구적 (데이터 손상/인스턴스 손실) → 시나리오 B로 진입
5. **공지**: Status page에 incident 등록 + Slack #incidents에 알림
6. **에스컬레이션**: 30분 내 미해결 시 온콜에게 핸드오프

---

## 5. 점검 체크리스트 (분기별)

- [ ] 최근 1주일 백업이 모두 객체 스토리지에 존재하는지
- [ ] 백업 파일 크기가 예상 범위 내인지 (갑작스러운 감소 = 백업 실패)
- [ ] 무작위 백업 1건을 임시 DB로 복원해서 row count 검증
- [ ] 전체 복구 시나리오를 스테이징에서 처음부터 끝까지 리허설
- [ ] migrate up/down 스크립트가 모든 마이그레이션에 대칭으로 존재하는지
