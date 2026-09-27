# API runtime security review: 01.06.06

Review date: 2026-09-27. Eight reported package rows represent six unique CVEs.
This release removes the affected runtime components. It does not claim the
Alpine build stages or unrelated PostgreSQL/web-only images are vulnerability-free.

| Reported CVE | Reported package | Remediation |
| --- | --- | --- |
| CVE-2026-85091 | zlib 1.3.2-r0 | No zlib or shared libraries in static Go runtime |
| CVE-2026-58469 | wget 1.25.0-r3 | No wget; native bounded Go health probe |
| CVE-2026-58470 | wget 1.25.0-r3 | Same removal; blocked even at medium severity |
| CVE-2026-58471 | wget 1.25.0-r3 | Same removal |
| CVE-2026-58472 | wget 1.25.0-r3 | Same removal |
| CVE-2025-60876 | busybox, busybox-binsh, ssl_client 1.37.0-r31 | No BusyBox binary, applets, shell, or SSL helper |

## Evidence and reasoning

The [Alpine wget tracker](https://security.alpinelinux.org/vuln/CVE-2026-58469)
and [BusyBox tracker](https://security.alpinelinux.org/vuln/CVE-2025-60876)
still list the supplied Alpine 3.24 revisions as possibly vulnerable on the
review date. An `apk upgrade` alone cannot establish remediation.
The [zlib upstream fix](https://github.com/madler/zlib/commit/df84af25dc1942490e1d1c899a07619152a46148)
addresses its buffer overflow; this application does not need that C library.

The API already builds with `CGO_ENABLED=0`. Final stages therefore copy the
static Go executable and required data from build stages into `scratch`, with
no base-distribution package layer. Go's compression implementation is not
the removed C zlib library. Binary vulnerability scanning still checks Go code.

The final image contents and binary linkage are checked independently of the
vulnerability database. Trivy runs with unfixed findings included; the policy
also rejects all six IDs independent of severity. No ignore rule or metadata
deletion is used as remediation. Scan results describe one image and one database
snapshot, not a permanent guarantee against future vulnerabilities.

## Application security and operations

- Authentication, authorization, player input validation, and application logs
  are unchanged. The existing [code review findings](CODE_REVIEW_01.06.04.md)
  remain open; this patch is scoped to container vulnerabilities.
- The health probe executes before secret/config loading, makes one HTTP request
  to the configured listener, rejects redirects, bypasses proxy environment
  variables, and logs no response body or credentials. Failures return nonzero.
- UID/GID 10001, read-only Compose storage, `/tmp` tmpfs, dropped capabilities,
  and `no-new-privileges` are preserved. Custom trust roots and timezone data are
  retained. Private keys or runtime secrets are not introduced into the image.
- Dependency manifests, checksums, and vendored Go code are unchanged. Existing
  Gosec, Govulncheck, race detection, and CodeQL gates remain required.
- No schema changes, migration scripts, persistent data changes, or game query
  changes. The small health probe is bounded; no gameplay load change is expected.
- Public tags move only after candidate image validation. Release finalization
  requires all designated workflows on the exact current main commit and never
  moves an existing version tag.

Build-time trust still depends on the existing Go/Alpine builder and certificate
sources. Removing runtime packages does not repair upstream builder components.
Continue builder updates and source analysis. External deployment health,
metrics, and alerts require operator verification after rollout.
