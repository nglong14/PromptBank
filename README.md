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

## Frontend (Next.js)

A simple light-theme frontend has been added in `frontend/` with:
- Sign up (`POST /api/v1/auth/register`)
- Login (`POST /api/v1/auth/login`)
- Health check (`GET /health`)
- Prompt list/create/detail/update
- Prompt version create/list
- Prompt derive

### Run frontend locally

1. Copy env file:

```bash
cd frontend
cp .env.local.example .env.local
```

2. Start dev server:

```bash
npm install
npm run dev
```

3. Open [http://localhost:3000](http://localhost:3000)

By default the frontend uses a Next.js rewrite (`/backend/*`) to avoid browser CORS issues.

- `NEXT_PUBLIC_API_BASE_URL=/backend`
- Optional `BACKEND_ORIGIN=http://localhost:8080` (or your backend host)

If you change either value, restart the Next.js server.