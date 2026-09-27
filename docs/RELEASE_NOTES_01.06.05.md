# Sovereign Conquest v01.06.05

This maintenance release establishes the repository's contribution, discussion,
funding, and private security-reporting channels. It closes configuration gaps
without adding gameplay behavior.

## Changes

- Add issue forms for reproducible bugs, concrete feature requests, and
  documentation corrections, with separate links for help, ideas, and private
  vulnerability reports.
- Add forms for the existing Announcements, General, Ideas, Q&A, and Show and tell
  discussion categories. Polls retains GitHub's native composer.
- Configure `paulkakell` as the GitHub Sponsors recipient. Account enrollment and
  payment acceptance are separate prerequisites, not established by this release.
- Publish a security policy covering supported releases, confidential reports,
  safe reproduction, disclosure handling, and operator precautions.
- Add support and community guidance with examples for every reporting channel.
- Advance the root, API, browser, and build-validation version to `01.06.05`.
- Add tested release finalization that requires CI, Build Validation, and image
  publication to succeed on the same current `main` commit. It preserves existing
  tags and does not delete branches or change repository permissions.

Classification: additive repository configuration and documentation; maintenance
fixes for missing community entry points. No breaking application change.

Discussions and private vulnerability reporting were already enabled and verified
on 2026-09-27. This release supplies the repository files that make those channels
useful. It does not claim that Sponsors enrollment or payouts have been completed.

## Validation

All 44 local release-safeguard tests passed, including 20 new cases for this
release. YAML syntax and form structure, category slugs, JavaScript and shell
syntax, version consistency, and whitespace were checked locally.

GitHub Actions runs the complete Go unit/regression suite, race detector, vet,
formatting, dependency verification, Gosec, Govulncheck, CodeQL, Compose validation,
three container builds, and fresh PostgreSQL integration. Image publication adds
provenance, Trivy scanning, and the published-image smoke test. The release tag is
withheld until all three main workflows pass. See
[validation details](https://github.com/paulkakell/sovereign-conquest/blob/v01.06.05/docs/VALIDATION_01.06.05.md)
and the Actions runs for this release commit.

## Compatibility and remaining risks

Application APIs, commands, environment variables, schemas, stored data, and
dependency versions are unchanged. No database migration or rollback migration
is required. No application performance path changes; no new load benchmark is
required. Existing request-logging tests remain part of the full suite.

This release does not fix the application findings recorded in
[the 01.06.04 code review](https://github.com/paulkakell/sovereign-conquest/blob/v01.06.05/docs/CODE_REVIEW_01.06.04.md),
including session revocation and scheduled work across replicas. Operate one API
replica until the scheduling issue is fixed. Security policy publication is not
a claim that those defects are resolved.

## Artifacts and rollback

The release produces tag `v01.06.05`, these release notes, and the combined image
`ghcr.io/paulkakell/sovereign-conquest:01.06.05`. Use its published digest when
pinning a deployment. The previous source remains available at `v01.06.04`, commit
`d3a8e23c9e7975895d55d21cb8d513821cacc9d9`.

To roll back the source, revert the release merge with
`git revert -m 1 <01.06.05-merge-commit>`. To roll back the application, set
`SC_IMAGE=ghcr.io/paulkakell/sovereign-conquest:sha-d3a8e23c9e79`, then run
`docker compose pull` followed by `docker compose up -d --remove-orphans`.
Confirm readiness, `/api/version`, sign-in, and a basic game command. Retain a
database backup under the normal deployment procedure; this release changes no
persistent data format. Repository forms can be reverted independently of a
running deployment. Existing Discussions and private reporting remain enabled.

Copyable commit notes are in
[COMMIT_NOTES_01.06.05.md](https://github.com/paulkakell/sovereign-conquest/blob/v01.06.05/docs/COMMIT_NOTES_01.06.05.md).
