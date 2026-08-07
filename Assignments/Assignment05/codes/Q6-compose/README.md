# Q6 — Compose

```
host :8080 ──> app (composeprac) ──edge-net──> nginx ──helper-net──> helper
                      └──────────redis-net──> redis
```

## Run

```bash
# 1) build the provided java image and tag it `composeprac`
docker build -t composeprac compose-challenge

# 2) bring the stack up (builds the helper image on the fly)
docker compose up -d

# 3) hit the endpoint, then read the two log lines
curl -s -X POST http://localhost:8080/call-me
docker compose logs app --no-log-prefix
```

## Constraints

| Constraint | How it is satisfied |
|---|---|
| only the java app is port-mapped | `ports:` appears only under `app` |
| the word `environment` is not used | container env comes from `env_file:` (`env/app.env`, `env/redis.env`) |
| nginx must sit between java and helper | `app` and `helper` share **no** network; `nginx` is the only member of both `edge-net` and `helper-net` |
| redis is required at startup | `depends_on: redis: {condition: service_healthy}` |

`.env` (compose interpolation) and `env/*.env` (`env_file:`) are two different
mechanisms — see the report.
