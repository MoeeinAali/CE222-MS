# Mini-NginX

Mini-NginX is a lightweight, configurable HTTP server written in Go. It supports four route behaviors from a YAML config file:

- Redirect requests to another URL
- Reverse proxy requests to a backend TCP/HTTP service
- Serve a single static file
- Serve a directory (including simple directory listing)

It is designed as a learning-friendly, minimal implementation of core reverse-proxy/web-server concepts.

## Features

- Config-driven routing via `config.yml`
- Multiple route types in one server instance
- Concurrent connection handling (goroutine per accepted connection)
- Basic structured logging to console and `server.log`
- No external web framework

#### Servedir Picture:

![](static/img.png)

## Architecture

### Component Diagram

```mermaid
graph TD
    A["Client Requests<br/>(HTTP Clients)"]
    B["TCP Listener<br/>(internal/server)<br/>Port: 8080"]
    C["Connection Handler<br/>(goroutine/conn)"]
    D["Router<br/>(internal/router)<br/>• Parse request line<br/>• Match route prefix<br/>• Select handler"]
    
    E[" Redirect Handler<br/>→ 302 Found"]
    F["ReverseProxy Handler<br/>→ Forward to backend"]
    G["StaticFile Handler<br/>→ Serve File"]
    H["DirServe Handler<br/>→ HTML listing"]
    
    I["Backend TCP<br/>(external service)"]
    J["HTTP Response<br/>to Client"]
    
    A -->|accepts| B
    B -->|per conn| C
    C --> D
    D -->|type: redirect| E
    D -->|type: reverseproxy| F
    D -->|type: staticfile| G
    D -->|type: servedir| H
    
    E --> J
    F --> I
    I -->|response| J
    G --> J
    H --> J
```

### Runtime Flow

The runtime flow is intentionally simple and layered:

1. **Bootstrap** (`cmd/server/main.go`)
   - Loads YAML config
   - Initializes logger
   - Creates and starts server
2. **Server lifecycle** (`internal/server/server.go`)
   - Opens TCP listener on configured port
   - Accepts client connections in a loop
   - Spawns a goroutine for each connection
3. **Routing** (`internal/router/router.go`)
   - Parses request line (`METHOD PATH HTTP/...`)
   - Matches request path against configured route prefixes
   - Dispatches to route handler by type
4. **Handlers** (`internal/handlers/*.go`)
   - Execute route-specific behavior (`redirect`, `reverseproxy`, `staticfile`, `servedir`)

### Request Flow (High Level)

`Client -> TCP Listener -> Router -> Handler -> HTTP Response`

## Design Patterns Used

This project uses a few practical patterns commonly found in backend systems:

- **Front Controller (lightweight)**
  - `router.Handle` centralizes request parsing and dispatch.
- **Strategy-style dispatch**
  - Route `type` in config selects handler behavior at runtime.
- **Configuration-as-code / Declarative routing**
  - Behavior is declared in YAML instead of hardcoded route setup.
- **Dependency injection (manual, simple)**
  - `*config.Config` is passed into server/router path instead of global config state.
- **Concurrency per connection**
  - Each accepted connection is processed in its own goroutine.

## Project Structure

```text
Mini-NginX/
├── cmd/
│   └── server/
│       └── main.go              # Entrypoint: load config, init logger, start server
├── internal/
│   ├── config/
│   │   └── config.go            # YAML schema + loader
│   ├── handlers/
│   │   ├── redirect.go          # 302 redirect responses
│   │   ├── reverseproxy.go      # Forwards request to backend and streams response
│   │   ├── servedir.go          # Directory serving/listing
│   │   └── staticfile.go        # Serves one file with detected MIME type
│   ├── logger/
│   │   └── logger.go            # Logger init and Info helper
│   ├── router/
│   │   └── router.go            # Request line parsing + route dispatch
│   └── server/
│       └── server.go            # TCP listener loop + connection handling
├── config.yml                   # Route and port configuration
├── go.mod
├── go.sum
└── server.log                   # Log output (created/overwritten at startup)
```

## Configuration

Example `config.yml`:

```yaml
listen-on: 8080
paths:
  "/go-to-google":
    type: redirect
    target: "https://google.com"
  "/chat/":
    type: reverseproxy
    target: "localhost:3000"
  "/file.pdf":
    type: staticfile
    target: "/absolute/path/to/file.pdf"
  "/downloads/":
    type: servedir
    target: "/absolute/path/to/directory/"
```

### Route Types

- `redirect`: returns `302 Found` with `Location` header to `target`
- `reverseproxy`: opens TCP connection to `target`, forwards request, relays backend response
- `staticfile`: reads file from `target` and serves it as HTTP response
- `servedir`: serves files under `target`; if request maps to a folder, returns HTML listing

## How to Start

### Prerequisites

- Go installed (project module uses Go 1.24.x in `go.mod`)

### 1) Configure routes

Edit `config.yml` to match your local files/directories and backend targets.

### 2) Run the server

```bash
go run ./cmd/server
```

Server starts on `listen-on` port from `config.yml`.

### 3) Try sample requests

```bash
curl -i http://localhost:8080/go-to-google
curl -i http://localhost:8080/file.pdf
curl -i http://localhost:8080/downloads/
```

If you configured reverse proxy:

```bash
curl -i http://localhost:8080/chat/
```

## Build Binary (Optional)

```bash
go build -o mini-nginx ./cmd/server
./mini-nginx
```

## Notes and Limitations

- This is a minimal HTTP parser (request line + headers forwarding in proxy mode).
- Route matching is prefix-based via `strings.HasPrefix`.
- `server.log` is truncated on each startup.
- Error handling and security hardening are intentionally minimal for simplicity.
