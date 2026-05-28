# Baby-Rabbit

A small, in-memory, RabbitMQ-style message broker written in Go and designed
as a textbook example of **Clean Architecture**. Built as the practical part
of the *Microservice Systems Design* (CE222-MS) Assignment 2 at Sharif
University of Technology.

The service exposes a REST/HTTP API for creating named queues, pushing and
popping messages, inspecting status, and listing queues. Messages carry a
TTL and are removed lazily on `Pop` and proactively by a background sweeper.

---

## Features

| Requirement (from the brief) | Where it lives |
|---|---|
| `push` / `pop` | `usecase.QueueUseCase` → `repository.RingBufferQueue` |
| Queue status (`size`, `capacity`) | `GET /queues/:id` → `usecase.QueueUseCase.Status` |
| Configurable capacity per queue | `domain.QueueMetadata.Capacity` |
| Per-message TTL | `domain.Message.TTL` + `service.TTLCleaner` |
| HTTP/REST transport | `internal/delivery/http` |
| In-memory storage | `internal/repository` |

---

## Clean Architecture

The codebase strictly follows the dependency rule: **source code dependencies
only point inward**.

```
┌──────────────────────────────────────────────────────────────┐
│  cmd/server (composition root: wires concrete adapters)      │
└──────────────────────────────────────────────────────────────┘
                              │
              ┌───────────────┼───────────────┐
              ▼               ▼               ▼
    ┌──────────────┐  ┌──────────────┐  ┌────────────────┐
    │  delivery/   │  │  repository/ │  │  service/      │
    │   http       │  │ (RingBuffer, │  │ (TTLCleaner)   │
    │              │  │  QueueMgr)   │  │                │
    └──────┬───────┘  └──────┬───────┘  └────────┬───────┘
           │                 │                   │
           ▼                 ▼                   ▼
    ┌─────────────────────────────────────────────────────┐
    │            usecase  (application logic)             │
    │   QueueService port, IDGenerator port, Clock port   │
    └────────────────────────┬────────────────────────────┘
                             ▼
    ┌─────────────────────────────────────────────────────┐
    │           domain  (entities + ports)                │
    │   Message, Queue, QueueManager, errors, status      │
    └─────────────────────────────────────────────────────┘
```

### Layer responsibilities

1. **`domain/`** — Pure language constructs. Defines the `Message` entity,
   the `Queue` and `QueueManager` ports, the `QueueStatus` value object,
   and the catalogue of business errors (`ErrQueueFull`, `ErrQueueEmpty`,
   `ErrQueueNotFound`, …). No imports of frameworks or third-party
   libraries. Stable, replaceable nowhere.

2. **`usecase/`** — Application use cases. Exposes the inbound port
   `QueueService` consumed by the delivery layer and outbound ports
   `IDGenerator` and `Clock` so the use case does not depend on
   `uuid` or `time.Now` directly. This is where Dependency Inversion
   buys you testability — see the unit tests for proof.

3. **`repository/`** — Adapters that implement the domain ports with a
   thread-safe in-memory ring buffer. Holds no business rules.

4. **`delivery/http/`** — Gin-based adapter. Handlers depend on the
   `usecase.QueueService` interface (never the concrete type), translate
   DTOs ↔ domain types, and map domain errors to HTTP status codes in a
   single place (`writeDomainError`). Swapping Gin for gRPC, Fiber, or
   chi would only touch this directory.

5. **`service/`** — Background workers. `TTLCleaner.Run(ctx)` sweeps
   expired messages on every tick and exits gracefully when the context
   is cancelled.

6. **`pkg/`** — Pure infrastructure: `idgen.UUID`, `clock.Real`, `logger`.
   These are *concrete* and depend on third-party libraries, but the
   inner layers only ever know about them through tiny interfaces.

7. **`cmd/server/`** — The composition root. The only place where every
   concrete type meets. Wires the dependency graph, installs signal
   handlers, runs the HTTP server, and shuts everything down cleanly.

### Why this matters

- **DIP in action.** `QueueUseCase` declares `IDGenerator` and `Clock`
  interfaces *inside the application layer* (input/output ports). The
  outer layer must conform to them, not the other way around.
