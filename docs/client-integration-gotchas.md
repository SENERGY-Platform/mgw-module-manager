# Client integration gotchas

What a client needs beyond the README. The README's
[Concepts](../README.md#concepts) and [HTTP API](../README.md#http-api) sections
describe the change request, the job model and the endpoints; this document only
carries what they do not say and what a specification cannot express.

Everything below was verified against a running core, not read from the docs.

## Applies when

Writing or maintaining a client against the **next-gen** API (this repository's
`main`, installer branch `next-gen`).

**Not this if**: the previous, pre-next-gen API — none of it applies there.

## Polling: `end` at the Go zero value means still running

A job is `{id, description, start, end}`. While it runs, `end` stays at
`0001-01-01T00:00:00Z` rather than being absent or null. A client that checks for
a missing field never sees the job finish.

Once it is set, fetch the outcome from `/results/<type>/{JOB_ID}` — the type has
to match the operation, and a recreate writes into `deployments`.

## Enable and disable return before the containers move

`POST /deployments-enable` and `-disable` only flip the `enabled` flag and return
the deployment IDs immediately. The runtime monitor starts or stops the actual
containers afterwards, in a loop of roughly 5s.

So a client must not read container state from that response. Poll
`GET /health/deployments`, or the `state` in `modules-reduced`: `1` healthy,
`2` unhealthy, `0` disabled or undetermined.

## The encoding depth depends on the path taken

Module IDs contain slashes, and the gateway decodes one level:

- **through the nginx gateway** — path parameters must be **double** URL-encoded
- **directly against the service** — single encoding

This repository's own clients demonstrate both: `test_client` goes through the
gateway and uses escape depth 2, the internal `lib/clients` use depth 1. Query
parameter lists (`module_ids`, `ids`) are CSV (`collection_format:"csv"`).

Guessing the depth from the wrong example produces a 404 on a path that exists.

## "Deployment update pending" has to be derived

There is no such flag. `GET /modules-reduced` carries, per module, the
`module_version` the deployment was created for; comparing that against the
installed version is what yields the state.

## Container logs need a second service

This service exposes no log endpoint. `DeploymentInfo.containers` (in
`GET /modules/{MOD_ID}`) maps service references to container metadata including
the container **name**; the log stream then comes from the container-engine
wrapper at `GET /core/api/ce-wrapper/logs/{container-name}` — that endpoint
accepts names, not only IDs.

## Wire quirks, reported upstream

A client cannot be written correctly without these. The first one's fix will be a
breaking change:

- **Empty slices serialize as `null`**, not `[]` — for example `channel_errors`
  and `tags`. Always null-check.

## Two smaller ones

- A module is only removed while it has **no** deployment. Delete the deployment
  first, otherwise the item fails with `deployment exists`.
- Per-module failures land in `failed[]` of the result; **the job itself still
  completes successfully.** A client that only checks the job status reports
  success for a change that partly did not happen.
