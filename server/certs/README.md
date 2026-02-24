Custom CA certificates (optional)

Some deployment environments (corporate networks, intercepting proxies, private module proxies) require
custom Certificate Authorities (CAs) to be trusted during the container image build.

If your Docker build fails while downloading Go modules with errors like:
  - "x509: certificate signed by unknown authority"
  - "tls: failed to verify certificate"

Place one or more CA certificates (PEM-encoded) in this directory with a `.crt` extension, for example:
  - `server/certs/corporate-root-ca.crt`

The `server/Dockerfile` copies `server/certs/` into the build stage and runtime stage and runs
`update-ca-certificates`, so Go, git, and apk can trust these CAs.

Notes
  - Do not commit private/internal CA certificates to public repositories.
  - If you must keep the repo public, prefer mounting certificates at build time or using a private fork.
