# Security policy

## Supported versions

Security fixes target the latest stable, tagged release listed on the
[Releases page](https://github.com/paulkakell/sovereign-conquest/releases).
Use the corresponding versioned image or a verified image digest.

| Version | Security maintenance |
| --- | --- |
| Latest stable tagged release | Supported |
| Earlier releases | Upgrade to the latest release; backports are not guaranteed |
| Development branches and untagged commits | Reports are welcome, but these are not supported production releases |

The maintained application version is recorded in `VERSION`. Support does not
mean the software is free of known defects; review the release notes and the
[documented security and reliability findings](docs/CODE_REVIEW_01.06.04.md).

## Report a vulnerability privately

Use GitHub's
[Report a vulnerability form](https://github.com/paulkakell/sovereign-conquest/security/advisories/new).
Private vulnerability reporting is enabled for this repository. Reports are
visible to the repository's authorized security collaborators, not the public.
Do not open a public issue or discussion containing exploit instructions.

Include:

- Affected version, commit, image digest, and deployment method when known.
- A description of the flaw, required access, and likely impact.
- Minimal reproduction steps or a small proof of concept using your own test data.
- Expected and actual behavior, with sanitized logs or screenshots if useful.
- Any suggested mitigation and whether you know of public disclosure already.

Never include real passwords, session tokens, administration keys, private keys,
complete environment files, database dumps, or another player's personal data.
If a credential was exposed, revoke or rotate it and report the type of exposure.

If the private form is unavailable, start a General discussion containing only
a request for a private reporting channel. Wait for a secure channel before
sharing details.

## What belongs in a security report

Examples include authentication or authorization bypass, account takeover,
injection, cross-site scripting, unauthorized access to player or corporation
data, exposed secrets, and ways to bypass server validation or duplicate credits,
turns, cargo, or assets. A dependency advisory should identify how this application
uses the affected component when possible.

Use the normal bug form for ordinary defects. Use Discussions for intended
gameplay mechanics, balance concerns, installation help, and feature ideas.

## Handling and disclosure

Maintainers will review the report privately, request missing details, and
coordinate any fix, mitigation, and public advisory with the reporter. Response
and remediation times depend on maintainer availability and severity; there is
no guaranteed response deadline. Follow up in the private report if needed.

Please allow time to investigate before publishing exploit details. Reporter
credit can be included with permission. No paid bounty is promised by this policy.

Test only on systems and accounts you own or have explicit permission to assess.
This policy does not grant permission to test third-party deployments. Avoid
disrupting live games, accessing other players' data, or changing their assets.

## Operator precautions

Use strong, unique secrets and verified TLS for production database connections.
Keep PostgreSQL off public interfaces and place public application access behind
TLS. Keep backups, pin deployment artifacts, and review release notes before
upgrading. Run one API replica until the documented scheduling limitation is fixed.
The existing session-revocation and administrative-reset findings remain relevant;
this policy is not an application security fix.
