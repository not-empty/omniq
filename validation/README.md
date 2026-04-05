# OmniQ Validation Workspace

This folder is the shared validation workspace for OmniQ core behavior across:

- Python SDK
- Node SDK
- Go SDK

It lives in the contract repository on purpose.

The contract repo should remain the place where we define:

- what must be validated
- which scenarios matter
- what Redis truth should look like
- what "same behavior" means across SDKs

The SDK repos should only receive validation helper scripts when they are truly needed.

---

## Goals

This workspace is meant to validate:

- core queue behavior
- grouped execution behavior
- lease and retry behavior
- admin and maintenance behavior
- child workflow primitives
- monitoring and management views
- cross-SDK parity

This is broader than "publish and consume".

The idea is to validate OmniQ as a system, not only a single happy-path API.

---

## Relationship With Other Docs

Use this folder together with:

- [SDK validation checklist](/Users/disarli/Documents/ops/omniq/docs/sdk-validation-checklist.md)
- [Core contract](/Users/disarli/Documents/ops/omniq/docs/omni-contract.md)
- [Config contract](/Users/disarli/Documents/ops/omniq/docs/CONFIG.md)
- [Redis map](/Users/disarli/Documents/ops/omniq/docs/omniq_redis_map.md)

Recommended reading order:

1. Contract and Redis map
2. Checklist
3. Scenario spec in this folder
4. Actual SDK execution

---

## Layout

Each scenario has its own folder:

- `scenario-01-basic-publish-reserve`
- `scenario-02-heartbeat`
- `scenario-03-ack-success`
- `scenario-04-ack-fail-retry`
- `scenario-05-ack-fail-terminal`
- `scenario-06-promote-delayed`
- `scenario-07-reap-expired`
- `scenario-08-pause-resume`
- `scenario-09-grouped-jobs`
- `scenario-10-retry-remove-admin`
- `scenario-11-child-workflow`
- `scenario-12-monitor-queue-stats`
- `scenario-13-monitor-groups`
- `scenario-14-monitor-lanes`
- `scenario-15-error-surface`

Each scenario folder should eventually contain:

- `README.md`
- optional runner notes or command snippets
- expected Redis keys and expected outcomes
- optional result notes

---

## Execution Model

Validation should run inside the unified Docker environment defined in:

- [docker-compose.yml](/Users/disarli/Documents/ops/omniq/docker-compose.yml)

Typical flow:

```bash
docker compose up -d
docker compose exec omniq-python sh
docker compose exec omniq-node sh
docker compose exec omniq-go sh
```

Redis inside containers:

- host: `omniq-redis`
- port: `6379`

You can also run the suite from one place with:

```bash
./validation/run_suite.sh
```

Or run only selected scenarios:

```bash
./validation/run_suite.sh 25 26 27
```

The runner:

- ensures the Docker services are up
- runs Python, Node, and Go for each requested scenario
- saves each scenario output under `validation/results/<timestamp>/`
- returns non-zero if any scenario runner fails

---

## Validation Philosophy

We want to avoid the "copy of the copy" problem in validation too.

So the process should be:

1. Define scenario once in the contract repo.
2. Execute the same logical scenario in Python first.
3. Port the same scenario to Node and Go.
4. Compare SDK outputs and Redis truth.
5. Only then decide whether there is a bug, contract gap, or acceptable language-specific difference.

Python remains the matrix for SDK behavior,
but Redis plus the contract remain the ultimate source of truth.

---

## Result Logging

For now, keep results lightweight and human-readable.

Use a simple run note format such as:

```text
Scenario:
Queue:
SDK:
Command:
Observed SDK result:
Observed Redis result:
Expected:
Status:
Notes:
```

Later, if needed, we can add machine-readable result files.

---

## When To Add SDK-Side Files

Do not create test files in all SDK repos by default.

Only add SDK-side helpers when:

- a scenario is too repetitive to run manually
- the same command will be used many times
- the helper stays tiny and scenario-focused
- the helper does not become a second source of truth

The main validation design should stay centralized here.
