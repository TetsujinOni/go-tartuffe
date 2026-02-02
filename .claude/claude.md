# go-tartuffe Project Guidelines

## Overview
Go implementation of mountebank service virtualization.
**Status**: 92.1% compatibility (232/252 mountebank API tests)

## Quick Validation

```bash
# Build and run Go tests
go build -o bin/tartuffe ./cmd/tartuffe
go test ./test/integration/... -v

# Run mountebank API tests
cd /home/tetsujinoni/work/mountebank
pkill -f tartuffe 2>/dev/null || true
MB_EXECUTABLE=/home/tetsujinoni/work/go-tartuffe/bin/tartuffe-wrapper.sh npm run test:api
```

## Key Directories

```
internal/imposter/     # Core implementation (inject.go, matcher.go, proxy.go, behaviors.go)
internal/models/       # Data models (imposter.go, stub.go)
internal/api/handlers/ # HTTP API handlers
test/integration/      # Integration tests (228+ tests)
bin/tartuffe-wrapper.sh # Wrapper for mountebank tests
```

## Known Issues

**Port conflicts**: Kill stale processes before tests
```bash
pkill -f tartuffe 2>/dev/null || true
```

**pkill exit codes**: Exit 144 = success (processes killed). Always use `|| true` after pkill.

## Won't Fix (20 tests)
- shellTransform (security risk)
- Async injection (goja limitation)
- Replayable export, HTTPS key creation, old proxy syntax
- TCP behaviors with composition

## Commit Format
```
<type>: <subject>
Co-Authored-By: Claude <noreply@anthropic.com>
```
Types: feat, fix, docs, test, refactor, perf, chore
