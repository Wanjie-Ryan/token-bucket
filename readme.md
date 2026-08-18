# Token Bucket Rate Limiter

A standalone Go + Echo rate limiting service, built to learn what breaks when rate limiting moves from a single instance to multiple replicas behind a load balancer, and how atomic Redis Lua scripting fixes it.

Public contract: `POST /check` with a client key, returns `{allowed: bool, remaining: int}`.

## Architecture at a glance

- **Redis** — shared state store every rate limiter instance eventually hits.
- **Nginx** — load balancer, spreads requests across N replicas.
- **Algorithm** — token bucket (allows burst up to capacity, smooths to a steady average rate), made atomic via a Redis Lua script.

## Build phases

### Phase 1 — Docker infrastructure ✅ done

`docker-compose.yml` defines two services, `redis` (`redis:7-alpine`) and `nginx` (`nginx:alpine`), joined to an explicit custom bridge network (`rl-network`) so containers can resolve each other by service name once more services join later. Nginx bind-mounts a local `nginx.conf` over the image's default config; for now that config is just a `/healthz` stub proving the container boots correctly — the real `upstream`/`proxy_pass` load-balancing config comes in Phase 5.

Verified: `curl http://localhost:8080/healthz` → `nginx up`, `docker exec rl_redis redis-cli ping` → `PONG`.

### Phase 2 — Rate limiter service skeleton ✅ done

Single instance, in-memory, fixed-window counter. Proved the HTTP shape — Echo, the `App` struct wiring, a `Limiter` interface every algorithm implements — before Redis or the real token-bucket algorithm entered the picture. Lives on at `POST /check/fixed-window`, never replaced.

### Phase 3 — Naive Redis integration ✅ done

Read-then-write against Redis, deliberately not atomic. A 20-request k6 burst against a limit of 5 let **all 20** through — the check-then-act race, caused by the gap between `GET` and `SET`, not by Redis itself (Redis is single-threaded, every individual command is atomic). Lives on at `POST /check/naive-redis`.

### Phase 4 — Redis Lua script fix ✅ done

Token bucket algorithm executed atomically inside a Lua script (`app/ratelimiter/scripts/token_bucket.lua`), replacing the naive check-then-act. Same 20-request burst now lands on exactly `5` allowed, deterministically, every run — atomicity closes the race completely rather than just reducing it. Lives on at `POST /check/token-bucket`.

### Phase 5 — Multi-instance behind nginx ✅ done

Three named replicas (`app1`/`app2`/`app3`), nginx now doing real round-robin `proxy_pass` load balancing instead of the Phase 1 stub, no app container ports published — nginx is the only reachable entry point. Proved the payoff directly: the same burst test through nginx let ~15 through on `/check/fixed-window` (in-memory state, private per replica) but held at exactly `5` on `/check/token-bucket` (shared Redis state, atomic regardless of which replica handled the request).

### Phase 6 — Failure modes ✅ done

Decided fail-closed (deny requests) when Redis is unreachable, over fail-open — informed by real production experience where uncontrolled Redis outages caused panics elsewhere. Distinguished a deliberate fail-closed *policy* from an accidental fail-*crashed* bug (an unhandled panic, like the earlier missing-`InitRedis()` bug) — `middleware.Recover()` guards against the latter regardless of which policy is chosen.

### Phase 7 — Dashboards & log correlation ✅ done

