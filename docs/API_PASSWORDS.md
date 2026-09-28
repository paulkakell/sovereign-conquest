# Password API contract

Version 01.06.10 fixes the former 100-character validation message. Registration
and password changes accept **8-72 UTF-8 bytes**, matching bcrypt's input limit.
ASCII characters each occupy one byte. For example, 72 ASCII letters or 36
copies of `é` fit; 73 ASCII letters or 37 copies of `é` do not. Passwords sent to
these endpoints are not trimmed. There are no mandatory character classes.

`POST /api/register` accepts:

```json
{"username":"example_user","password":"replace-with-a-unique-password"}
```

`POST /api/change_password` requires the existing bearer session and accepts:

```json
{"old_password":"current-password","new_password":"replace-with-another-unique-password"}
```

An invalid new-password length returns HTTP 400 before hashing or database
access, with an `error` message describing the 8-72-byte range and Unicode
behavior. Supplied passwords are never echoed. A rejected password change
leaves the existing password intact. Valid passwords at either boundary remain
accepted; successful changes retain the existing authentication behavior.
Login semantics and existing password hashes are unchanged.

The environment variable `INITIAL_ADMIN_PASSWORD` is a separate bootstrap
setting with a stronger **16-72-byte** requirement after surrounding whitespace
is trimmed. See [startup configuration](CONFIGURATION.md) for all four secrets.

## Session behavior from 01.07.00

Successful `POST /api/change_password` returns `{ "ok": true, "token": "<replacement bearer token>" }`. Store the returned token before further requests. All earlier tokens for that account are revoked. A password-required account may access `/api/state` and `/api/change_password` only. The bundled client handles the replacement token automatically.

Administrator resets use the same 8-72 UTF-8 byte limits and normally require a password change at next login. See [User Management](USER_MANAGEMENT.md). Suspension and banning block both login and authenticated API access. Expiry or restoration requires a fresh login; old tokens remain revoked.
