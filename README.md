# Eudaimonia

A decentralized, Discord-style chat platform: communities hosted on independently run
**nodes**, users identified by their own keys, voice through a per-node LiveKit SFU.

The architecture research and the language decision (Go backend, React frontend) are in
[docs/RESEARCH.md](docs/RESEARCH.md). The code is currently a skeleton: the development
environment, container images, a Go node with health endpoints, and the React app.

## Layout

```
backend/    Go node: cmd/server, internal/{config,httpapi}, migrations/, Dockerfile
frontend/   React 19 + TypeScript + Vite app, nginx image for production
docs/       Research and decision records
devenv.nix  Reproducible dev environment (Go, Node, Postgres, LiveKit, linters)
compose.yaml  Full stack in Docker: postgres, livekit, backend, frontend
```

## Develop

Requires [Nix](https://nixos.org) and [devenv](https://devenv.sh); `direnv allow` is
optional and activates the shell automatically.

```bash
devenv shell
```

```bash
devenv up
```

`devenv up` starts Postgres 18 (127.0.0.1:5432), LiveKit in dev mode (127.0.0.1:7880), the
backend with live reload (http://localhost:8080) and Vite (http://localhost:5173, which
proxies `/api` and `/ws` to the backend). Inside the shell:

| Command | Purpose |
|---|---|
| `backend-test` | gofmt, go vet, go test |
| `frontend-build` | type-check and bundle the React app |
| `db-migrate` / `db-status` | run / inspect goose migrations |
| `lint-docker` | hadolint on both Dockerfiles |
| `devenv test` | all of the above |

Copy `.env.example` to `.env` to override any setting; every value has a dev default.

## Run the stack in Docker

```bash
docker compose up --build
```

Frontend on http://localhost:3000, backend on http://localhost:8080, LiveKit on
ws://localhost:7880. The backend image is a static binary on distroless (non-root); the
frontend image is nginx serving the Vite bundle and proxying `/api` and `/ws`.

## Status

- Backend endpoints: `GET /healthz`, `GET /readyz`, `GET /api/v1/node`; `GET /ws` returns
  501 until the event protocol exists.
- No database schema yet; the migrations directory is empty by design.
- Both Dockerfiles pass hadolint and `compose.yaml` passes `docker compose config`, but no image
  has been built on this machine (Docker is not installed here).
