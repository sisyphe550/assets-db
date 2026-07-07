# AGENTS.md

## Cursor Cloud specific instructions

### Repository state
This repo is currently **design-only**. It contains planning documents
(`backend/doc/01-desgin.md`, `backend/doc/02-plan.md`) and empty placeholders
(`README.md`, `backend/docker-compose-env.yml`). There is **no source code, no
`go.mod`, and no `docker-compose.yml` yet**. The planned system (FAMS) is a
Go/`go-zero` microservices backend to be implemented in phases P0–P9 (see
`backend/doc/02-plan.md`). Until P0 lands there is no application to build/run;
only the toolchain below is set up.

### Toolchain available in this environment
- **Go 1.22.2** (`/usr/bin/go`) — satisfies the design's `Go >= 1.22` requirement.
- **goctl 1.10.1** — the `go-zero` code generator, installed to `$(go env GOPATH)/bin`.
- **Docker 29.6.1** + **compose v5** — for the design-mandated infra (Postgres, MySQL, Mongo, Redis, Kafka, etcd, Jaeger, Prometheus, Grafana).
- **Node 22** — for the planned Univer-based collaborative spreadsheet frontend.

### Non-obvious startup caveats
- **`goctl` on PATH**: `$(go env GOPATH)/bin` is appended to `PATH` in `~/.bashrc`.
  In a non-login/non-interactive shell run `export PATH="$PATH:$(go env GOPATH)/bin"` first.
- **Docker daemon is NOT auto-started** (no systemd in this container). Start it manually once per VM session and give it a few seconds:
  ```bash
  sudo nohup dockerd > /tmp/dockerd.log 2>&1 &
  sleep 8
  sudo docker info | grep -i "storage driver"   # expect: fuse-overlayfs
  ```
  Docker commands need `sudo`.
- **Docker storage driver**: `/etc/docker/daemon.json` is preconfigured to use
  `fuse-overlayfs` with `containerd-snapshotter` disabled (required for Docker 29
  in this Firecracker VM). Do not switch to `overlay2`.

### Once code exists (P0+)
Per `backend/doc/02-plan.md`, infra is started with docker compose (files created in P0):
```bash
docker compose -f deploy/docker/docker-compose.yml --env-file deploy/docker/docker-compose-env.yml up -d
```
Go services then run individually via `go run service/<svc>/{api,rpc}/...`.
`go test ./pkg/... ./service/...` runs unit tests; integration/e2e tests use
`-tags=integration` / `-tags=e2e` and require the infra stack running. Do not put
`docker compose up` in the startup/update script — start it manually as a service step.
