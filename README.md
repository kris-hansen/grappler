# Grappler

A lightweight orchestration tool for running multiple git worktrees (backend + frontend pairs) simultaneously with port isolation.

## Problem

When working with multiple git worktrees of the same project, you encounter port conflicts:
- Worktree 1 backend wants port 8080 → Worktree 2 backend also wants 8080 → **CONFLICT**
- Worktree 1 frontend wants port 3000 → Worktree 2 frontend wants 3000 → **CONFLICT**

Grappler solves this by dynamically allocating unique ports for each worktree group and injecting them as environment variables.

## Features

- **Git worktree discovery**: Automatically scans repositories and pairs backend/frontend worktrees
- **Dynamic port allocation**: Assigns unique ports to avoid conflicts (8000-8999 for backends, 5000-5999 for frontends)
- **Environment injection**: Injects `SERVER_PORT` and `CONDUCTOR_PORT` environment variables
- **Process management**: Starts, stops, and monitors service processes
- **Log aggregation**: Captures stdout/stderr to separate log files per service
- **Health checking**: Verifies services started successfully
- **Conductor proxy integration**: Works with existing conductor proxy pattern

## Installation

### Install from the remote module

```bash
go install github.com/kris-hansen/grappler/cmd/grappler@latest
```

### Build from source

```bash
go build -o grappler ./cmd/grappler
```

### Install to PATH

```bash
go install ./cmd/grappler
```

## Usage

### 1. Initialize configuration

Scan your repositories to discover worktrees and generate configuration:

```bash
grappler init ~/erebor/core ~/erebor/web
```

This will:
- Scan both repositories for git worktrees
- Intelligently pair backend/frontend worktrees
- Generate `~/.grappler/config.yaml` with discovered groups
- Create `~/.grappler/state.json` for tracking running groups

### 2. Check status

View all configured groups and their status:

```bash
grappler status
```

Output:
```
Grappler Status
================================================================================
GROUP                BACKEND PORT    FRONTEND PORT   STATUS     ACCESS
--------------------------------------------------------------------------------
main                 -               -               stopped    -
  Backend:  main
  Frontend: main

dakar-davis          -               -               stopped    -
  Backend:  feature/ere-5326
  Frontend: feature/ere-6001
```

### 3. Start a group

Start backend and frontend services for a group:

```bash
grappler start main
```

Grappler will:
1. Allocate unique ports (e.g., backend: 8000, frontend: 5000)
2. Start backend with `SERVER_PORT=8000`
3. Start frontend with `CONDUCTOR_PORT=5000`
4. Wait for services to become healthy
5. Display access URLs

Output:
```
Starting group "main"...
  Backend port:  8000
  Frontend port: 5000

Starting backend...
✓ Backend started (PID: 12345)

Starting frontend...
✓ Frontend started (PID: 12346)

Waiting for services to be healthy...
✓ Backend healthy (http://localhost:8000)
✓ Frontend healthy (http://localhost:5000)

==================================================
Group "main" is running

Access frontend via conductor proxy:
  http://5000.port.localhost:3000

Direct access:
  Backend:  http://localhost:8000
  Frontend: http://localhost:5000

==================================================
```

### 4. Stop a group

Stop running services and release ports:

```bash
grappler stop main
```

## How It Works

### Worktree Pairing Logic

Grappler intelligently pairs backend and frontend worktrees:

1. **Conductor workspaces**: Pairs worktrees in `conductor/workspaces/` by matching the worktree name
   - `conductor/workspaces/core/dakar` + `conductor/workspaces/web/davis` → `dakar-davis` group

2. **Main repositories**: Pairs the main worktrees from both repos
   - `~/erebor/core` (main branch) + `~/erebor/web` (main branch) → `main` group

3. **Unpaired worktrees**: Creates backend-only groups for worktrees without a pair

### Port Allocation

- **Backend ports**: 8000-8999 (injected as `SERVER_PORT`)
- **Frontend ports**: 5000-5999 (injected as `CONDUCTOR_PORT`)
- Ports are tracked in `~/.grappler/state.json`
- Ports are released when a group is stopped

### Conductor Proxy Integration

Grappler works with your existing `conductorProxy.cjs`:
- Frontend runs on allocated port (e.g., 5000)
- Access via proxy: `http://5000.port.localhost:3000`
- The proxy forwards requests to `localhost:5000`

## Configuration

Configuration is stored in `~/.grappler/config.yaml`:

```yaml
version: "1"
groups:
  main:
    name: main
    backend:
      directory: /Users/krish/erebor/core
      branch: main
      command: go run cmd/api-server/main.go
    frontend:
      directory: /Users/krish/erebor/web
      branch: main
      command: pnpm conductor:customer
proxy:
  enabled: true
  use_existing_conductor: true
```

### Customizing Groups

You can manually edit the config to:
- Change commands (e.g., use different frontend apps)
- Add environment variables
- Modify port ranges
- Add/remove groups

## Logs

Logs are stored in `~/.grappler/logs/`:
- `<group>-backend.log` - Backend stdout/stderr
- `<group>-frontend.log` - Frontend stdout/stderr

## Architecture

```
grappler/
├── cmd/grappler/          # CLI entry point
├── internal/
│   ├── config/            # Configuration and state management
│   ├── worktree/          # Git worktree scanning and pairing
│   ├── ports/             # Port allocation
│   ├── process/           # Process lifecycle management
│   └── cli/               # Command implementations
```

## Requirements

- Go 1.21+
- Git (for worktree discovery)
- Services must respect `SERVER_PORT` and `CONDUCTOR_PORT` environment variables

## Roadmap

### Phase 2: Polish
- `grappler logs` command with follow mode
- Better health check configuration
- Colored output and progress indicators
- Tab completion

### Phase 3: Enhancements
- tmux/zellij integration
- Watch mode (auto-restart on file changes)
- Resource monitoring
- Config templates

## License

MIT

## Author

Kris Hansen
