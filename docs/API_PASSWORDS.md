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
