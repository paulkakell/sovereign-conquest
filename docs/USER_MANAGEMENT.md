# User Management (01.07.00)

Sign in with an administrator account, finish the required password change if
prompted, then click **User Management**. The button is hidden for other players;
every endpoint also verifies the role and current account session on the server.
No additional environment setting or ADMIN_SECRET is needed for account management.
The destructive season-reset endpoint still requires ADMIN_SECRET separately.

## Find and edit an account

Search by part of a username (case insensitive) or an exact user ID. Filter by
Active, Suspended, or Banned, then select **Edit**. Lists contain at most 50 accounts;
use **Next page** or **First page** to navigate. Accounts without a player remain
listed and their account fields can still be edited. An expired suspension appears
as Active while its historical reason and expiry remain visible.

The form saves only changed fields, including zero and false values. Editing a
stale account returns a conflict rather than overwriting another administrator's
changes. Click **Reload account** to review the current record before retrying.
If editing your own username, password, or password-change requirement revokes your
session, the form saves first and then returns you to sign-in.

| Field | Rules and example |
| --- | --- |
| Username | Unique, trimmed, 3-20 UTF-8 bytes, no control characters. Example: `FrontierPilot`. |
| Administrator | Enable to grant administration and existing game administrator privileges; disable to remove them. All old sessions end. You cannot disable your own role. |
| Require password change | Checked accounts must change their password before gameplay, messaging or administration. They may still load their own state and use the password-change endpoint. |
| New password / confirmation | Leave blank to preserve the current password. Reset requires 8-72 UTF-8 bytes; Unicode may occupy several bytes per character. Typing a reset checks Require password change by default; explicitly uncheck only when issuing a permanent replacement. |
| Credits / XP | Whole numbers from 0 to 9000000000000000. XP determines level. Example: set credits to 0 or XP to 20000. The admin API encodes these two values as decimal strings to preserve precision. |
| Ship type | `SCOUT`, `TRADER`, `FREIGHTER`, or `INTERCEPTOR`. Example: select TRADER and explicitly set the intended capacities. Ship changes do not automatically purchase upgrades or recalculate capacity. |
| Cargo upgrades / turn upgrades | Whole numbers 0-20 / 0-10. Set capacities explicitly when making a correction. |
| Turns / maximum turns | Whole numbers 0-1000000. Turns must not exceed maximum turns. Setting turns restarts the regeneration timestamp. |
| Cargo capacity / ore / organics / equipment | Whole numbers 0-1000000. Combined cargo cannot exceed capacity. Example: capacity 80, ore 10, organics 0, equipment 25. |
| Sector ID | An existing positive sector ID. Moving a player also marks that destination discovered. |
| Season ID | An existing positive season ID. This corrects the player's association; it does not run a season reset or move other assets. |
| User/player IDs, creation dates, password-change and moderation timestamps, last regeneration | Read-only system records. IDs keep foreign-key relationships intact; timestamps record the action that actually occurred. |
| Level | Read-only, derived from XP. |
| Status, suspension end and reason | Change through Account access, described below. |

Corporation ownership, other players, planets and global assets are not fields of
the account/player record and are outside this editor. Session counters, password
hashes and internal revision bookkeeping cannot be arbitrarily edited.

## Suspend, ban or restore

Enter a reason of 1-1000 UTF-8 bytes, then choose the action and confirm the named
account. Save or discard unsaved edits before changing access.

- **Suspend user:** optionally choose a future end date/time in your local timezone.
  For example, suspend a player until tomorrow at 18:00 while investigating abuse.
  Leaving the end empty suspends until an administrator restores access.
- **Ban user:** deny access indefinitely. For example, ban a confirmed abusive
  account with a reason recorded for future administrators. There is no automatic
  expiry and the user's data is preserved.
- **Restore access:** lift either restriction and record why. The user must sign in
  again. For example, restore a suspension after an appeal is resolved.

Timed expiry is checked on each request and login, so no scheduler is required.
All protected endpoints honor restrictions, including messages, attachments, the
admin map, password changes and season reset. Requests that already passed their
authorization check can finish; subsequent requests are rejected. Suspension,
banning, restoration, username changes, role changes, password changes and resets
invalidate earlier bearer tokens. Expiry never revives a revoked token.

Administrators cannot suspend, ban or demote themselves. Concurrent changes are
serialized, the acting admin is checked again inside the transaction, and an active
administrator with a player must remain. Renaming the configured bootstrap admin
no longer recreates the old username on restart when another admin record exists.
Keep INITIAL_ADMIN_USERNAME aligned with your intended recovery account.

Moderation affects this account. It does not identify a person across newly
registered accounts, implement an IP ban, freeze passive world production, or
remove their game assets. Those are separate product decisions.

