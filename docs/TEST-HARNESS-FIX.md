# Mountebank Test Harness Compatibility Fix

## Problem

The mountebank test harness (`mbTest`) was failing during initialization with this error:

```
Error: Command failed: /home/tetsujinoni/work/go-tartuffe/bin/tartuffe-wrapper.sh stop --pidfile mb.pid
2026/01/16 02:03:07 failed to read pid file: open mb.pid: no such file or directory
Node.js v25.2.1
mb failed to start
```

### Root Cause

The mountebank test framework uses a `restart` workflow:

1. **`restart` command** is called (in `tasks/mb.js`)
2. This internally calls **`stop`** first (line 98: `await stop()`)
3. Then calls **`start`** (line 99: `await start()`)

The problem: On first run or after clean shutdown, the pidfile doesn't exist yet. When `stop` is called, tartuffe's stop command would:
- Try to read the non-existent pidfile
- Fail with `log.Fatalf()` → exit code 1
- This causes the test harness to abort initialization

## Solution

Modified `cmd/tartuffe/main.go` (`runStop()` function) to exit successfully when there's nothing to stop:

### Changes Made

1. **Graceful handling of missing pidfile:**
   ```go
   if os.IsNotExist(err) {
       fmt.Println("no pidfile found, nothing to stop")
       os.Exit(0)  // Exit successfully
   }
   ```

2. **Graceful handling of already-stopped process:**
   ```go
   if err == os.ErrProcessDone {
       fmt.Printf("process %d already stopped\n", pid)
       os.Remove(*pidFile)
       os.Exit(0)  // Exit successfully
   }
   ```

3. **Cleanup pidfile after successful stop:**
   ```go
   os.Remove(*pidFile)
   fmt.Printf("stopped mountebank process %d\n", pid)
   ```

## Impact

### Before Fix
- Test harness would fail during initialization
- **0 tests executed** - complete failure
- Error: "mb failed to start"

### After Fix
- Test harness runs successfully
- **66 tests passing**
- **186 tests failing** (expected - these are actual compatibility gaps)
- Test execution time: ~1 minute

## Compatibility

This change matches mountebank's behavior:
- `mb stop` with no pidfile → exit 0 (success)
- Goal: "If there's nothing to stop, that's fine"

The fix enables the test harness to work properly while maintaining backward compatibility with normal usage.

## Testing

### Verification Steps

1. **Manual test - missing pidfile:**
   ```bash
   ./bin/tartuffe stop --pidfile nonexistent.pid
   echo "Exit code: $?"
   # Output: "no pidfile found, nothing to stop"
   # Exit code: 0
   ```

2. **Wrapper script test:**
   ```bash
   /home/tetsujinoni/work/go-tartuffe/bin/tartuffe-wrapper.sh stop --pidfile nonexistent.pid
   echo "Exit code: $?"
   # Output: "no pidfile found, nothing to stop"
   # Exit code: 0
   ```

3. **Full mountebank test suite:**
   ```bash
   cd /home/tetsujinoni/work/mountebank
   npm run test:api
   # Result: 66 passing (1m), 186 failing
   ```

4. **Go test suite (regression check):**
   ```bash
   go test ./internal/... ./cmd/...
   # Result: All tests pass
   ```

## Related Files

- **Fixed:** `cmd/tartuffe/main.go` (runStop function)
- **Related:** `bin/tartuffe-wrapper.sh` (already had `|| true` workaround)
- **Test harness:** `mountebank/tasks/mb.js` (restart workflow)

## Commit

- **Commit hash:** 4170bfa
- **Branch:** feat/missing-backlog
- **Message:** "fix: make stop command exit successfully when pidfile doesn't exist"

## Next Steps

Now that the test harness works, we can:
1. Use it to validate mountebank compatibility
2. Identify specific test failures that need fixing
3. Track progress as we implement Phase 2 features
4. Compare go-tartuffe's behavior against mountebank's expected behavior

The 186 failing tests represent actual compatibility gaps to address, not harness issues.
