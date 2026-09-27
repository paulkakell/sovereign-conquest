# Code review for 01.06.04

Review date: 2026-09-27

Baseline: `main` at `03625c68a141f649aa87480b13d61527febceff1`.

## Scope and limits

This is a source review of authentication, authorization, input handling, secrets,
request logging, administration, and database interactions. It covers application
code under `server/` and relevant browser rendering under `web/static/`. No
repository or parent-directory `AGENTS.md` was found during this review.

All findings below are preexisting at the baseline. They are not regressions
introduced by the pending dependency updates or roadmap merge, and this review
does not claim they were fixed in 01.06.04. Source references use baseline line
numbers. The roadmap references identify planned work, not completed fixes.

This review did not run a live exploit, mutate a database, reproduce the
multi-replica scenario, or execute the release validation suite. Build, test,
dependency, and release-gate results must be reported separately. Static code
inspection is not a claim that a production incident occurred or that all
security defects have been found.

## Findings

### CR-01: Password changes do not revoke existing sessions

Priority: P1, account recovery and administrator-session security.

Evidence:

- `server/internal/api/server.go:154-169` validates JWT signatures and places the
  claimed user and player IDs into context without checking a persisted session
  version or credential-change timestamp.
- `server/internal/api/server.go:311,350` mints seven-day tokens using
  `auth.MintToken`, which assigns session version zero.
- `server/internal/api/server.go:398-402` changes the password hash and timestamp
  but does not invalidate prior tokens.
- `server/internal/api/admin_guard.go:47-54` checks the signed token and current
  administrator flag without validating credential freshness.
- `server/internal/auth/auth.go` defines `Claims.SessionVersion`, but the user
  schema and API do not implement the corresponding revocation mechanism.

A stolen token can remain usable after the legitimate user changes their
password. The same gap affects a previously issued token when bootstrap recovery
promotes an existing account and resets its password. The separate administrator
secret still protects the soft-wipe endpoint.

Required correction: persist a session version, mint it into tokens, validate it
on normal and administrative requests, and increment it atomically on password
changes and recovery. Return or require a fresh authenticated session after a
change. Verify user/player ownership as part of that lookup.

Related validation gap: registration and password changes accept up to 100 bytes
at `server/internal/api/server.go:218,377`, while bcrypt accepts at most 72 bytes.
Align the API and browser limits so oversized passwords receive a validation
error rather than an internal-server error.

Acceptance: an old normal or administrator token fails after a password change
or recovery; a freshly issued token works; missing users and mismatched player
IDs fail closed; password limits are tested at the byte boundary.

Roadmap: F04 and SC-I03, M1.

### CR-02: Per-invocation locks do not prevent duplicate scheduled intervals

Priority: P1 before running multiple API replicas; not a single-replica blocker.

Evidence:

- `server/internal/game/tick.go:58-66` creates a separate timer in every process.
- `server/internal/game/ticker_lock.go:30-43` obtains a transaction-scoped lock,
  performs work, and releases the lock at commit without recording a due cursor.
- `server/internal/game/event_leader.go` likewise releases its leader lock after
  each invocation rather than holding leadership across the schedule.
- `server/internal/game/protectorate.go:184-190` runs its recurring mutation in
  every process without this coordination.

Staggered replicas can each acquire the lock and apply production during the same
configured interval. A lock prevents overlapping work, but does not establish
that the interval has already been processed. This can multiply port and planet
production and change event frequency as replicas are added.

Required correction: record and advance a durable due cursor in the same
transaction as each scheduled mutation. Include bounded catch-up and recorded
random outcomes where retries could otherwise reroll events.

Acceptance: one worker and two staggered workers produce the same totals for the
same elapsed intervals; retries and restarts cannot apply an interval twice.

Interim operating recommendation: run one API replica until this behavior has a
database-backed regression test and a verified correction.

Roadmap: F05 and SC-I06, M2. The existing locking change is traceable to
`ca7e3dc` and predates this consolidation.

### CR-03: Malformed input can invoke the destructive default wipe

Priority: P2; correct before using the endpoint on a live season.

Evidence: `server/internal/api/server.go:1154-1162` ignores the JSON decoder error
and invokes `game.SoftWipe` using the resulting request, including zero values
when the body is empty or malformed.

The deployed handler still requires a signed administrator identity and the
separate administrator secret. This is input-validation failure on an authorized
destructive action, not an unauthenticated wipe vulnerability.

Required correction: reject malformed or empty bodies, reject trailing JSON,
validate the request contract, and perform no mutation when parsing fails.
Preserve both existing authorization requirements.