## API reference

All routes require `Authorization: Bearer <current-admin-token>`. No cookies are
accepted as authentication. Responses are JSON and production transport marks API
responses `Cache-Control: no-store`.

| Method and path | Purpose |
| --- | --- |
| `GET /api/admin/users?q=pilot&status=active&after=<cursor>` | Search/filter; all query fields optional. `q` maximum 64 bytes, `after` maximum 256. Returns `users` and `next_cursor` (empty when finished). |
| `GET /api/admin/users/{user_id}` | Returns `user`, including optional `player`, `revision`, effective status and audit timestamps. |
| `PATCH /api/admin/users/{user_id}` | Partial update with required current `revision`. Unknown fields, direct hash/ID/timestamp writes, trailing JSON and payloads over 32 KiB are rejected. |
| `POST /api/admin/users/{user_id}/moderation` | Required `revision`, `action` (`suspend`, `ban`, `activate`), and `reason`; optional future RFC3339 `suspended_until` for suspension only. |

Account correction:

```json
{"revision":0,"username":"FrontierPilot","is_admin":false,"must_change_password":false,"player":{"credits":"45000","xp":"2200","ship_type":"TRADER","ship_cargo_upgrades":2,"ship_turn_upgrades":1,"turns":95,"turns_max":110,"sector_id":14,"cargo_max":80,"cargo_ore":10,"cargo_organics":0,"cargo_equipment":25,"season_id":1}}
```

Password reset (replace the example with a newly generated password):

```json
{"revision":1,"new_password":"Example-Replacement-Only-2026!","must_change_password":true}
```

Access examples (use a future date and the revision from your latest response):

```json
{"revision":2,"action":"suspend","reason":"Abuse investigation","suspended_until":"2026-10-01T00:00:00Z"}
{"revision":3,"action":"suspend","reason":"Investigation remains open"}
{"revision":4,"action":"ban","reason":"Confirmed repeat abuse"}
{"revision":5,"action":"activate","reason":"Appeal accepted"}
```

Successful mutations return `ok`, the refreshed `user`, and `sign_in_again`.
Errors: 400 invalid input, 401 absent/invalid/revoked session, 403 restricted
account/password change required/not administrator, 404 unknown user, 409 stale
revision/duplicate username/admin lockout, 500 failed transaction, 503 session
lookup unavailable. A 403 access error includes a machine-readable `code` such as
`account_suspended`, `account_banned`, or `password_change_required`; revoked
sessions use `session_revoked` with 401. Reasons are admin-only and are not
returned to unauthenticated login attempts.

## Data, audit and performance

Startup adds session_version, account_revision, account_status, suspended_until,
moderation_reason and moderated_at to users, plus indexes. Existing accounts
backfill as active at revision/session version zero. Existing version-zero tokens
remain valid until their account is changed. Migrations are idempotent and run
before HTTP startup; no new environment variables or dependencies are introduced.

Each save locks the account/player, validates values and foreign keys, and writes
an admin_audit_log entry in the same transaction. Audit includes actor, target,
action, changed fields, non-secret before/after account values, reason and time.
Passwords, hashes, JWTs and request bodies are excluded. Failed audit insertion
rolls back the change. Structured logs contain actor/target IDs, action and field
names. Ordinary health probes and request metrics are unchanged.

Every authenticated request adds one indexed user/player lookup to enforce access.
Lists use an ID cursor and a hard page limit. Substring searches can scan accounts;
consider a trigram index if measured search cost warrants it at a larger scale.
Admin writes serialize with one transaction advisory lock; normal gameplay does
not take that lock. The CI session benchmark measures parallel lookup cost against
its disposable PostgreSQL service, not production capacity.

## Upgrade and rollback

Back up PostgreSQL and record the currently running image digest before upgrading.
Deploy the verified 01.07.00 image/digest, check readiness, sign in as an admin and
exercise one disposable account through suspend and restore.

The retained `v01.06.13` source and its verified image digest are the rollback
baseline. **Older binaries do not enforce suspension, banning or session versions.**
Close public ingress before rolling back. Either fix forward or restore the
pre-upgrade database backup and rotate JWT_SECRET before reopening. Review all
moderation decisions made since the backup; rollback must not silently restore
access for users whose bans are still required.

Additive columns can remain for a binary rollback. If restoring the old schema is
necessary after preserving/exporting moderation evidence, stop the API and remove
only the six columns listed above and idx_users_username_id. This discards new
moderation/session metadata, so keep access closed until a compatible release is
running or restrictions have been explicitly reconciled. Integration tests check
legacy-column reads after reversal, reapplication/backfill, and preservation of
users/player records. This is a maintenance rollback, not a live rolling downgrade.
