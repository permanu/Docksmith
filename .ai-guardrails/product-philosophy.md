# Product Philosophy — Apple Ecosystem Principle

> Docksmith is part of the Deploy by Permanu ecosystem.
> This philosophy applies to every line of code.

## Core Principle

> Everything inside the Deploy ecosystem must be best-in-class DX.
> Users stay not because they're locked in, but because leaving means downgrading their experience.

## Docksmith-Specific Application

Docksmith generates Dockerfiles for the Deploy platform. The Dockerfile we generate must be **better than what the user would write themselves**.

### DO

| Rule | Why |
|------|-----|
| Generate distroless, tini, non-root, healthchecks by default | Users shouldn't need to know Docker best practices |
| Clear, actionable error messages when detection fails | "We couldn't detect your framework" + what to do next |
| Optimal layer caching (dependencies before source) | Faster rebuilds = users feel the speed |
| Multi-stage builds that minimize image size | Smaller images = faster deploys = better DX |
| Support every major framework out of the box | Users shouldn't have to write a Dockerfile ever |
| Generate .dockerignore automatically | One less thing to think about |

### DON'T

| Anti-pattern | Why |
|-------------|-----|
| Don't generate generic Dockerfiles | If it looks like `docker init` output, we've failed |
| Don't silently produce a wrong Dockerfile | Fail loudly with context, never silently wrong |
| Don't require user config for common frameworks | Zero-config for 90% of projects |
| Don't ignore security (root user, no healthcheck) | Every generated Dockerfile must be production-grade |

## The Test

**"Would a user get a better Dockerfile by writing it themselves or using another tool?"**

If yes → we haven't earned the right to auto-generate. Improve detection, improve templates.
