# PromptBank

Week 1-2 backend foundation is implemented:
- Go API service with chi router
- Postgres-backed auth and prompt CRUD/versioning/derive flows
- JWT auth middleware
- Docker Compose for local Postgres + Redis + API
- CI pipeline for format checks and tests

## Quick Start

1. Start dependencies and API:

```bash
docker compose up --build
```

2. Check health:

```bash
curl http://localhost:8080/health
```

## Environment Variables

Copy `.env.example` to `.env` for local non-docker runs.

- `PORT`
- `DATABASE_URL`
- `JWT_SECRET`
- `JWT_EXPIRES_MINUTES`
- `REDIS_ADDR`

## API (Week 1-2)

Public:
- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`
- `GET /health`

Protected (Bearer token):
- `GET /api/v1/prompts`
- `POST /api/v1/prompts`
- `GET /api/v1/prompts/{promptID}`
- `PATCH /api/v1/prompts/{promptID}`
- `POST /api/v1/prompts/{promptID}/versions`
- `GET /api/v1/prompts/{promptID}/versions`
- `POST /api/v1/prompts/derive`