- **Framework independence.** The domain and use cases never import
  `gin`, `uuid`, or `zap`. You could compile the inner half of the
  binary without any HTTP framework at all.
- **Testability.** `internal/usecase/queue_usecase_test.go` substitutes
  `seqID` for `uuid` and `fixedClock` for the wall clock with no
  ceremony — the seam already exists in the design.
- **Single error vocabulary.** Business failures travel as typed
  `domain.Err*` values from the inner layers and are mapped to HTTP
  status codes exactly once, at the boundary. No string-matching, no
  leaking of HTTP concepts inward.

---

## Project layout

```
Baby-Rabbit/
├── cmd/
│   └── server/
│       └── main.go              # composition root + graceful shutdown
├── internal/
│   ├── domain/                  # entities + ports + errors
│   │   ├── errors.go
│   │   ├── manager.go
│   │   ├── message.go
│   │   └── queue.go
│   ├── usecase/                 # application logic
│   │   ├── port.go              # QueueService / IDGenerator / Clock
│   │   ├── queue_usecase.go
│   │   └── queue_usecase_test.go
│   ├── repository/              # adapter: in-memory ring buffer
│   │   ├── queue_manager.go
│   │   ├── ring_buffer_queue.go
│   │   └── ring_buffer_queue_test.go
│   ├── delivery/
│   │   └── http/                # adapter: Gin HTTP transport
│   │       ├── dto.go
│   │       ├── handler.go
│   │       └── router.go
│   ├── service/
│   │   └── ttl_cleaner.go       # background sweeper
│   └── pkg/
│       ├── clock/clock.go
│       ├── idgen/uuid.go
│       └── logger/logger.go
├── Baby-Rabbit.postman_collection.json
├── go.mod
└── README.md
```

---

## Running

```bash
go run ./cmd/server
# server listens on :8080
```

Run the test suite:

```bash
go test ./...
```

---

## HTTP API

| Method | Path                       | Purpose                              | Success | Errors                           |
|--------|----------------------------|--------------------------------------|---------|----------------------------------|
| GET    | `/healthz`                 | Liveness probe                       | 200     | —                                |
| POST   | `/queues`                  | Create a queue                       | 201     | 400 (bad input), 409 (duplicate) |
| GET    | `/queues`                  | List queues                          | 200     | —                                |
| GET    | `/queues/:queue`           | Queue status (`size`, `capacity`)    | 200     | 404                              |
| POST   | `/queues/:queue/push`      | Enqueue a message (with TTL seconds) | 202     | 400, 404, 409 (full)             |
| POST   | `/queues/:queue/pop`       | Dequeue next non-expired message     | 200     | 204 (empty), 404                 |

### Examples

```bash
# create
curl -sX POST localhost:8080/queues \
  -H 'Content-Type: application/json' \
  -d '{"name":"orders","capacity":100}'
# {"id":"…","name":"orders","capacity":100}

# push with 60s TTL (0 = never expires)
curl -sX POST localhost:8080/queues/<id>/push \
  -H 'Content-Type: application/json' \
  -d '{"value":"hello","ttl":60}'

# status
curl -s localhost:8080/queues/<id>
# {"id":"…","name":"orders","size":1,"capacity":100}

# pop
curl -sX POST localhost:8080/queues/<id>/pop
# {"id":"…","value":"hello","created_at":"…"}

# pop again → 204 No Content
```

A Postman collection is included: `Baby-Rabbit.postman_collection.json`.

---

## Concurrency & lifecycle

- Each `RingBufferQueue` is guarded by a `sync.Mutex`.
- `QueueManager` uses an `RWMutex` so reads scale.
- `Pop` is **non-blocking** by design: an HTTP request must not hang
  forever waiting on a message — clients should poll or retry.
- The TTL sweeper runs in a goroutine, takes a `context.Context`, and
  stops cleanly on `SIGINT` / `SIGTERM`. The HTTP server is shut down
  with a 5-second grace window.

---

## Third-party packages

| Package | Why |
|---|---|
| `github.com/gin-gonic/gin` | Minimal HTTP router used only in the delivery adapter. |
| `github.com/google/uuid`   | UUID v4 generation; isolated behind the `IDGenerator` port. |
| `go.uber.org/zap`          | Structured logging, used only in the outer infrastructure. |
