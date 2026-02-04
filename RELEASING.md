# Release process (Core)

This repo is the authoritative source for OmniQ v1 behavior.

## Tagging

1. Update docs and scripts (everything in `docs/` and `scripts/`).
2. Ensure the contract still matches behavior.
3. Create a tag:

```bash
git tag -a v1.0.0 -m "OmniQ core v1.0.0"
git push origin v1.0.0
```

## What SDKs should do

SDK repositories should pin `scripts/` via **submodule** to a tag of this repo.

Example (inside an SDK repo):

```bash
git submodule add https://github.com/<ORG>/omniq-core.git scripts
cd scripts
git checkout v1.0.0
cd ..
git add .gitmodules scripts
git commit -m "Pin omniq-core scripts to v1.0.0"
```

When you release `v1.0.1`, SDKs can choose when to bump.
