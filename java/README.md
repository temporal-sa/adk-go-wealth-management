# Java Open Investment Account Worker

An optional Java (Spring Boot) implementation of `OpenInvestmentAccountWorkflow`, the child workflow
the Open Account Agent starts to open a new investment account. It is a drop-in replacement for the
Go version in [`internal/workflows/openaccount.go`](../internal/workflows/openaccount.go): same KYC
and compliance steps, same saga (create the account record, debit the linked bank account, register
with the custodian, with compensations in reverse order), and the same status updates signalled to the
parent chat workflow.

This folder is copied unchanged from the reference port
([temporal-sa/oai-temporal-wealth-management](https://github.com/temporal-sa/oai-temporal-wealth-management)),
and every port shares it. Some code comments still say the activities are in Python; in this port
they are the Go worker's.

The workflow has no business logic of its own. It calls the activities by name (`get_client`,
`open_investment`, `remove_investment`, `debit_linked_account`, `credit_linked_account`,
`register_account`, `update_client`) on the Go worker's task queue (`TEMPORAL_TASK_QUEUE`, default
`Supervisor`), so the Go worker must be running too, and the chaos panel's flags apply unchanged. The
only Java activity is the one that signals status changes to the parent workflow; it runs on the Open
Account task queue with the workflow.

## Prerequisites

* [Java JDK 25](https://adoptium.net/). The Gradle wrapper (`./gradlew`, Gradle 9.8.2) is included, so
  no separate Gradle install is needed.
* Everything the Go demo needs (Go, Temporal CLI, Redis, Gemini API key). See the
  [root README](../README.md).

## Running locally

Start Temporal (`temporal server start-dev`) and Redis (`redis-server`).

The Open Account task queue name is passed as the first argument to both workers. Start the Go worker
with it, from the repository root, so the chat starts the child workflow on that queue (and the Go
worker doesn't run its own version of the workflow):

```bash
./scripts/startlocalworker.sh Open-Account-Java
```

Then start the Java worker with the same name:

```bash
cd java && ./startlocalworker.sh Open-Account-Java
```

The script sets `TEMPORAL_TASK_QUEUE_OPEN_ACCOUNT` from the argument and runs `./gradlew runWorker`.
It connects to `127.0.0.1:7233`, namespace `default`, as set in `src/main/resources/application.yaml`.

Start the API and the web UI as described in [docs/temporal.md](../docs/temporal.md), and send the
compliance approval with `./scripts/localsendcomplianceapproval.sh` as usual.

To try the Java workflow without the chat or a Gemini key, run the open-account scenarios against it,
with this worker running, from the repository root:

```bash
./scripts/startlocalopenaccount.sh Open-Account-Java
```

See "Testing the open-account workflow on its own" in [docs/temporal.md](../docs/temporal.md).

## Running with Temporal Cloud

Create `setcloudenv.sh` in the repository root (copy `setcloudenv.example`), as described in
[docs/temporal.md](../docs/temporal.md#running-the-demo-on-temporal-cloud). Start the Go worker with
the queue name, then the Java worker with the same name:

```bash
./scripts/startcloudworker.sh Open-Account-Java
```

```bash
cd java && ./startcloudworker.sh Open-Account-Java
```

The Java cloud script sources `../setcloudenv.sh`, maps `TEMPORAL_TLS_CLIENT_CERT_PATH` to the
`TEMPORAL_CERT_PATH` the Java worker reads, uses the Spring profile `tc`
(`src/main/resources/application-tc.yaml`), and runs `./gradlew runWorker`.

Java's TLS needs the client key as an unencrypted PKCS8 key (`-----BEGIN PRIVATE KEY-----`). If
`TEMPORAL_TLS_CLIENT_KEY_PATH` points to a different format, the script stops and prints the command
to convert it, for example:

```bash
openssl pkcs8 -topk8 -nocrypt -in /path/to/key.key -out /path/to/key.pkcs8.key
```

Then add the converted key to `setcloudenv.sh` (the Go components keep using
`TEMPORAL_TLS_CLIENT_KEY_PATH`):

```bash
export TEMPORAL_KEY_PKCS8_PATH="/path/to/key.pkcs8.key"
```

## Bug in the workflow code (chaos panel)

The chaos panel's **Bug in workflow code** switch gives new account openings the promo code
`WELCOME-BONUS`. In `src/main/java/io/temporal/demo/wealthmanagement/workflow/OpenInvestmentAccountWorkflowImpl.java`,
`BONUS_RATES` is keyed by `"WELCOME_BONUS"` (underscore), so `welcomeBonus` throws a
`NullPointerException` right after compliance approval. The exception fails the workflow task, not the
workflow, so the workflow stays **Running**, paused at that line. To fix it, use the commented-out
`BONUS_RATES` line above it and restart the Java worker; the workflow resumes on the next
workflow-task retry. Re-arm the bug afterwards, from the repository root:

```bash
git checkout java/src/main/java/io/temporal/demo/wealthmanagement/workflow/OpenInvestmentAccountWorkflowImpl.java
```

See [Failure scenarios](../docs/temporal.md#failure-scenarios) for the other scenarios.

## Claim check

The start scripts source `../setclaimcheck.sh` if it exists, so with the claim check on (see
[docs/temporal.md](../docs/temporal.md#claim-check-optional)) the Java worker uses the same Redis
claim-check codec as the Go components. Locally, it uses the Redis at `spring.data.redis` in
`application.yaml` (`localhost:6379`), not `REDIS_HOST` / `REDIS_PORT`; with the Cloud profile, it uses
`REDIS_HOST` / `REDIS_PORT`.

## Tests

The saga, retry and workflow-bug tests use the Temporal Java SDK's in-memory test environment with
stub activities, so no Temporal server, Redis or Gemini key is needed:

```bash
cd java && ./gradlew clean build && ./gradlew test
```
