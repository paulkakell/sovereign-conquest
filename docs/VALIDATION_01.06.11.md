# Validation and security review for 01.06.11

Base: `d4654ff3234ce03382a51a9269352acd2a006ea3` (v01.06.10, PR #30).

## Findings and reproduction

The existing GitHub workflow's security access returned exactly four open alerts
on main. Baseline SARIF contains four `go/path-injection` findings, each with
security score 7.5. Evidence: [baseline run 36371286327](https://github.com/paulkakell/sovereign-conquest/actions/runs/36371286327),
commit `3518047bf4d000eed06c7e5ae069eb67bca73c62`.

| Alert | Original location | Affected operation |
|---|---|---|
| #2 | server/internal/api/server.go:118 | Asset metadata lookup |
| #3 | server/internal/api/server.go:121 | Directory-index metadata lookup |
| #4 | server/internal/api/server.go:130 | Direct asset serving |
| #5 | server/internal/api/server.go:146 | HTML/index/fallback serving |

The exploit regression failed on the original handler: relative and absolute
file symlinks, directory symlinks and an escaping index returned outside fixture
content. The new rooted handler passes those cases, encoded traversal, HEAD,
malformed names, safe relative symlinks, missing routes/assets, HTML cache policy,
API 404s, Range, conditional requests and canonical index redirects. A concurrent
replacement test repeatedly swaps a symlink between public and private targets.
The SARIF gate was also run against the actual baseline report and rejected all
four high findings with exit 1.

## Release checks

Local validation passed: full Go unit/regression and race suites, vet, module
verification, a CGO-disabled static API build, Gosec, 179 JavaScript tests,
26 Python policy/recovery tests, JavaScript/shell syntax and diff whitespace
checks. Govulncheck found zero reachable or imported-package vulnerabilities;
the existing unused-module advisory is reviewed separately below.
Hosted results and the implementation reference will be recorded after completion.
Docker is unavailable in the local workspace; clean container builds, real
Compose rendering, database integration and image scans run on GitHub runners.

## Security, dependencies and compatibility

- Input and I/O: every lookup stays within `os.Root`; the checked open regular
  file supplies the response. No request-derived name reaches unrestricted
  `os.Stat` or `http.ServeFile`. No query exclusions or alert dismissals.
- Authentication and authorization: bcrypt, JWT validation, administrator checks,
  attachment authorization and rate limits are unchanged. API routes cannot
  fall through to the SPA, including normalized duplicate-slash routes.
- Secrets and logs: no new credentials; generic 404s contain no paths or OS
  errors. Existing structured HTTP logs, probes and metrics remain intact.
- Dependencies: no manifest, checksum or vendored changes. Use the existing
  Go 1.27.1 toolchain. Module, reachable-code and container scanners remain gates.
  GO-2026-5932 concerns the unused `golang.org/x/crypto/openpgp` package;
  the application imports bcrypt, not OpenPGP. No applicable dependency fix exists.
- Configuration: bump all active version/image references to 01.06.11;
  `WEB_ROOT` semantics tighten only for unsafe layouts. Existing ports, flags,
  volumes, secret requirements and default image behavior remain intact.
- Database: no schema or SQL changes; forward/backward data compatibility is
  unaffected and no migration/rollback script is needed.
- Rollback: the prior source tag is preserved; use the retained 01.06.10 image
  digest and settings without deleting data. Rollback reintroduces the old
  handler, so eliminate external web symlinks first.

## Performance

Three local benchmark samples for a small static asset measured 6.31-6.62 us
before and 7.22-7.42 us after the fix. Median cost increased about 0.78 us
(12%), while allocations fell from 30 to 28 per request. The extra root handle
provides a filesystem boundary. These are handler microbenchmarks, not
production throughput measurements. No database or gameplay path changes.
