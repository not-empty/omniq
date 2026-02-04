# OmniQ Core (Redis + Lua) — v1

OmniQ is a **language-agnostic queue core** implemented in **Redis + Lua**.
Language SDKs embed these Lua scripts and must follow the contract.

## What’s in this repo

- `scripts/` — Lua scripts that implement the core protocol
- `docs/` — the v1 contract and configuration docs
  - `docs/omni-contract.md` — **normative** client contract
  - `docs/CONFIG.md` — configuration contract and semantics

## Versioning

This repo is the **source of truth** for:
- Lua script behavior
- contract/docs

Release tags follow SemVer, e.g. `v1.0.0`.

**SDK repos must pin a tag** of this repo (recommended: Git submodule).
