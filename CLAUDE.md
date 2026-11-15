# catalog: standing rules

Bundt's seed catalog: Helm charts that port open-source Terraform modules
(Cloud Foundation Fabric, terraform-aws-modules, Azure Verified Modules) to
Config Connector, ACK and Azure Service Operator objects, each with its
Bundt Package (`charts/<cloud>/<module>/`). Run `task check` before every
commit (gofmt and vet of `tools/`, the Go tests, the chart and example
tests, coverage current, shell and workflow lint); `task ci` runs the same
at CI's tool pins and posts the `offload/ci` status, and the pre-push hook
(`task hooks`) runs it on every push.

<!-- FACTORY-COORD: BEGIN -->
## Multi-Agent & Multi-Human Coordination (`coord` + Linear)

Every repository under `highfivelabs` shares one live coordination ledger and browser dashboard (`coord`, or its `factory coord` alias, living in the standalone `coord` repo, backed by Cloud Run + Firestore or local daemon at `http://127.0.0.1:8090`, and synced to Linear). Multiple agents (Claude Code, Antigravity, Codex, remote VM workers) and humans work across these repositories concurrently.

### How Linear Fits With `coord`

| Concern | System of Record | Rule for Agents |
| --- | --- | --- |
| **Backlog, priority, problem statement, acceptance criteria** | **Linear** (`HFL-142`, etc.) | Always anchor your session to a Linear issue ID when one exists. Include the issue ID in your branch name (`feat/hfl-142-...`) and PR title so Linear links the PR automatically. |
| **Live lease lock, 60s heartbeats, active window/worktree, Spot VMs, token spend, nudges** | **`coord`** (`http://127.0.0.1:8090` / Cloud Run + Firestore) | Use `coord` (`coord ...` or `factory coord ...`) for all live locking, `-note` progress updates, token tracking, and Spot VM tracking. **Never spam Linear with step-by-step progress comments.** |
| **Milestone transitions (`claim`, `handoff`, `resume`, `release -complete`)** | **Both** (`coord` syncs to Linear automatically) | Running `coord claim HFL-142` fetches the Linear issue title/URL, moves unstarted issues to `In Progress`, and posts a claim comment. Running `coord handoff` or `coord release -complete` posts a structured baton/completion summary (including token usage) to Linear automatically. |

### Standing rules for every session

1. **Check before starting.** At the start of a task, run `coord status` (or `coord status -repo <repo>`) to see what tickets, branches, scopes, and cloud VMs are actively claimed or waiting in `handoff_ready`.
2. **Claim your work and broadcast current activity.** Before editing files or launching cloud resources, claim your Linear ticket (for example `HFL-142`) or repository scope (only use `repo:scope` like `bundt:apps/bundt/service` for untracked maintenance):
   ```sh
   coord claim HFL-142 -note "Implementing Firestore store"
   ```
   This starts a background heartbeat daemon tied to your session PID (which renews your lease every 60s and automatically reports session token usage every 3rd heartbeat), moves the Linear issue to `In Progress`, and writes `.coord/session.json`. Whenever you move to a new phase, update your live dashboard status with `coord heartbeat -note "Running task check"` (you can also pass `-tokens-in N -tokens-out N -cost-usd USD` or write `.coord/usage.json`).
3. **Never step on an active lease silently.** If `coord claim` exits with a conflict (`409 Conflict`, exit code `3`), another live session holds that work item. Choose one of four explicit paths:
   - **Bow out (default):** Leave the ticket to the active holder and pick another unclaimed ticket (`coord status`).
   - **Adversarial pairing:** Work on the same ticket in a **separate git branch and worktree** (for example competing implementation, critic/reviewer, or Spectacle behavior-guard vs code builder):
     ```sh
     coord claim HFL-142 -mode adversarial -role challenger
     ```
   - **Collaborative pairing:** Work on a **disjoint directory scope** within the same repository:
     ```sh
     coord claim HFL-142 -mode collaborative -scope apps/bundt/web
     ```
   - **Steal / Take over:** Only when instructed by a person or when taking over stalled work:
     ```sh
     coord claim HFL-142 -steal -reason "taking over on user request"
     ```
4. **Watch for revocations and human nudges from the dashboard.**
   - If `.coord/REVOKED` appears in your checkout or `coord heartbeat` reports `lease_state: revoked`, another session or person has taken over the lease. Stop editing immediately.
   - If `.coord/NUDGE` appears in your checkout (sent from the `http://127.0.0.1:8090` dashboard or `coord nudge`), read `.coord/NUDGE` immediately, follow the person's guidance (or run `coord handoff` if `HANDOFF_REQUESTED`), and remove `.coord/NUDGE` once handled.
5. **Distribute heavy work to Spot / Preemptible VMs and record them.** Whenever offloading builds, test suites, mutation runs, or code generation to cloud VMs (GCP Spot in `hfl-dev-514205` / `hfl-test-g2`, AWS Spot, or Azure Spot where free credits apply), attach the VM to your claim so the dashboard shows which VMs are in use:
   ```sh
   coord vm-attach -provider gcp -name offload-runner-01 -zone us-central1-a -spot -task "task check:mutation"
   # When the VM finishes or is torn down:
   coord vm-detach -name offload-runner-01
   ```
   When running an agent directly *inside* a Spot/Preemptible VM, `coord claim` automatically detects the cloud VM metadata and watches the cloud preemption endpoint during heartbeats to auto-handoff if preempted.
6. **Pass the baton on usage limits or unfinished stops.** If you approach model usage/rate limits, context window limits, spot preemption, or must stop before completing the ticket, push your WIP branch and record a structured handoff baton (which also posts the baton to Linear):
   ```sh
   coord handoff -reason usage_limit -push \
     -done "Implemented Firestore store and unit tests" \
     -next "Wire Cloud Run deploy script and run task check"
   ```
   To pick up a `handoff_ready` or `stale` ticket in a new harness (for example switching from Claude Code to Antigravity or Codex), run:
   ```sh
   coord resume HFL-142
   ```
7. **Release when done.** When the PR is opened or the task is complete:
   ```sh
   coord release -complete -summary "Opened PR #42; all checks green"
   ```
<!-- FACTORY-COORD: END -->
