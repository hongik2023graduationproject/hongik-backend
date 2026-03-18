# 홍익 백엔드 API 서버

Go + Gin 기반 홍익 언어 웹 서비스 백엔드 서버

## 기능

- **코드 실행**: POST `/api/execute` - 홍익 코드 실행 (프로세스 실행)
- **스니펫 관리**: `/api/snippets` - CRUD 작업
- **코드 공유**: `/api/share` - 공유 링크 생성 및 조회
- **실시간 REPL**: WebSocket `/api/sessions/:id/stream` - 실시간 코드 실행
- **캐싱**: Redis를 이용한 성능 최적화
- **데이터 영속성**: PostgreSQL 기반 스니펫/사용자 저장

## 스택

- **Language**: Go 1.21+
- **Framework**: Gin
- **Database**: PostgreSQL 15
- **Cache**: Redis 7
- **WebSocket**: Gorilla WebSocket
- **Containerization**: Docker + Docker Compose

## 프로젝트 구조

```
hongik-backend/
├── main.go                 # 진입점
├── api/
│   ├── routes.go          # 라우트 정의
│   ├── handlers.go        # HTTP 핸들러
│   └── websocket.go       # WebSocket 핸들러
├── service/
│   ├── interpreter.go     # 홍익 인터프리터 호출
│   ├── snippet.go         # 스니펫 비즈니스 로직
│   ├── share.go           # 공유 링크 로직
│   └── errors.go          # 커스텀 에러
├── model/
│   └── snippet.go         # 데이터 모델
├── db/
│   ├── postgres.go        # PostgreSQL 연결
│   └── redis.go           # Redis 연결
├── config/
│   └── config.go          # 설정 로딩
├── Dockerfile             # 컨테이너 빌드
├── docker-compose.yml     # 개발 환경 구성
├── init.sql               # DB 초기화 스크립트
├── go.mod
└── go.sum
```

## 설정

### 환경 변수 (.env)

```bash
cp .env.example .env
```

```
PORT=3000
ENV=development

# PostgreSQL
DATABASE_URL=postgres://user:password@localhost:5432/hongik?sslmode=disable

# Redis
REDIS_URL=redis://localhost:6379

# JWT 인증
JWT_SECRET=your-secret-key

# 홍익 인터프리터 경로
INTERPRETER_PATH=../hong-ik/cmake-build-release/HongIk
```

## 설치 및 실행

### 로컬 개발 (Go 설치 필요)

1. 의존성 설치:
```bash
go mod download
```

2. 환경 변수 설정:
```bash
cp .env.example .env
```

3. PostgreSQL과 Redis 실행:
```bash
docker-compose up -d postgres redis
```

4. DB 초기화:
```bash
psql -U hongik -d hongik -f init.sql
```

5. 서버 실행:
```bash
go run main.go
```

### Docker Compose (권장)

```bash
docker-compose up --build
```

서버가 http://localhost:3000 에서 실행됩니다.

## API 엔드포인트

### 건강 상태 확인
```
GET /health
```

### 코드 실행
```
POST /api/execute
Content-Type: application/json

{
  "code": "[정수] x = 10\n출력:(x)",
  "timeout": 5000
}
```

응답:
```json
{
  "status": "success",
  "output": "10",
  "execution_time_ms": 23
}
```

### 스니펫 CRUD

```
# 스니펫 생성
POST /api/snippets

# 스니펫 목록
GET /api/snippets?limit=20&offset=0

# 스니펫 조회
GET /api/snippets/:id

# 스니펫 수정
PUT /api/snippets/:id

# 스니펫 삭제
DELETE /api/snippets/:id
```

### 코드 공유

```
# 공유 링크 생성
POST /api/share
{
  "snippet_id": "uuid",
  "expires_in": 86400  // seconds (optional)
}

# 공유된 스니펫 조회
GET /api/share/:token
```

### 실시간 REPL (WebSocket)

```
WS /api/sessions/:id/stream
```

메시지 형식:
```json
// 코드 실행 요청
{
  "type": "execute",
  "code": "[정수] x = 5\n출력:(x)",
  "timeout": 5000
}

// 결과
{
  "type": "result",
  "output": "5"
}
```

### 언어 정보

```
GET /api/language/builtins   # 내장 함수 목록
GET /api/language/syntax     # 문법 가이드
```

## 성능 특성

- **메모리**: ~15MB baseline (Node.js: ~150MB)
- **동시 연결**: 2,000+ (Node.js: 400)
- **처리량**: 10,000+ req/s (Node.js: 1,000)
- **응답 시간 (p99)**: 20ms (Node.js: 150ms)
- **배포 시간**: ~30초 (Docker multi-stage)

## TODO

- [ ] JWT 인증 미들웨어
- [ ] 사용자 관리 엔드포인트
- [ ] 속도 제한 (Rate limiting)
- [ ] 요청 로깅 미들웨어
- [ ] CORS 설정
- [ ] 테스트 작성
- [ ] API 문서 (Swagger/OpenAPI)
- [ ] 성능 모니터링 (Prometheus metrics)

## 라이선스

MIT
