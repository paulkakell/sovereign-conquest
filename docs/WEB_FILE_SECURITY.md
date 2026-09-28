# Web file security

Version 01.06.11 confines bundled web serving to `WEB_ROOT`. The combined image
uses `/app/web`; an empty setting disables bundled web serving in API-only mode.
No new environment variable or dependency is required.

Every request opens a root directory handle and resolves file names relative to
that handle. Go's `os.Root` prevents both traversal and symlink escapes while
resolving paths. The response reads the same open regular file that was checked,
avoiding a separate unrestricted path lookup after validation. Handles close at
the end of each request. This does not change the attachment download endpoint,
which uses database-backed authorization and content.

| Request or layout | Behavior |
|---|---|
| `/app.js`, `/style.css` | Serve the in-root asset with normal HTTP metadata |
| `/`, `/bug.html` | Serve HTML with `Cache-Control: no-store` |
| `/nested/` with `nested/index.html` | Serve the in-root directory index without listing the directory |
| Missing `/client/route` | Serve the in-root root index for SPA navigation |
| Missing `/missing.js` | Return 404 |
| `/api/unknown` | Keep the JSON API 404; never return the SPA |
| `alias.js -> app.js` | Supported relative symlink within the root |
| `alias.js -> ../private/secret.txt` | Return 404 |
| Absolute symlinks, escaping index symlinks | Return 404 |
| Traversal components, backslashes, NUL bytes | Return 404 without exposing filesystem paths |

GET and HEAD remain supported, including byte ranges, Last-Modified conditional
requests and the existing relative redirect for `/nested/index.html`.
Only missing extensionless paths trigger SPA fallback; access-denied and escape
failures do not. Unknown write methods remain 404.

For custom layouts, put every public asset beneath `WEB_ROOT`, or use a relative
symlink whose target stays inside it. Keep credentials, backups and private
files outside that directory. The configured root and its mounts are trusted
deployment inputs; `os.Root` is a filesystem boundary, not a sandbox for an
operator who can change container mounts.

Build Validation retains CodeQL SARIF for 30 days. Run the same gate locally on
an exported report with:

```bash
node scripts/check-codeql-results.cjs path/to/go.sarif
```

Exit 1 indicates a high/critical finding or invalid report. Suppressions and
unchanged baseline markers do not exempt high findings. A successful CodeQL
upload alone is not evidence that the scan found no vulnerabilities.
