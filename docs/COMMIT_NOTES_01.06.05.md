# Commit notes for 01.06.05

```text
chore(community): configure GitHub contribution channels for 01.06.05

Add bug, feature, and documentation issue forms with support and private
security-reporting links. Add forms for the existing discussion categories,
configure paulkakell as the GitHub Sponsors recipient, and publish security,
support, and community guidance.

Align release metadata at 01.06.05 and add tested release finalization that
requires CI, build validation, and GHCR publication on the exact main commit.

Classification: additive repository configuration and maintenance fixes.
No breaking API, gameplay, schema, environment, or dependency changes.

Validation: 44 local release tests and YAML/syntax checks passed; full Go,
security, container, and PostgreSQL checks are required hosted release gates.
Sponsor payment acceptance depends on the owner's active Sponsors profile.

Base: d3a8e23c9e7975895d55d21cb8d513821cacc9d9 (v01.06.04, PR #23).
Rollback: revert the release merge or use the previous source-specific image.
```