- **Structured logging** — OpenTelemetry tracer (local ID generation, no exporter) + `logrus.WithContext` + a `RequestLogger` middleware logging every request with `trace_id`/`span_id`/`status`/`latency_ms`, mirroring identity-service's `FullTransactionLogger` pattern.
- **Metrics** — `echoprometheus` for automatic HTTP metrics, plus a custom `rate_limiter_checks_total{limiter, result}` counter tracking allowed/denied/redis_error per algorithm, exposed at `/metrics`.
- **Prometheus + Grafana** — Prometheus scrapes each replica's `/metrics` directly (never through nginx — metrics are per-process, same "no shared memory" lesson as Phase 2/5 showing up again), Grafana's Prometheus datasource auto-provisioned via `grafana/provisioning/datasources/`, first dashboard panel built querying `sum(rate_limiter_checks_total) by (result)`.
- **Uptime Kuma** — external HTTP prober hitting `http://nginx/healthz` every 30s, tracking uptime % and response time, independent of the metrics/logs pipeline.
- **Elasticsearch + Kibana + Filebeat** — Filebeat autodiscovers every container via the Docker socket and an unconditional template (the "hints" autodiscover approach didn't reliably generate configs, switched to a plain template match), ships logs into Elasticsearch, searchable in Kibana's Discover view.

## Key lessons this project actually taught

- **In-memory rate limiting has a visibility problem**, not an atomicity problem — each replica's own memory can't see another replica's traffic. Fixed by moving state to Redis.
- **Shared state introduces a different problem: atomicity.** Redis itself is single-threaded and atomic per-command; the danger is in *our own code* doing multiple uncoordinated commands (check, then act) with a gap between them that concurrent requests can race into.
- **Lua scripting closes that gap completely**, not partially — the entire check-and-write runs as one unit of work on Redis's single execution thread, so the fix is deterministic, not merely "less likely to fail."
- **Horizontal scaling (nginx + N replicas) is exactly the scenario that exposes both problems at once** — and is also the proof that the Redis+Lua fix actually works, since correctness held regardless of which replica handled a given request.
- **Fail open vs fail closed is a real design decision, not a default** — and is separate from making sure failures are *handled* at all (a panic is neither "open" nor "closed," it's a bug).
- **Metrics are per-process, same as in-memory rate-limiter state was** — scraping through a load balancer gives inconsistent results; Prometheus scrapes every replica directly and aggregates in queries instead.

## Repo workflow

`master` is protected — no direct pushes, all changes go through a PR from `dev`. Core functionality (phases 1–7) landed on `master` before this rule was added.

### MUTEXES

- A mutext (mutual exclusion lock) guarantees that only one goroutine can be inside a protected section of code at a time.
- Everyone else who tries to enter has to wait their turn.

![Mutex illustration](illustrations/mutex.png)

### K6

- open source load testing tool, where you write test scripts in JS/TS to simulate real traffic against API
- Fire a burst of concurrent requests at your rate limiter and watch what breaks.

## Naive-redis

- The race condition does not exist in redis, redis itself is single threaded, and executes each command atomically, with no interleaving.
- Race condition exists entirely in the code; the gap btn GET finishing and SET starting is where another goroutine's GET can sneak in.
- Redis did what it was asked; we just asked it two separate, uncoordinated things instead of one atomic thing.
- We fix this by introducing **LUA** which makes our check-and-write into a single atomic value, so there's no gap left to race into.

- If requests for the same key arrived on at a time **(ideal world/scenario)** spaced out even slightly, this bug would never show up - each pair GET/SET would complete cleanly before the next one started, and the limiter would work correctly.
- It only breaks when multiple requests check then write; windows windows overlap in time
- This bug is dangerous in prod; it can pass every normal test and manual check, and only reveals itself the moment real concurrent traffic hits.

## TOKEN BUCKET

- Each client key gets a "bucket" holding some number of tokens, capped at a CAPACITY.
- Tokens refill continuously at a fixed **refill_rate (tokens/second)**
- A request costs 1 token: if the bucket has >=1, allow request, and deduct 1; otherwise deny.
- Capacity governs how big a burst you can abosrb all at once; refill_rate governs the steady-state average once the busrt is spent.
- Refill is computed lazily on read, at request time:
- ## elapsed = now - last_refill AND tokens = min(capacity, tokens +elapsed \* refill_rate)
- compute that, then decide allow/deny, then store the updated tokens and last_refill = now.
- This is also why it doesn't have fixed-window boundary-busrt flow; there's no hard reset at a clock edge, tokens trickle back continously.

## why LUA actually closes the race this time.

- There was a bug between GET and SET - two separate round trips, each indivisually atomic; but not atomic together.
- A lus script sent via EVAL doesn't have that gap; Redis runs the entire script - read state, do the math, decide, write state - as a single unit of work on its one execution thread.
- No other command can interleave this, the "check" and the "act" stop being two ops and become one ops.

**Running K6 - k6 run k6-naive-redis-race.js**

## Scaling running instances using Docker and NGINX as reverse proxy.

- The purpose of a rate limiter is to know the total count across every instance for a given client, which is precisly the kind of state that can't live in any one instance's private memory once you've scaled out.
- So can this Redis enhanced with LUA, can it survive an architectural change? Going from 1 process to N?

**Purpose of NGINX**

- Nginx sits in front of the 3 app containers as reverse proxy.

1. Client sends a request to nginx :8080, not any app container directly, the app containers have no published host ports, so nginx is the only reachable entry point.
2. For each incoming request, nginx picks one backend from the upstream list. With no algorithm specified, default is round-robin; request 1 -> app 1, request 2 -> app2, request 3 -> app3, request 4 -> app 1 again, cycling forever.

**proxy_pass**

- tells nginx: open a new connection to whichever backend was picked, forward the client's request to it, wait for that backend's response, then relay that response back to original client.
- Client never knows which of the 3 actually handled it.
- Selection is stateless per request - nginx isn't tracking "this client talked to app2 last time, keep sending it there". Every new request just gets the next name in the rotation, regardless of who's asking.

## LOGGING AND METRICS

## LOGS -> Kibana

- Detailed, per-event record: at this exact moment, this request came in, took this long, returned this status.
- Kibana is just a search/visualization UI, that sits on top of ElasticSearch which is a storage, which is the actual storage and search engine.

## Metrics -> Prometheus -> Grafana

- Aggregated numbers overtime. "how many requests were allowed vs denied in the last 5 minutes" "whats p95 latency right now"
- Prometheus is a pull-based metrics collector - it periodically scrapes a /metrics endpoint your app exposes and stores the numbers as a time series.
- Grafana is the dahsboard layer on top - it queries prometheus and draws graph.

## Uptime -> uptime kuma

- Simplest of the 4, and independent of the 3. Its an external prober; it periodically hits a URL (nginx :8080/healthz) from outside the app and tracks "was it reachable" how fast did it respond, whats the uptime percentage overtime.
- Used for availability monitoring; is this URL reachable right now? how fast did it respond, and whats the uptime % been over time.
- You can wire uptime to alert you when service is down

- /etc/grafana/provisioning/datasources/ — for data source definitions
- /etc/grafana/provisioning/dashboards/ — for dashboard definitions
- /etc/grafana/provisioning/alerting/ — for alert rules
