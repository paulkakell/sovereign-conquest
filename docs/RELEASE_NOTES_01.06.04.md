# Sovereign Conquest v01.06.04

## Why this release exists

Nine open branches contain useful roadmap, dependency, and build changes. Several
cannot pass the existing contract tests individually because those tests pin the old
action and compiler versions. This maintenance release integrates the branches,
resolves their shared module conflict, and aligns the release contract.

See the [branch audit](BRANCH_AUDIT_01.06.04.md) for reviewed commit identities.

## Changes

- Add the canonical product and implementation roadmap from PR #22, document revision
  00.02.01. All fifteen implementation packets remain Planned.
- Update chi to 5.3.2 (#14), pgx to 5.11.0 (#21), and x/crypto to 0.57.0 (#20),
  including x/sync 0.23.0, x/text 0.42.0, checksums, and vendored sources.
- Update both Docker builders and CI/CodeQL to Go 1.27.1 (#18, #19).
- Update Buildx to v4 (#11), provenance attestation to v4 (#13), and Trivy to
  v0.36.0 (#12). Disable organization-only artifact storage records for this
  personal repository while retaining registry provenance.
- Align backend, browser assets, version assertions, configuration guidance, and
  release smoke expectations at 01.06.04.
- Add a code review with six preexisting findings and concrete follow-up gates.
- Add bounded release finalization that waits for CI, Build Validation, and GHCR
  publication success on the same main commit before tagging and removing merged
  branches. Unknown, changed, protected, or unmerged branches are preserved.

Classification: additive documentation and maintenance fixes. No new gameplay,
API, configuration, schema, or data-format behavior is intentionally introduced.

## Compatibility

Source builds now require Go 1.26 or later because of x/crypto; the supported build
and validation compiler is Go 1.27.1. Hosted GitHub runners support the updated
Node 24 actions; custom runners need version 2.327.1 or later.

pgx 5.11.0 changes connection-string parsing and date/time decoding. Standard
Compose URLs remain supported. Operators with special characters in database
credentials must use correctly URL-escaped values; unusual keyword connection
strings and Windows backslash paths merit separate validation. Text-format
`timestamptz` decoding uses the client's local time zone. Existing application
queries use native pgx, positional scanning, and the default extended protocol.

## Validation and deployment

See [validation evidence](VALIDATION_01.06.04.md), the merged release pull request,
and its exact-commit Actions runs. The repository's hosted integration gate builds
combined/API/web containers and tests a fresh PostgreSQL database. The publisher
scans and smoke-tests the resulting GHCR image. The release tag and branch cleanup
are withheld if any required workflow fails.

No database migration is introduced. Existing migration/upgrade coverage is not
made complete by this dependency release. The [code review](CODE_REVIEW_01.06.04.md)
records session revocation, repeated per-replica ticks, chat notification failures,
and reset/audit defects that remain unresolved. Keep one API replica until durable
scheduling is implemented; repair the documented reset defects before a live reset.

## Artifacts

The publisher produces `ghcr.io/paulkakell/sovereign-conquest:01.06.04`, `:main`,
and `:sha-<12-character-source-sha>`. Finalization creates Git tag `v01.06.04` and
its GitHub release only after all three release workflows succeed. The previous
baseline `03625c68a141f649aa87480b13d61527febceff1` is preserved as `v01.06.03`.

## Rollback

The prior image was verified accessible before release (HTTP 200). Pin
`SC_IMAGE=ghcr.io/paulkakell/sovereign-conquest@sha256:da629d8ffa417656e470a1fa12af717372d8d8e57cd89ed0d78f7f46155e60e1`, then run
`docker compose pull && docker compose up -d --remove-orphans`. The prior baseline
is retained in Git history and tagged; the verified digest avoids reliance on a moving tag. No database rollback is needed for this release because no
schema or persistent data changes are introduced. Revert the release merge with
`git revert -m 1 <release-merge-sha>` to reverse the source integration. The audit
record contains every deleted branch's source commit, so its ref can be recreated.
