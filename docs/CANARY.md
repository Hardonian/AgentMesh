# Progressive Delivery & Canary Deployments

## Overview

Upgrading an AI agent involves non-deterministic model weights, updated prompts, and new tool definitions. AgentMesh provides progressive delivery controls to safely test and roll out revisions.

## Lifecycle States

- `DRAFT`: Initial revision authoring.
- `TESTING`: Automated evaluation test suites running against baseline.
- `CANARY`: Staged production traffic rollout (1%, 5%, 10%, 25%, 50%, 100%).
- `ACTIVE`: Fully promoted production version.
- `ROLLED_BACK`: Automatically or manually reverted to previous stable baseline.
- `RETIRED`: Deprecated revision receiving zero traffic.

## Automated Rollback

If a canary candidate breaches defined SLO thresholds (e.g. Error Rate > 5% or P95 Latency > 3,000ms), AgentMesh triggers an **Automated Rollback**:

- Candidate traffic weight is immediately cut to 0%.
- 100% of traffic is restored to the Last Known Good baseline revision.
- An alert and audit event are emitted to the control plane.
