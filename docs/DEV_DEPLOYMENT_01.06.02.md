# Development Deployment 01.06.02

Pull the immutable image:

```bash
docker pull ghcr.io/paulkakell/sovereign-conquest:dev-01.06.02
```

Use PostgreSQL 16, bind port 8080 only to the intended test interface, and provide unique database, JWT, administrator, and bootstrap password values. For production-like testing, put TLS termination in front of the application, trust forwarding headers only from that proxy, and use a TLS-verified PostgreSQL connection when `APP_ENV=production`.

## Functional test checklist
1. Confirm `/api/livez`, `/api/readyz`, and `/api/version`.
2. Complete the seeded administrator password change.
3. Register a player and run `SCAN`, `MOVE`, and a trade.
4. Exercise messages, attachments, bug reporting, planet transfers, corporation banking, market filters, and route suggestions.
5. Restart the stack and confirm persistence.
6. Review structured request logs and readiness behavior.
7. Record defects against the `dev-01.06.02` tag and image digest.

## Rollback
Stop the candidate, preserve logs, and start the prior immutable image tag or digest after reviewing schema compatibility. Take and verify a PostgreSQL backup before season resets or destructive tests.
