# Commit notes for 01.06.04

```text
chore(release): consolidate roadmap and dependency branches for 01.06.04

Integrate the roadmap and eight dependency/build proposals while preserving
all source commits in the release ancestry.

- Merge PRs #11, #12, #13, #14, #18, #19, #20, #21, and #22.
- Align Docker, CI, and CodeQL on Go 1.27.1.
- Resolve go.mod conflicts and retain synchronized checksums/vendor metadata.
- Repair action/compiler/version contract assertions for the new release.
- Record the code review, compatibility limits, validation, and rollback.
- Finalize tags/releases and remove only verified merged branch tips after
  all release workflows pass for the same main commit.

Classification: additive documentation and maintenance fixes.
Compatibility: source minimum Go rises to 1.26; no API or schema changes.
```

Use the validation record and exact GitHub Actions results for test claims.
Preexisting security and gameplay defects are documented, not claimed fixed.
