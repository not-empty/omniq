# OmniQ Release Checklist

This document is the practical release checklist for the OmniQ ecosystem:

- `omniq-core`
- `omniq-python`
- `omniq-node`
- `omniq-go`
- `omniq-php`

It is intentionally execution-oriented.
Do not treat release as a documentation-only task.

---

## Release Order

Recommended order:

1. `omniq-core`
2. `omniq-python`
3. `omniq-node`
4. `omniq-go`
5. `omniq-php`

This order is safest when SDKs depend on a new contract shape from core.

---

## Global Checklist

Before publishing anything:

- create a release branch in each repo
- review `git diff` in each repo
- confirm no accidental local-only files are included
- confirm the intended version number in each repo
- confirm the Lua scripts that must stay identical are still identical across repos
- prepare a short release note with the functional changes
- confirm README examples are aligned with the current API

Before tagging:

- merge the final release branch
- rerun the required checks
- confirm the branch is clean
- push the branch
- create and push the tag only after the release commit is final

---

## Core

Repository checks:

- confirm `scripts/*.lua` are the intended source of truth
- confirm contract docs are current:
  - `docs/omni-contract.md`
  - `docs/sdk-validation-checklist.md`
  - `docs/omniq_redis_map.md`
- confirm validation docs are current:
  - `validation/README.md`
  - any new scenario README files

Execution checks:

- bring up the unified validation environment
- run at minimum:
  - scenario `01` basic publish/reserve
  - scenario `28` consume max attempts
  - scenario `29` queue name validation
  - scenario `30` scan queues discovery rules
  - scenario `31` multi-queue `NOSCRIPT` recovery
  - scenario `32` transport backend smoke
- preferably run the full suite:
  - `VALIDATION_BACKENDS='standalone cluster' bash ./validation/run_suite.sh`
- confirm all scenarios pass

Publishing steps:

- push the branch
- create/push the release tag
- optionally create a GitHub release with release notes

---

## Python

Repository checks:

- confirm metadata in `pyproject.toml`
  - package name
  - version
  - URLs
- confirm package data still includes Lua files
- confirm README matches the released API

Execution checks:

- run syntax/build validation
- build artifacts:
  - `python3 -m build`
- inspect `dist/`
- test install from built artifacts in a clean environment if practical
- run a small smoke test against Redis if practical

Publish steps:

- publish to PyPI
- verify the published version page
- verify install from PyPI works

---

## Node / TypeScript

Repository checks:

- confirm metadata in `package.json`
  - package name
  - version
  - `main`
  - `module`
  - `types`
  - `files`
- confirm built output is meant to be published
- confirm README matches the released API

Execution checks:

- run:
  - `npm run typecheck`
  - `npm run build`
- inspect `dist/`
  - JS output
  - CJS output
  - `.d.ts`
  - copied Lua scripts
- produce a tarball:
  - `npm pack`
- inspect tarball contents
- test install/import in a clean temp project if practical
- run a small smoke test against Redis if practical

Publish steps:

- publish to npm
- verify the package page
- verify install/import from npm works

---

## Go

Repository checks:

- confirm `go.mod` module path
- confirm embedded Lua files are current
- confirm README matches the released API

Execution checks:

- build the library
- build or smoke-test key examples if practical
- test a consumer-facing install flow if practical:
  - `go get <module>@<tag>`

Publish steps:

- push the branch
- create and push the release tag
- verify the tag resolves correctly through `go get`

Note:

Go publishing is tag-driven.
There is no separate package-store publish step like PyPI or npm.

---

## PHP

Repository checks:

- confirm metadata in `composer.json`
- confirm vendored Lua files are current
- confirm README matches the released API

Execution checks:

- run PHP syntax validation
- run Composer install/update as needed
- run a small smoke or contract validation flow if practical
- confirm `ext-redis` and `ext-pcntl` expectations are documented

Publish steps:

- push the branch
- create and push the release tag
- publish to Packagist or the intended Composer distribution path
- verify install through Composer works

---

## Cross-Repo Consistency

Before final sign-off:

- confirm `reserve.lua` is identical in:
  - core
  - python
  - node
  - go
- confirm contract-expansion fields are aligned across SDKs
- confirm examples for the same feature exist where expected
- confirm validation scenarios reflect the released contract
- confirm the full validation suite passes on both standalone Redis and Redis Cluster

---

## Minimum Release Evidence

The release is not complete until you have:

- the final commit hash for each repo
- the final tag for each repo
- validation output for core scenarios
- build output for Python
- build and typecheck output for Node
- build output for Go
- release/install confirmation for PHP
- confirmation that published artifacts install successfully

---

## Suggested Notes Template

For each repo, record:

- repo
- branch
- final commit
- tag
- version
- checks run
- publish command used
- publish result
- post-publish verification result
- notes
