# go-tartuffe Port Worklog

A timeline and summary of the AI-assisted port of mountebank from Node.js to Go.

## Project Overview

- **Source Project**: [mountebank](https://www.mbtest.dev) (Node.js)
- **Target**: go-tartuffe (Go 1.25)
- **AI Tool**: Claude Code with Claude Opus 4.5
- **Duration**: November 27 - December 2, 2025 (6 days)

## Output Statistics

| Metric | Count |
|--------|-------|
| Go source files | 71 |
| Lines of production code | ~12,238 |
| Lines of test code | ~9,752 |
| Total lines of Go | ~22,000 |
| Integration test files | 19 |

## Session Timeline

### Day 1: November 27, 2025
**Session**: Project initialization

- Initial project setup
- Created basic Go module structure
- Exploration of mountebank codebase for porting strategy

---

### Day 2: November 28, 2025
**Sessions**: Main development (3.7MB of conversation data, ~1,031 exchanges)

**Morning Session (07:53 - 08:27 UTC)**
- Created CLAUDE.md guidance for codebase
- Phase 1: Standard HTTP imposter implementation
- Phase 2: API endpoints and basic functionality

**Afternoon Session (14:11 - 17:40 UTC)**
- Gap analysis between mountebank and go-tartuffe
- Implemented persistence features (datadir, filesystem serialization)
- Added CLI options and EJS template support
- Implemented formatter and datadir CLI options

**Evening Session (17:41 - 23:31 UTC)**
- Created PROGRESS.md for tracking implementation status
- Implemented proxy responses (proxyOnce, proxyAlways, proxyTransparent)
- Added inject responses with JavaScript execution (goja engine)
- Implemented fault responses (CONNECTION_RESET_BY_PEER, RANDOM_DATA_THEN_CLOSE)
- Added behaviors: wait, copy, lookup, decorate, shellTransform, repeat
- Implemented binary mode (base64 encoding/decoding)
- Added TCP protocol support with text/binary modes
- Implemented selectors (JSONPath, XPath)
- Added predicate options (caseSensitive, keyCaseSensitive, except)
- Implemented HTTPS imposter support with certificate handling
- Added SMTP protocol support

**Late Night Session (22:50 - 23:31 UTC)**
- Designed and implemented plugin interface architecture
- Added support for custom protocol plugins (out-of-process and Go plugins)
- Implemented repository plugins for custom persistence backends

---

### Day 3: November 29, 2025
**Sessions**: Testing and gap closure (continued from Nov 28)

**Morning Session (00:06 - 04:43 UTC)**
- Comprehensive gap analysis using mountebank mbTest suite
- Fixed high and medium priority gaps
- Resolved failing tests
- Integration test improvements

**Features Completed**:
- gRPC protocol with dynamic proto loading
- All RPC types: unary, server streaming, client streaming, bidirectional
- gRPC reflection support
- Metadata matching for gRPC
- Full behavior support for gRPC (wait, copy, decorate, lookup, shellTransform)

---

### Day 4: November 30, 2025
**Session**: Stabilization and testing (9.7MB of conversation data, ~2,629 exchanges)

- Final gap resolution
- Test suite stabilization
- All integration tests passing
- Documentation updates to PROGRESS.md

---

### Day 5: December 1, 2025
- Session cleanup and consolidation
- Minor refinements

---

### Day 6: December 2, 2025
**Session**: Production readiness (current)

- Created Dockerfile with multi-stage build
- Added .dockerignore for optimized builds
- Implemented GitHub Actions CI pipeline (test, lint, build, docker)
- Comprehensive gap analysis for GitHub publication
- Created .gitignore
- Expanded README.md with full documentation
- Added CONTRIBUTING.md
- Added SECURITY.md
- Created this PORT-WORKLOG.md

## Features Implemented

### Protocols (5)
- HTTP - Full request/response handling
- HTTPS - TLS with auto-generated or custom certs, mutual TLS
- TCP - Raw TCP mocking with text/binary modes
- SMTP - Email capture and recording
- gRPC - Dynamic proto loading, all RPC types, reflection

### Predicates (11)
equals, deepEquals, contains, startsWith, endsWith, matches, exists, and, or, not, inject

### Predicate Options (5)
caseSensitive, keyCaseSensitive, except, xpath, jsonpath

### Response Types (4)
is, proxy, inject, fault

### Proxy Modes (3)
proxyOnce, proxyAlways, proxyTransparent

### Behaviors (6)
wait, repeat, copy, lookup, decorate, shellTransform

### CLI Commands (4)
start, stop, save, replay

## Architecture Decisions

1. **JavaScript Engine**: Used [goja](https://github.com/dop251/goja) for JavaScript injection support (predicates, responses, behaviors)

2. **gRPC Support**: Used [protocompile](https://github.com/bufbuild/protocompile) for runtime .proto parsing instead of requiring protoc

3. **Plugin System**: Dual support for:
   - Out-of-process plugins via stdin/stdout protocol (mountebank compatible)
   - In-process Go plugins via `plugin.Open()` for better performance

4. **Repository Abstraction**: Pluggable storage backends (memory, filesystem, custom)

## Conversation Statistics

| Session | Date | Size | Exchanges |
|---------|------|------|-----------|
| Init | Nov 27 | 11KB | ~15 |
| Main Dev | Nov 28 | 3.7MB | ~1,031 |
| Testing | Nov 28-30 | 9.7MB | ~2,629 |
| Production | Dec 2 | 299KB | ~176 |
| **Total** | - | **~13.7MB** | **~3,851** |

## Key Learnings

1. **Iterative Development**: Building features incrementally with continuous testing was essential

2. **Reference Implementation**: Having the mountebank source and test suite locally enabled accurate compatibility

3. **Context Continuation**: Multiple context window exhaustions required session summaries for continuity

4. **Gap-Driven Development**: Regular gap analyses against the reference implementation guided priority

## Compatibility Target

mountebank v2.9.3 API compatibility - designed as a drop-in replacement for HTTP protocol testing with improved performance from compiled Go binary.
