# Security Policy

## Reporting a Vulnerability

Email security@permanu.com with a description of the vulnerability.

We will respond within 48 hours and aim to release a fix within 7 days
for critical issues.

Do not open a public issue for security vulnerabilities.

## Scope

Docksmith generates Dockerfiles from user input. Security-relevant areas:

- **Dockerfile injection**: user-controlled strings (build commands, start commands)
  interpolated into generated Dockerfiles
- **Path traversal**: config paths, framework file paths
- **Dependency confusion**: YAML framework definitions from untrusted sources

## Supported Versions

Only the latest release receives security updates.
