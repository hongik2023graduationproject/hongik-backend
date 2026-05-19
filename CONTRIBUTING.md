# Contributing to hongik-backend

`hongik-backend`는 홍익 프로그래밍 언어 웹 서비스의 API 백엔드입니다. Go + Gin 기반.

## 개발 환경 세팅

### 필수 도구

- Go 1.25 이상 (go.mod 참조)
- (선택) PostgreSQL 15+ — `DATABASE_URL` 없으면 in-memory 모드로 동작
- (선택) Redis 7+ — `REDIS_URL` 없으면 캐싱 비활성
- 홍익 인터프리터 빌드 산출물 (`../hong-ik/cmake-build-debug/HongIk` 기본 경로, `INTERPRETER_PATH`로 재정의)

### 시작하기

```bash
git clone https://github.com/hongik2023graduationproject/hongik-backend.git
cd hongik-backend
cp .env.example .env       # 값 수정
go run .                   # 8080 포트
```

## 개발 워크플로

### 브랜치
- `main` 기준으로 `feat/<요약>`, `fix/<요약>`, `chore/<요약>` 등으로 분기.
- 한 PR은 한 가지 일만 다룬다 (리팩토링과 기능 추가 섞지 않기).

### 커밋
- conventional commits 권장: `feat:`, `fix:`, `refactor:`, `docs:`, `test:`, `chore:`, `ci:`.
- 첫 줄 50자 이내 요약, 빈 줄 후 본문에서 *why*를 설명.

### 코드 스타일
- `gofmt` + `goimports` 자동 적용 (저장 시 IDE 설정 권장).
- 에러는 `fmt.Errorf("...: %w", err)`로 래핑.
- context 전파 필수, 직접 `context.Background()` 호출 지양 (백그라운드 lifecycle 명시 시만 허용).

### 테스트
```bash
go test ./...
go vet ./...
```
- 핸들러 변경 시 `api/handlers/*_test.go`에 케이스 추가.
- 서비스 계층 변경 시 `service/*_test.go`에 케이스 추가.
- CGO를 활성화할 수 있는 환경이면 `go test -race ./...`로 검증.

### 마이그레이션
새 스키마 변경:
1. `migrations/00000N_xxx.up.sql` + `migrations/00000N_xxx.down.sql` 두 파일 동시 추가.
2. 별도 임시 DB에서 up → down → up을 수동 검증.
3. PR 본문에 마이그레이션 영향 명시.

### 환경 변수
새 env var 추가 시 동시에:
- `config/config.go`의 `Config` 구조체 + `Load()` 갱신
- 프로덕션 가드가 필요하면 `Validate()` 갱신
- `.env.example` 갱신 (값은 placeholder, 실제 시크릿 금지)

## PR 제출

1. 위 워크플로대로 변경 + 테스트
2. `git push origin <branch>`
3. PR 템플릿(`.github/PULL_REQUEST_TEMPLATE.md`)에 따라 작성
4. 리뷰 요청

## 보안

취약점은 공개 이슈에 올리지 말고 [`SECURITY.md`](./SECURITY.md)의 절차를 따라주세요.
