# Demo Services Redesign: The "Ship It" Trio

A proposal for the three demo services behind the kaja.tools live demo — one Twirp,
one gRPC, one OpenAPI — designed from scratch so that each protocol is shown where it
shines, and so that a single script in the kaja editor can chain all three into one
story.

## Why redesign at all

The current demo pair (users = Twirp, teams = gRPC) works, but it has three
limitations as a demo:

1. **It's static.** Nothing happens unless the visitor types something. A fresh
   visitor's first `GetAllUsers` call returns whatever the last stranger left behind.
2. **It's all unary CRUD.** The two protocols look interchangeable — the demo never
   answers "why would I have both gRPC *and* Twirp *and* REST in one shop?", which is
   exactly the situation kaja exists for.
3. **The composition is thin.** The only cross-service story is "put a user id into a
   team" — a foreign-key join, not a workflow.

Adding OpenAPI is the moment to fix this: pick a domain where the three protocols
have naturally different jobs, and where the output of one call is the natural input
of the next.

## The proposal: an internal CI/delivery platform

The fictional "internal microservices group" becomes the delivery platform of a small
software company. Three services around the act of shipping software:

| Service | Protocol | Role | Why this protocol |
|---|---|---|---|
| `builds` | **Twirp** | CI orchestrator: trigger, list, cancel, retry builds | Twirp's sweet spot: boring internal request/response commands, JSON-curl-able |
| `logs` | **gRPC** | Log pipeline: tail, follow, search build logs | Streaming is gRPC's unique power, and `tail -f` is the most intuitive streaming API there is |
| `releases` | **OpenAPI** | Artifact registry / release catalog | Registries are REST-shaped in real life (GitHub Releases, npm, Docker registry): resources, path params, pagination, 404s |

This mirrors how real companies actually mix protocols — internal RPC commands over
Twirp, high-throughput streaming over gRPC, and a resource catalog over REST — so the
demo doubles as the *argument* for kaja.

The domain is also self-referential in a charming way: the seeded projects are
`website`, `kaja`, and `quirks` — the demo builds the very software you're looking at.

### The money-shot script

The demo the landing page screenshots should show — one script, three protocols, one
real workflow, with live streaming in the middle:

```ts
// Kick off a build (Twirp)
const { build } = await builds.createBuild({ project: "website", branch: "main" });

// Tail its logs live as it runs (gRPC server streaming)
for await (const line of logs.tail({ buildId: build.id, follow: true })) {
  console.log(line.text);
}

// The build passed — fetch the release it published (OpenAPI)
const { build: finished } = await builds.getBuild({ id: build.id });
const release = await releases.getRelease({ project: "website", version: finished.version });
console.log(release.changelog);
```

Every step's output feeds the next step's input, the middle section visibly streams
for ~30 seconds, and the ending is a satisfying artifact. No CRUD demo can match the
"it's alive" feeling of log lines arriving in real time in an editor.

## Service designs

### `builds` (Twirp)

```proto
syntax = "proto3";
package builds;

service Builds {
  rpc GetAllBuilds(GetAllBuildsRequest) returns (GetAllBuildsResponse); // newest first, filter by project/status
  rpc GetBuild(GetBuildRequest) returns (GetBuildResponse);
  rpc CreateBuild(CreateBuildRequest) returns (CreateBuildResponse);    // project + branch -> queued build
  rpc CancelBuild(CancelBuildRequest) returns (CancelBuildResponse);
  rpc RetryBuild(RetryBuildRequest) returns (RetryBuildResponse);       // failed build -> new build
  rpc GetProjects(GetProjectsRequest) returns (GetProjectsResponse);    // seeded: website, kaja, quirks
}

message Build {
  string id = 1;
  string project = 2;
  string branch = 3;
  BuildStatus status = 4;          // QUEUED, RUNNING, PASSED, FAILED, CANCELED
  string commit_sha = 5;           // fake but plausible
  repeated BuildStep steps = 6;    // checkout, deps, compile, test, package
  string version = 7;              // semver, set when PASSED
  string created_at = 8;
  string finished_at = 9;
}

message BuildStep {
  string name = 1;
  BuildStatus status = 2;
  int32 duration_ms = 3;
}
```

**The simulation engine is the heart of the demo.** `CreateBuild` starts a goroutine
that walks the build through its steps over ~30–60 seconds (randomized), emitting
crafted log lines into the `logs` service as it goes. Roughly 15% of builds fail on a
flaky test — failures are part of the demo (they give `RetryBuild` a purpose and give
the REST API a natural 404 to show). On success, the build mints the project's next
semver version and `POST`s a release to the `releases` service — the services
*really* call each other, so the microservices story is true, not staged.

A background **release train** creates a build on a random project every few minutes.
This is what fixes the "static demo" problem: a visitor's very first `GetAllBuilds`
returns a believable, populated history, and there's usually a build running *right
now* to tail.

### `logs` (gRPC)

```proto
syntax = "proto3";
package logs;

service Logs {
  // Server streaming: the tail -f of the demo. follow=true keeps the stream open
  // while the build runs; on a finished build it replays instantly and closes.
  rpc Tail(TailRequest) returns (stream LogLine);
  // Unary grep across a build's (or all recent) logs.
  rpc Search(SearchRequest) returns (SearchResponse);
  // Client streaming: how a build agent pushes lines. Used by the builds service
  // for real, and demoable by visitors ("be your own CI agent").
  rpc Push(stream LogLine) returns (PushResponse);
}

message LogLine {
  string build_id = 1;
  int64 seq = 2;
  string timestamp = 3;
  Level level = 4;      // DEBUG, INFO, WARN, ERROR
  string step = 5;      // which build step emitted it
  string text = 6;
}
```

