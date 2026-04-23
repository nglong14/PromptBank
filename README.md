# PromptBank

## Table of contents
- [Overview](#overview)
- [Feature](#feature)
- [Structure](#structure)
- [Technologies used](#technologies-used)
- [Installation](#installation)
- [Contact](#contact)
- [Room for improvement](#room-for-improvement)

## Overview
PromptBank is a prompt engineering workspace with:
- A Go backend API for auth, prompt management, versioning, and AI-assisted operations
- A Next.js frontend for composing prompts and managing prompt assets
- Gemini-powered helpers for normalization, framework suggestion, scoring, and iterative refinement

The project focuses on making prompt creation structured, repeatable, and versioned.

## Feature
- User authentication (register/login) with JWT
- Prompt CRUD and ownership-scoped access control
- Prompt versioning and derive flow
- Prompt composition from:
  - Asset fields (`goal`, `persona`, `context`, `tone`, `constraints`, `examples`)
  - Selected framework
  - Selected techniques
- LLM capabilities:
  - Wizard answer normalization
  - Framework suggestion
  - Prompt quality scoring
  - Iterative refinement with tool-calling
  - Technique-specific AI suggestions (few-shot examples, role persona, constraints)
- Frontend pages for signup/login, prompt list, prompt detail/composition, version browsing

## Structure
```text
PromptBank/
├─ cmd/
│  ├─ api/                 # API entrypoint
│  └─ worker/              # Worker scaffold (currently placeholder)
├─ internal/
│  ├─ http/                # Router + handlers
│  ├─ llm/                 # Gemini client + normalize/score/refine/suggestion logic
│  ├─ repository/          # Postgres repositories
│  ├─ security/            # JWT and auth helpers
│  ├─ compose/             # Prompt composing pipeline
│  ├─ framework/           # Framework definitions and slot mapping
│  ├─ technique/           # Technique definitions and apply logic
│  └─ asset/               # Asset models and normalization
├─ frontend/               # Next.js app
├─ migrations/             # SQL migrations
├─ docker-compose.yml      # Local infra: Postgres + Redis + API
└─ README.md
```

## Technologies used
- Backend: Go (Chi router, pgx, JWT)
- Frontend: Next.js, React, TypeScript
- Database: PostgreSQL
- Cache/queue-ready infra: Redis
- AI integration: Google Gemini (`google/generative-ai-go`)
- Dev environment: Docker Compose

## Installation
### 1) Backend + infrastructure
1. Copy environment file:
   ```bash
   cp .env.example .env
   ```
2. Start services:
   ```bash
   docker compose up --build
   ```
3. Verify API health:
   ```bash
   curl http://localhost:8080/health
   ```

### 2) Frontend
1. Configure frontend env:
   ```bash
   cd frontend
   cp .env.local.example .env.local
   ```
2. Install and run:
   ```bash
   npm install
   npm run dev
   ```
3. Open:
   - [http://localhost:3000](http://localhost:3000)

Notes:
- Default frontend API base is `/backend` (rewrite-based proxy to avoid CORS issues)
- If you change `NEXT_PUBLIC_API_BASE_URL` or `BACKEND_ORIGIN`, restart the frontend server

## Contact
- Open an issue in this repository for bugs, feature requests, or questions.

## Room for improvement
- **Async worker pipeline**  
  `cmd/worker` is currently a scaffold. Move long-running/expensive AI tasks to background jobs (e.g., Redis-backed queue) and let API endpoints return job IDs + status polling/webhooks.

- **Prompt lineage UX and deeper lineage features**  
  Lineage is stored on derive (`prompt_lineage`) but can be expanded with:
  - lineage graph view in frontend
  - branch comparison between versions/prompts
  - clearer ancestry metadata in prompt detail and version cards