Acceptance: malformed, truncated, empty, and multiple-object JSON produce client
errors and leave seasons and player state unchanged; a valid authorized request
continues to work.

Roadmap: add these acceptance cases to SC-I03's reset work, then preserve them in
SC-I13. The roadmap already covers the reset path but does not explicitly list
this decoder-error case.

### CR-04: Season wipe preserves fleet value and inconsistent capacities

Priority: P2; blocks using legacy soft wipe as a fair competitive-season reset.

Evidence:

- `server/internal/game/admin.go:46-58` resets credits, cargo, and turn capacities
  but leaves ship type, ship upgrade counters, XP, and level intact.
- `server/internal/game/shipyard.go:151-177` calculates resale from ship type.
  A retained FREIGHTER therefore remains worth 42,000 credits after the reset.
- `server/internal/game/admin.go:51` assumes sector 1, unlike registration's
  selection of an existing starting sector.

Preserving progression can be an intentional product policy, but preserving an
expensive hull while resetting its capacities to starter values is inconsistent
and gives retained fleets a financial advantage at the next start.

Required correction: publish an explicit reset manifest and use a shared starter
state for hull, upgrades, capacities, credits, and a valid protected spawn. Treat
long-term progression separately according to the published season rules.

Acceptance: after reset, ship type and capacities agree, upgrade counters match
the chosen policy, a pilot cannot sell a retained seasonal freighter, and spawn
selection succeeds on every supported generated world.

Roadmap: F03 and SC-I03, M1; complete seasonal archive and settlement in SC-I13, M6.

### CR-05: Corporation chat uses a busy transaction connection

Priority: P2, message delivery correctness.

Evidence: `server/internal/game/corp.go:214-225` iterates `memberRows` while issuing
`tx.Exec` on the same pgx transaction and discards the returned errors. The
vendored driver's `pgconn/pgconn.go` explicitly reports `conn busy` for a second
operation on a busy connection.

The source indicates that a corporation message can be stored and reported as
sent without recipient log delivery. This was not reproduced against a live
PostgreSQL instance during this review.

Required correction: perform one checked `INSERT ... SELECT` for recipient logs,
or collect the recipients and close the rows before issuing checked writes.
Keep message storage and required delivery records in the same transaction.

Acceptance: a real database test verifies delivery to every current member,
exclusion of nonmembers, and rollback on a required delivery failure.

Roadmap: F07 and SC-I02, M1.

### CR-06: Destructive administration lacks durable actor attribution

Priority: P2, operational accountability and recovery.

Evidence: `server/internal/schema/preflight.go:71-80` creates `admin_audit_log`,
but no application code inserts into it. `game.SoftWipe` accepts no actor identity
and deletes gameplay logs as part of the reset. Transport request logs record the
path and status but do not replace an attributable administrative audit record.

Required correction: write the authorized actor, operation, source and target
season, requested reset policy, affected counts, and outcome in an appropriate
durable audit record. Avoid recording secrets or private message contents. Make
successful audit recording atomic with the destructive transaction.

Acceptance: a committed reset has one attributable audit entry; a failed reset
does not leave a success entry or partially reset data; ordinary players cannot
create or alter audit records through the API.

Roadmap: SC-I03's minimum reset audit, M1; SC-I13's full season transition, M6.

## Controls observed

- JWT validation restricts algorithms and requires expiration and issuance data.
- Password hashing uses bcrypt cost 12.
- Reviewed SQL data values use bound parameters. Trading locks player and port
  rows and uses a transaction for balance and inventory mutations.
- Corporation withdrawals require leadership and use a conditional balance update.
- Attachment retrieval checks sender/recipient ownership, with explicit separate
  administrator access. Message text is escaped or rendered through text content.
- The deployed HTTP stack applies request-size limits, endpoint throttles,
  timeouts, security headers, and proxy-header sanitization.
- Production startup validates secret lengths and database transport settings.
  Proxy trust defaults to disabled. Request logging avoids credentials, request
  bodies, and client-address persistence.

These observations describe specific controls, not a clean security certification.
Deployment must still supply unique secrets and the documented production
configuration.

## Release and operating recommendation

The preexisting findings do not by themselves require rejecting a documentation
or dependency consolidation that preserves behavior and passes its own gates.
Record them as unresolved work rather than claiming that branch consolidation
resolves security or gameplay correctness.

Prioritize session revocation in the next application correctness release. Run
one API replica until scheduled intervals are durably deduplicated. Avoid using
legacy soft wipe as a production competitive-season transition until input
validation, reset consistency, actor attribution, and the declared archival
policy are implemented and tested.