`Tail` with `follow: true` on a RUNNING build is the wow moment. The fake log lines
deserve real craft — plausible compiler/test-runner output with the occasional
easter egg — because streaming log output is what people will screenshot.

Covering server streaming, client streaming, and unary in one small service also
replaces most of what `quirks` v1 demonstrates, but with meaning attached.

### `releases` (OpenAPI 3.1)

Spec-first, mirroring the proto-first ethos: `apps/releases/openapi/openapi.yaml`
is the source of truth, Go server generated with `oapi-codegen` via a `scripts/openapi`
sibling of `scripts/protoc`. The spec itself is served at
`https://kaja.tools/releases/openapi.yaml` — that URL is what kaja loads.

```
GET    /projects                                  → all projects with latest version
GET    /projects/{project}/releases               → paginated (?page, ?per_page)
GET    /projects/{project}/releases/latest
GET    /projects/{project}/releases/{version}
GET    /projects/{project}/releases/{version}/artifact   → binary .tar.gz download
POST   /projects/{project}/releases               → 201 (called by builds; 409 on duplicate version)
DELETE /projects/{project}/releases/{version}     → 204 ("yank a release")
```

The schema deliberately exercises things protobuf can't express, so the OpenAPI
support in kaja gets a real workout:

- `version` with a semver `pattern`, `format: date-time` timestamps, `format: uri` links
- a markdown `changelog` field (generated from the build's fake commits)
- `artifacts[]` with `sha256` and `size_bytes`
- string enums (`status: published | yanked`)
- RFC 7807 `application/problem+json` error bodies for 404/409 — showing how kaja
  renders REST errors next to Twirp errors and gRPC status codes
- pagination via query params + `Link` header
- one binary (non-JSON) response

The link back is `build_id` on every release — REST → Twirp composition in the
reverse direction (`releases.getRelease(...)` → `builds.getBuild({ id: release.buildId })`
→ `logs.tail(...)`: "why does v1.4.2 crash? read its build logs").

## What happens to users, teams, and quirks

- **users, teams**: retire from the public demo (`apps/kaja/kaja.json`) once the trio
  is live; delete the apps in a follow-up once nothing references them. They taught
  us the patterns (pebble storage, Twirp path prefixes, gRPC ingress) that the trio
  reuses.
- **quirks**: keep, but as the edge-case test harness it already is rather than part
  of the headline demo. Its streaming RPCs, odd names, and v1/v2 packages are for
  exercising kaja itself. Optionally keep it in the demo config as a fourth,
  clearly-labeled "kitchen sink" project.

## Infrastructure plan

Everything follows existing patterns in this repo:

- `apps/builds` — Twirp server with `twirp.WithServerPathPrefix("/builds/twirp")`,
  pebble DB on the workspace volume, `k8s/base/builds.yaml`, path `/builds` in
  `k8s/base/ingress.yaml`.
- `apps/logs` — gRPC server, `k8s/base/logs.yaml`, path `/logs.Logs/` in
  `k8s/base/ingress-grpc.yaml`. If teams is retired, repoint the gRPC reflection
  paths (currently → `teams-service`) at `logs-service`.
- `apps/releases` — plain HTTP server (chi + oapi-codegen), `k8s/base/releases.yaml`,
  path `/releases` in `k8s/base/ingress.yaml`, serving both the API and
  `/releases/openapi.yaml`.
- `apps/kaja/kaja.json` — three projects: `builds` (`RPC_PROTOCOL_TWIRP`,
  `https://kaja.tools/builds`), `logs` (`RPC_PROTOCOL_GRPC`, `dns:kaja.tools:443`),
  `releases` (the new OpenAPI protocol enum, pointed at the spec URL).
  *Assumption: kaja's config gains an OpenAPI protocol value; the exact enum name
  follows whatever kaja ships.*
- Inter-service calls (`builds → logs`, `builds → releases`) go over in-cluster
  service DNS using the services' own public APIs — dogfooding, and it keeps the
  "these are real microservices" story honest.

### Public-demo hygiene

The demo is on the open internet, so the trio is self-maintaining:

- Seed on boot if the DB is empty; the release train keeps data fresh.
- Rolling GC: keep the last ~200 builds and their logs, ~50 releases per project.
- Caps: max concurrent simulated builds (e.g. 5), length limits on all user-supplied
  strings, a modest global rate limit on `CreateBuild`.
- Everything user-created is disposable by design — a nightly reset is acceptable
  and invisible because seeded data looks the same as organic data.

## Alternatives considered

- **Keep users/teams, add an OpenAPI booking/rooms service** ("book a meeting room
  for a team"). Cheapest option and a coherent story, but it stays all-unary CRUD —
  it never shows streaming, never shows *why* the protocols differ, and the demo
  stays static.
- **Delivery/logistics** (orders via Twirp, courier GPS stream via gRPC, catalog via
  REST). Understandable to everyone, but streaming coordinates are just numbers;
  streaming *log lines* are self-explanatory and native to the audience.
- **Observability** (metrics stream, incidents, status page). Good fit but overlaps
  with what real vendors demo; CI is closer to kaja's own build-and-ship identity.

The CI trio wins because kaja's audience is developers who live in exactly this
workflow, and because it gives each protocol a job only it can do.

## Suggested build order

1. `releases` (OpenAPI) alone with hand-seeded data — it's the new protocol and can
   ship to the demo first as a standalone project.
2. `builds` + the simulation engine + release train, POSTing to `releases`.
3. `logs` + wire the simulator's log lines through `Push`/`Tail`.
4. Swap `apps/kaja/kaja.json` to the trio, retire users/teams from the demo, update
   ingress and reflection routing.
5. Refresh landing-page screenshots/demo video around the money-shot script.
