# Security Considerations

This document outlines security decisions and limitations in go-tartuffe.

## Disabled Features

### shellTransform Behavior

**Status**: ❌ Not Supported

**Reason**: Security Risk - Arbitrary Command Execution

The `shellTransform` behavior from mountebank allows executing arbitrary shell commands to transform responses. This feature is **intentionally disabled** in go-tartuffe because:

1. **Command Injection Risk**: Allows execution of arbitrary system commands
2. **Privilege Escalation**: Commands run with the same privileges as the tartuffe process
3. **System Access**: Unrestricted access to filesystem, network, and system resources
4. **Attack Surface**: Increases attack surface for malicious imposter configurations

#### Mountebank Example (Not Supported)

```json
{
  "responses": [{
    "is": { "body": "Hello" },
    "behaviors": [{
      "shellTransform": "node transform.js"
    }]
  }]
}
```

#### Alternative: Use JavaScript Decorate Behavior

Instead of `shellTransform`, use the `decorate` behavior which runs JavaScript in a sandboxed environment:

```json
{
  "responses": [{
    "is": { "body": "Hello" },
    "behaviors": [{
      "decorate": "function(request, response) { response.body = response.body.toUpperCase(); }"
    }]
  }]
}
```

**Benefits of `decorate`**:
- ✅ Sandboxed JavaScript execution (goja VM)
- ✅ No system command access
- ✅ Same transformation capabilities for most use cases
- ✅ Access to request/response/state objects
- ✅ JSON, string, and data manipulation

#### Migration Guide

| shellTransform Use Case | decorate Alternative |
|------------------------|---------------------|
| Transform response body | Use JavaScript string/JSON methods |
| Add computed headers | Modify `response.headers` object |
| Conditional responses | Use JavaScript conditionals |
| External data lookup | Use `lookup` behavior with CSV files |
| Complex transformations | Use JavaScript with JSON.parse/stringify |

**Example Migration**:

Before (shellTransform):
```json
{
  "behaviors": [{
    "shellTransform": "jq '.data | length'"
  }]
}
```

After (decorate):
```json
{
  "behaviors": [{
    "decorate": "function(req, res) { var data = JSON.parse(res.body); res.body = JSON.stringify(data.length); }"
  }]
}
```

## Supported Security Features

### JavaScript Injection Sandboxing

✅ JavaScript code runs in goja (ES5.1) VM
✅ No access to Node.js `require()` or filesystem
✅ No access to `process.env` (intentional limitation)
✅ Limited to safe JavaScript operations

### API Key Authentication

✅ Optional API key authentication via `--apikey` flag
✅ Validates `X-Api-Key` header on all requests

### IP Whitelisting

✅ Control which IPs can access the API via `--ipWhitelist` flag

### Local-Only Mode

✅ Restrict to localhost connections only via `--localOnly` flag

## Best Practices

1. **Run with Minimal Privileges**: Run tartuffe with least-privilege user account
2. **Enable API Key**: Use `--apikey` in production environments
3. **Use Local-Only**: Use `--localOnly` for local development
4. **Validate Imposter Configs**: Review imposter configurations before loading
5. **Prefer JavaScript over Shell**: Use `decorate` behavior instead of external commands
6. **Audit Logs**: Monitor log files for suspicious activity

## Compatibility Impact

The removal of `shellTransform` affects mountebank compatibility:

- **Mountebank tests using shellTransform**: ~4 tests will fail
- **Recommended**: Update test configurations to use `decorate` behavior
- **Migration**: See examples above for common use cases

## Reporting Security Issues

If you discover a security vulnerability in go-tartuffe, please report it to:
- GitHub Issues: https://github.com/TetsujinOni/go-tartuffe/issues
- Mark as "Security" label

Do not publicly disclose vulnerabilities until they are addressed.
