## 01.06.01 release candidate

This branch includes fresh-database startup repair, browser/API compatibility fixes, HTTP abuse controls, administrator authorization, replica-safe scheduled jobs, non-root API images, health probes, and an expanded CI gate.

Operational references:

- [Release notes](docs/RELEASE_NOTES_01.06.01.md)
- [Configuration guide](docs/CONFIGURATION.md)
- [Changelog](CHANGELOG.md)

Release status: blocked pending trusted regeneration and validation of `server/go.sum` and `server/vendor`. The repository connector did not permit that dependency operation, so no dependency-upgrade claim is made for 01.06.01.
