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

SDK repositories should copy `scripts/` into lua script folders