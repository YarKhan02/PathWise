# PathWise — project context

Read this before writing code. It is the authoritative description of what
PathWise is, what has been decided, and what must not be re-litigated without
a deliberate decision.

---

## 1. What we are building

PathWise is a Go static-analysis CLI that **traces execution and authorization
paths through a Go web application** rather than merely listing access rules.

The one claim the tool has to earn:

> For endpoint **E** and identity **I**, these database rows and these response
> fields are reachable — and here is the code or the HTTP response that
> **proves** it.

Everything else — the graph model, the dashboard, the git diff, the storage
layer — is presentation or persistence around that claim.

It has three responsibilities, kept as separate modules over one shared data
model:

1. **Static analysis** — routes, call chains, authorization logic, SQL
   queries, response fields.
2. **Runtime verification** — exercise the running app as different
   identities, compare observed behavior with what static analysis predicted.
3. **Graph and evidence storage** — combine both into an authorization graph
   where every relationship cites source code or a runtime observation.

The dashboard must never need to understand Go syntax, and the runtime tester
must never parse source itself. Both consume the shared model.

### The hard problem, stated plainly

The original architecture note contained this sentence:

> "The source analyzer also needs to distinguish a function being called from
> a function's result actually controlling access."

**That sentence is the entire project.** Finding `CanAccessStudent` in a
handler is a grep. Proving that its `false` branch *dominates every path* from
handler entry to the response write is SSA dominance analysis. The same
applies to query predicates: knowing whether the value in `WHERE id = $1`
traces back to request input or to the authenticated identity is
interprocedural value tracking across package boundaries.

This is Phase 3. It is months of work, not weeks. Do not let the surrounding
scaffolding create the impression the project is nearly done when Phase 2
lands.

### Non-negotiable principle

**Use Go's AST and type information, never regular expressions.** Regexes can
find candidate strings; they cannot follow calls, resolve symbols, or
understand control flow. A name match is not evidence.

---

## 2. Current state

**Phases 0–6 have landed; Phase 7 (breadth) is what remains.** `pathwise scan`
recovers the route table (method, pattern, handler, middleware) from SSA, reads
the live PostgreSQL schema from `information_schema`, and — via a CHA→VTA call
graph — maps each endpoint to the tables it reaches with the operations it
performs (e.g. `POST /login` writes `sessions`, reads `users`). A query built at
runtime is recorded as `dynamic_sql`, never silently skipped. On top of that it
proves authorization semantics (dominance + predicate provenance) to emit the
four findings, renders them as text/JSON/SARIF, serves an embedded React
dashboard (`pathwise serve`), and verifies conclusions against a running app
(`pathwise test`), storing runtime results apart from the static ones. What is
left: Phase 7 — more framework adapters, GORM, git diff between scans, CI mode —
plus two carried-forward items (migration-based schema reconstruction already
landed for `--allow-no-db`; the deferred filtered-subgraph dashboard
optimization).

| Phase | Scope | State |
|---|---|---|
| 0 | Skeleton, data model, storage, corpus | **done** |
| 1 | Live PostgreSQL schema + Gin route discovery | **done — routes, live schema (pgx), and migration-based reconstruction (pg_query_go) for `--allow-no-db`, marked inferred** |
| 2 | SSA call graph (CHA→VTA), endpoint → tables | **done — endpoint → tables/ops mapped; table names by keyword scan, dynamic queries flagged** |
| 3 | Authorization semantics: dominance + predicate provenance | **done — all 5 seeded corpus flaws detected, 0 false positives. role_check_unenforced (×2), missing_tenant_filter, response_field_exceeds_scope, and missing_ownership_check (IDOR, active with --db)** |
| 4 | Findings and CLI output | **done — findings produced during the scan; `scan --format json\|sarif` emits machine output (SARIF 2.1.0 for CI); text output cites file:line** |
| 5 | Runtime verification | **done — `pathwise test` exercises a running app as developer-supplied identities (`--identity role=token`), compares observed HTTP behavior to the static role prediction, emits five outcomes (pass/fail/divergent/blocked/unverified), persists each scenario+run to `test_scenarios`/`test_runs` (append-only, stored apart from static findings). GET-only safety with per-endpoint non-GET opt-in (`--allow "METHOD /path"`); parameterized paths left unverified; tokens never serialized or stored** |
| 6 | Dashboard | **done — React+TS+Cytoscape SPA embedded via embed.FS, served by `pathwise serve`; Overview/Endpoints/Access-Graph/Database/Findings/Inspector/Runtime-Tests all live (Runtime Tests reads the latest stored run via `/api/scan`). Scan History / Git Comparison remain honest stubs awaiting Phase 7; filtered-subgraph-on-selection still deferred (full graph fine at this size)** |
| 7 | More adapters, GORM, git diff, CI | |

### Verified

- `go build ./...`, `go vet ./...`, `gofmt -l .` all clean
- `go.sum` generated (`go mod tidy`); the corpus app type-checks
  (`make corpus-tidy && cd testdata/apps/basic-gin && go build ./...`)
- All tests pass, including the new `internal/adapters/gin` suite, which
  recovers all 8 corpus routes with correct handlers and middleware order and
  no false positives (the Phase 1 route-discovery exit criterion)
- `internal/schema` live reader recovers all 5 corpus tables, their columns,
  primary keys and 5 explicit foreign keys against a real PostgreSQL — the
  Phase 1 schema exit criterion. Its test (`TestReadLiveMatchesCorpus`) needs a
  database: set `PATHWISE_TEST_DB` to a libpq DSN for an empty database, and it
  applies `migrations/0001_init.up.sql` itself; without that env var it skips,
  so `go test ./...` stays green with no database present.
- `internal/analyzer` builds the CHA→VTA call graph and, from each route's
  handler plus its middleware, maps the tables reached with their operations.
  `TestEndpointTableAccess` asserts the correct table set for all 8 corpus
  endpoints and that `/search/students` surfaces a `dynamic_sql` unresolved
  site — the Phase 2 exit criterion. Needs no database.
- `internal/authorization` proves, by SSA dominance over a handler, whether an
  authorization check actually gates the response. `TestCheckEnforcement`
  asserts the corpus's two seeded `role_check_unenforced` flaws read as
  unenforced (IsAdmin discarded; HasRole non-dominating) and the negative
  control (CanAccessStudent, abort dominates `c.JSON`) reads as enforced. The
  scan emits `role_check_unenforced` findings for the two flaws and none for
  the control.
- `internal/dataflow` parses SQL with pg_query_go (tables, WHERE predicates);
  `internal/findings` has the rules. On the corpus the scan now detects **4 of
  5 seeded flaws** — both role checks, `missing_tenant_filter` (batches read
  with no org_id predicate), and `response_field_exceeds_scope` (national_id
  serialized on `/students/{id}/profile`, which declares a scope) — with **no
  false positives** (verified: no tenant finding on the key-fetch of students,
  no scope finding on `/students/{id}/safe` which declares no scope). Rules are
  unit-tested in `internal/findings`.
- `missing_ownership_check` (IDOR) is done. `analyzer.Provenance` classifies a
  value as request / identity / constant / unknown, tracing a query helper's
  parameter back to the arguments its callers pass (`c.Param("id")` → request).
  `authorization.HasOwnershipGuard` detects an ownership guard — a branch on an
  identity-derived value that dominates the response write, covering both
  `CanAccessStudent` and the inline `user.OrgID != p.OrgID`. The rule fires when
  a request value reaches a primary/unique-key predicate (keys from the live
  schema, so IDOR needs `--db`), no identity predicate scopes the query, and no
  guard dominates — gated to authenticated endpoints and handler-reached queries
  so the public login's unique-email lookup and the auth middleware's
  session-token lookup never false-positive. `TestOwnershipCheck` covers the
  seeded IDOR and the four negatives with the schema keys supplied directly (no
  DB), and a `scan --db` on the corpus reports all 5 seeded flaws.
- `internal/runtime` plans and executes runtime scenarios and classifies each
  into the five Phase 5 outcomes. `TestRunClassifiesOutcomes` (httptest mock)
  covers all five, including the IDOR-style divergence (a student reaching an
  admin-only endpoint a broken app serves); `TestNonGetOptIn` proves a DELETE is
  blocked by default and fires only when `--allow`ed; `TestFailWhenPredicted...`
  covers the predicted-reachable-but-denied case. `sqlite.InsertTestResults` /
  `LatestTestResults` persist a run as test_scenarios+test_runs rows and read
  back only the newest batch (`TestTestResultsRoundTripAndLatest`). Verified end
  to end: `pathwise test` against a live mock persists to the store, and a fresh
  `serve` returns the run in `/api/scan.runtimeTests` for the dashboard page.
- CLI exercised end to end, including `scan --db` against a live database:
  discovers routes, reads schema, maps endpoint → tables/operations, refuses to
  guess a schema source, reports active vs inactive checks, persists a scan row,
  stubs name their phase

### Resolved since Phase 0

- **The corpus app did not compile.** `StudentProfile` was declared both as a
  type (`store.go`) and as a handler func (`handlers.go`); Go forbids that. The
  handler is now `GetStudentProfile` (consistent with `GetStudent`), updated in
  `handlers.go`, `main.go` and both references in `expected.yaml`. This had to
  be fixed before anything using `go/packages` could analyze the corpus.
- **Module path.** `github.com/YarKhan02/pathwise` is correct for this account;
  no rename needed.

### Phase 2 known limits (carried forward)

- **Table names come from a keyword scan, not a parse.** `analyzer.readQuery`
  reads the leading verb and the identifier after FROM/JOIN/INTO/UPDATE. It is
  marked `shortcut:` and is enough for the corpus and ordinary hand-written SQL,
  but it does not understand subqueries, CTEs, quoted or schema-qualified names,
  and reads no predicates. Phase 3's `pg_query_go` parse tree replaces it and
  also feeds the WHERE-clause provenance analysis this omits.
- **Only `dynamic_sql` query sites are reported as unresolved so far.** The
  corpus's reachable DB paths are entirely static calls, so there are no
  unresolved *call* sites (interface dispatch, reflection) to name yet. When a
  corpus app grows an interface-dispatched DB path, extend `AnalyzeReachableDB`
  to record those with `interface_dispatch`.
- **Closures reached only by being returned are not walked.** `RequireRole`
  returns its middleware as a closure; the closure's body is not traversed from
  `RequireRole` (it has no DB access, so the corpus is unaffected). Revisit if a
  closure-based middleware ever touches the database.

### Running on AcademyEngine (the real target)

AcademyEngine (`/Users/yarkhan/Tech/Academy/AcademyEngine`, a reference, not
part of this repo) is the project PathWise is meant to run on, and it is a close
shape match: Gin v1.12, grouped routes with `middleware.Auth(...)` +
`middleware.RequireRole("...")` on each group, string roles (`admin`, `owner`,
`teacher`, `student`, `creator`), raw `database/sql` with `$N` placeholders, no
ORM. Both discovery gaps are now closed; the tool should run on AcademyEngine
(routes, roles via group middleware, endpoint → tables). The SQL-dependent
*findings* (IDOR, tenant, scope) still await Phase 3's pg_query_go work.

1. **Routes register through a parameter — DONE.** Each domain exposes
   `func (m *Module) RegisterRoutes(r *gin.Engine)` and the server loops over
   modules calling `m.RegisterRoutes(r)`. The adapter now treats a `*gin.Engine`
   / `*gin.RouterGroup` **parameter** as a router root (base "/"), scans methods
   and closures as well as package-level funcs (via `ssautil.AllFunctions`), and
   resolves a method-value handler (`m.h.GetCourse`, a bound-method thunk) to the
   real method. `TestRouterParameterModulePattern` covers this shape.
   Remaining nuance (marked `shortcut:` in `node()`): a `*RouterGroup`
   parameter that is really a sub-group has its ancestor prefix/middleware
   under-reported, since those live at the call sites that pass it in — an
   `*Engine` parameter (AcademyEngine's shape) is exact.
2. **SQL lives in `//go:embed` files — DONE.** Query text is a package-level
   `var fooSQL string` populated by `//go:embed sql/foo.sql`.
   `analyzer.BuildEmbedIndex` reads each `//go:embed` string var's file once and
   maps the var's object to its contents; `queryText` resolves the `*ssa.Global`
   (or a load of it) at the call site through that index. A file that cannot be
   read is skipped, so its query falls back to `dynamic_sql` rather than a wrong
   guess. `TestEmbeddedSQLResolved` (and its no-index counterpart) cover it.

Neither gap affected the corpus (basic-gin registers routes directly in `main`
and uses inline SQL literals), which is why the tests stayed green throughout.

A third fix was needed once routes were discovered: AcademyEngine's handlers are
methods (`course.GetCourse`), and `analyzer.FunctionsByName` indexed only
package-level functions, so Phase 2/3 could not resolve a handler name to its
SSA function and reachability started from nothing. It now indexes methods too
(via `ssautil.AllFunctions`).

**Result on AcademyEngine** (`scan --project <academy> --allow-no-db`): 195
routes discovered, 48 tables touched across endpoints, 39 endpoints with
unresolved (dynamic) queries honestly flagged, 1401 facts — the endpoint →
middleware → tables map, working on the real target.

The **role → endpoint → tables** map is complete. The adapter records each
middleware's constant string arguments (`gin.Middleware{Name, Args}`), so a
`RequireRole("admin")` gate carries `args: ["admin"]`; the middleware fact
stores them. On AcademyEngine this yields every role (`admin`, `teacher`,
`student`, `owner`, `creator`) joined to its endpoints and, through the data
access facts, to the tables each endpoint reaches — e.g. `[student] GET
/attachment/s/course/{courseId}` reaches `attachments, student_batches,
student_course_access, students, users`. `args` captures the string literals of
any middleware, not only RequireRole, so it generalizes to other gates.

This map is now viewable two ways, both projections over stored facts:
- `internal/report` → a self-contained `report.html` (role filter + search,
  light/dark, no server), written next to the store on every `scan` and
  re-renderable with `pathwise report`.
- `internal/server` → `pathwise serve`, an **interactive React dashboard** on
  localhost (the Phase 6 locked stack: React + TypeScript + Cytoscape.js, built
  by Vite under `frontend/` and embedded via `embed.FS`, so it still ships in
  the one binary). `/api/scan` serves the full projection; the SPA renders it.
  It is a security investigation workbench: an app shell (sidebar + top bar), an
  Overview, an Endpoint explorer (filter by method/role, search), an Access
  Graph (Cytoscape — endpoints ↔ tables, click to trace, role filter), a
  Database explorer, a Findings view, and a contextual inspector showing each
  endpoint's authorization path, data access, roles, checks, findings and
  evidence. Runtime Tests / Scan History / Git Comparison show honest
  "not available yet" states (Phases 5/7). Role colors come from the dataviz
  skill's validated categorical palette.

Build pipeline: `frontend/` is a Vite app whose `npm run build` (via `make ui`)
writes into `internal/server/assets/`, which `go build` embeds. The built assets
are committed, so `make build` needs no Node; run `make ui` after changing the
frontend. CLAUDE §9's "filtered subgraph on node selection" is still deferred —
the full graph is sent once, fine at this size.

---

## 3. Locked decisions

These were decided deliberately. Changing one is a real decision with real
cost, not a refactor.

| Decision | Choice | Why |
|---|---|---|
| **Query layer** | Raw SQL through `pgx` / `database/sql` | GORM's chained builder is close to a separate product. Raw SQL yields recoverable query text at a known call site. `sqlc` is the natural second. |
| **Framework** | **Gin only** for v1 | Supporting five router families means implementing and testing five route-discovery adapters, not recognizing five import paths. Prove the pipeline end to end on one first. |
| **Finding taxonomy** | **Fixed at four** | Each kind dictates what the analyzer must prove. Building analysis first and hoping findings emerge turns this into a research project. |
| **Schema truth** | Live PostgreSQL authoritative; migrations for provenance | Replaying `ALTER TABLE` sequences correctly is a lot of work for a worse answer than `information_schema` already has. Migrations answer "which file introduced this column". |
| **Call graph** | **CHA refined by VTA** | `go/pointer` is deprecated and was removed from the callgraph tool; pointer analysis is no longer supported (golang/go#59676). `-algo` offers only static, cha, rta, vta. Copy Go's own vulncheck: build an initial CHA graph, refine through VTA. |
| **Storage** | SQLite via `modernc.org/sqlite` | Pure Go, no cgo. Valuable precisely because `pg_query_go` forces cgo later; storage should not add more. |
| **SQL parsing** | `pg_query_go/v6` | Uses real PostgreSQL server source for a true parse tree. Costs cgo. Add it in the phase that parses SQL, not before. |

### Scope of v1, deliberately narrow

Gin, raw SQL, live PostgreSQL, four findings. Depth provable before breadth
attempted.

---

## 4. The four findings

The taxonomy is fixed. Each entry states the proof obligation the analyzer
must discharge.

| Kind | Config? | What must be proven |
|---|---|---|
| `missing_ownership_check` | no | A request-controlled value reaches a primary-key or unique-key predicate; **no** identity-derived predicate constrains the same query; **no** ownership check dominates every path to the response write. This is the classic IDOR. |
| `role_check_unenforced` | no | A call classified as a role/permission check has its result **discarded**, **or** its failure branch does not **dominate** the response write. |
| `missing_tenant_filter` | **yes** | A query touches a table declared tenant-scoped and carries no predicate on that table's declared tenant column. |
| `response_field_exceeds_scope` | **yes** | A response serializes a field backed by a column declared sensitive, on an endpoint whose declared scope does not include it. |

### Why two of them need configuration

Nothing in a schema says `org_id` discriminates tenants rather than being an
ordinary foreign key. Nothing says `national_id` is more sensitive than
`name`. Those are project policy, not derivable properties.

**Consequence, already implemented:** running without `.pathwise.yaml` leaves
two of four checks inactive, and `scan` says so explicitly. Without config,
those checks emit an unverified fact naming `missing_config` — **never** a
silent pass.

The two zero-config checks are the default experience. Do not make the whole
tool silent-by-default behind config nobody has written yet.

---

## 5. Architecture

### Pipeline

The scan runs as a pipeline; each stage produces structured data the next
consumes.

```
01 Load project      modules, packages, migrations, router setup, git revision
02 Build schema      live PostgreSQL (authoritative) | migrations (degraded)
03 Discover routes   framework adapter → method, pattern, handler, middleware
04 Build call graph  CHA → VTA; handlers into services, auth helpers, DB access
05 Trace access      identity checks, role conditions, ownership/tenant filters,
                     SQL predicates, response construction
06 Build model       normalize into linked nodes/edges with source locations
                     and an explicit analysis status
07 Plan tests        endpoint × identity scenarios from the model
08 Verify & compare  execute permitted tests, compare with static expectations
```

Stage 05 produces a **logical trace, not a linear path**. Real applications
have branches, early returns, shared middleware, helpers and alternative
queries. The internal representation must preserve branches.

### Module layout

```
cmd/pathwise/            entry point
internal/
  cli/                   cobra command tree (root, scan, stubs)
  config/                .pathwise.yaml loading and policy
  model/                 evidence, facts, graph, findings — the shared model
  project/               module root + git revision resolution
  loader/                go/packages + SSA load, shared by all analyzers
  adapters/gin/          Gin route discovery (SSA) — phase 1
  schema/                live pgx schema reader — phase 1
  analyzer/              CHA→VTA call graph, endpoint → DB access — phase 2
  authorization/         check-enforcement proof via SSA dominance — phase 3
  dataflow/              pg_query_go SQL parse: tables, WHERE predicates — phase 3
  findings/              the finding rules (tenant, scope, role, ownership) — phase 3
  output/                json + SARIF 2.1.0 findings renderers — phase 4
  report/                self-contained HTML role → endpoint → resource map
  runtime/               scenario planner, safety classifier, HTTP executor — phase 5
  server/                dashboard: /api/scan + embedded React SPA (net/http)
  storage/sqlite/        immutable scans, facts, projections, migrations
  corpus/                known-answer corpus loader and validator
frontend/                React + TS + Vite + Cytoscape dashboard (builds into server/assets)
testdata/apps/           corpus applications (own modules, ignored by ./...)
configs/                 pathwise.example.yaml
```

Planned additions, by phase:

```
internal/schema/         live reader + migration replay — both landed (phase 1)
internal/analyzer/       callgraph + dominance landed               (phase 2-3)
internal/dataflow/       pg_query_go parse, predicates, provenance  (phase 3)
internal/authorization/  enforcement proof landed; richer classify  (phase 3)
internal/findings/       rule engine over facts                    (phase 4)
internal/graph/          projection builder + queries              (phase 4)
internal/runtime/        planner, safety, executor — landed         (phase 5)
internal/server/         net/http dashboard API — landed            (phase 6)
internal/gitdiff/        scan comparison                           (phase 7)
```

### The dependency rule

**`model` depends on nothing internal. Adapters depend on `model`; `model`
never depends on an adapter.** Otherwise adding a sixth framework ripples
through the codebase. `graph` depends on the shared model, never on an
individual adapter.

---

## 6. Data model

Defined in `internal/model/`. This is the most important component — every
analyzer emits into one common format rather than inventing its own.

### Facts are truth; the graph is a projection

`facts` is the source of truth. `graph_nodes` / `graph_edges` are a projection
over it and may be rebuilt at any time without re-parsing source.

**Why:** the inference rules that turn facts into edges and findings will
change constantly through Phase 3. Re-deriving a graph from stored facts is
cheap; re-parsing every analyzed project is not.

### Evidence

```go
type EvidenceStatus string
const (
    StatusCodeConfirmed   = "code_confirmed"    // proven from source + types + control flow
    StatusRuntimeObserved = "runtime_observed"  // a live request demonstrated it
    StatusInferred        = "inferred"          // a heuristic fired; may be wrong
    StatusUnverified      = "unverified"        // analysis could not conclude
)
```

These are **not a confidence ladder.** `code_confirmed` and `runtime_observed`
are different *kinds* of evidence, not different strengths of one kind, and
they are **never merged**. A static conclusion and a runtime observation about
the same endpoint are stored separately so a contradiction stays visible.
That contradiction is the single highest-value signal PathWise can emit.

### `unverified` always carries a reason

```go
type UnresolvedReason string
// interface_dispatch, dynamic_sql, dynamic_route, reflection,
// external_package, build_error, missing_config,
// unsupported_query_layer, no_response_schema
```

"Unverified because the call target is an interface with nine
implementations" is actionable. "Unverified" alone is **indistinguishable from
"safe"** to a reader — the most dangerous failure mode this tool has.

**This is enforced, not merely documented.** `Evidence.Valid()` rejects:

- `unverified` with no `UnresolvedReason`
- `runtime_observed` with no `ScenarioID`
- `code_confirmed` with no `SourceLocation`
- `inferred` with no stated heuristic in `Details`

and `Store.InsertFacts` **errors rather than storing** an invalid one. Keep it
that way. It makes the honesty property structural instead of a convention
someone has to remember under deadline in Phase 3.

### Facts

```go
type Fact struct {
    ID       int64
    ScanID   string
    Kind     FactKind          // route, middleware, call_edge, query,
                               // auth_check_candidate, auth_check_enforced, ...
    Subject  string            // STABLE identity, never embeds a line number
    Attrs    map[string]any
    Evidence Evidence
}
```

Note the two distinct kinds `auth_check_candidate` and `auth_check_enforced`.
**The distinction between them is the whole project**: a candidate is a call
that looks like an authorization check; an enforcement fact is a *proof* that
its result gates the response.

`Fact.Fingerprint()` is a content hash with sorted attr keys, used to
deduplicate within a scan. It deliberately ignores source position, so moving
code does not create a duplicate fact.

### Parameter provenance

```go
type ParamProvenance string
// request, identity, constant, derived, unknown
```

**This is the pivot of IDOR detection.** A `request`-derived value reaching a
primary-key predicate with no `identity`-derived predicate alongside it is the
signature of an ownership failure. `unknown` means value tracking failed: it
is neither safe nor unsafe, and must produce an unverified fact, never a
finding.

### Stable identities

Graph IDs and source positions move between revisions even when behavior does
not, so comparison joins on stable identities: normalized `METHOD /path`,
qualified Go function names, database object names.

`model.NormalizeRoute` rewrites framework parameter syntax to one form, so
Gin's `/students/:id` and Chi's `/students/{id}` share one identity. There is
a test asserting this property.

### Findings

`Finding` carries `Severity` **and** `Confidence` separately
(`proven` / `likely` / `possible`), because a high-impact guess and a
high-impact proof demand different responses from a developer. It also carries
`Premises []string` — the facts it was derived from — so a reader can audit
the chain rather than trusting the conclusion. And `RuntimeDivergence bool`,
set when runtime evidence contradicts the static conclusion.

`FindingKind.RequiresConfig()` reports whether a kind can fire with no
`.pathwise.yaml`. The CLI uses it to state which checks are inactive.

---

## 7. Storage

SQLite, separate from the analyzed application's PostgreSQL. Default path
`<project>/.pathwise/pathwise.db`. PathWise's own schema has its own
migrations, embedded via `embed.FS` and applied in filename order, one
transaction each, recorded in `schema_migrations`.

Tables: `projects`, `scans`, `facts`, `fact_locations`, `graph_nodes`,
`graph_edges`, `findings`, `test_scenarios`, `test_runs`, `sensitive_fields`,
`git_changes`.

### Invariants

- **Scans are immutable.** A new scan never overwrites an older one's facts.
  Historical evidence is what makes revision comparison and finding-lifecycle
  tracking possible. There is a test for this.
- **Scan IDs are lexicographically sortable** (`scan_<millis>_<random>`), so
  "the latest scan" is an `ORDER BY`, not a join against timestamps.
- **Projects are keyed by root path**, so repeated scans of one checkout
  accumulate history rather than forking into separate projects.
- `scans.schema_source` records `live | migrations_only | none`. A finding
  resting on an inferred schema is weaker than one resting on the live
  database, and the reader must be able to tell which they are looking at.
- `scans.disabled_checks` records which finding kinds could not run. **A clean
  scan with checks disabled is a partial scan and must never read as an
  all-clear.**
- `db.SetMaxOpenConns(1)` — SQLite tolerates one writer. Analysis is
  concurrent; writes are not.

---

## 8. The corpus — ground truth

`testdata/apps/` holds small applications whose authorization structure is
known in advance, each paired with an `expected.yaml`.

**Why this exists before the analyzers it judges:** without known-answer
applications there is no way to distinguish *found nothing* from *found
nothing because the analyzer silently gave up on an interface dispatch*. That
ambiguity would quietly invalidate every later milestone.

### Markers, not line numbers

Expectations anchor to marker comments in the corpus source, so editing an app
cannot silently invalidate its expectations:

```go
// pathwise:expect kind=missing_ownership_check id=idor-get-student
// pathwise:safe id=ownership-enforced
// pathwise:unresolved reason=dynamic_sql id=dynamic-sql-search
```

`App.Validate()` cross-checks `expected.yaml` against the markers and catches
both ways a corpus decays: an expectation whose marker was deleted, and a
marker nobody declared. Markers are parsed by hand rather than through
`go/ast`, so the corpus validates even while an app does not compile — which
is exactly the state a corpus app is in while being written.

### `basic-gin`

Gin + `database/sql`. 8 routes, 5 tables, 5 foreign keys. `org_id` is the
tenant discriminator.

Seeded flaws — one per finding kind, **two** for `role_check_unenforced`:

| id | kind | the point |
|---|---|---|
| `idor-get-student` | `missing_ownership_check` | `c.Param("id")` → PK predicate, no ownership check, no identity predicate |
| `role-discarded-admin-report` | `role_check_unenforced` | `_ = IsAdmin(user)` — result discarded. Sits behind a **real** `RequireRole` middleware on purpose: a correct analyzer must attribute the finding to the handler's dead check, not call the endpoint unprotected |
| `role-nondominating-dashboard` | `role_check_unenforced` | `HasRole` is branched on, but only inside `if c.Query("strict") == "1"`. The abort does not dominate the response write. **Requires dominance analysis**; looking for discarded return values passes the first case and silently fails this one |
| `tenant-list-batches` | `missing_tenant_filter` | `SELECT ... FROM batches` with no `org_id` predicate |
| `scope-profile-national-id` | `response_field_exceeds_scope` | Ownership enforced correctly, but serializes `national_id` outside declared scope. Must be reported as an over-broad response, not an authorization failure |

Negative controls — a finding on any of these is a false positive:

| id | the point |
|---|---|
| `ownership-enforced` | `CanAccessStudent` branched on, abort dominates the response write |
| `login-public-by-design` | Unauthenticated by design. An analyzer that flags every handler lacking an auth check fails here |
| `dynamic-sql-search` | `fmt.Sprintf`-built query. Correct output is **unverified / `dynamic_sql`**. A finding is a false positive; silence is a coverage gap |

`TestCorpusHasNegativeControls` fails if an app declares no `expected_safe`
sites, because **an analyzer measured only against seeded flaws scores 100% by
flagging everything.** `TestCorpusCoversEveryFindingKind` fails if a finding
kind has no corpus instance.

---

## 9. Phases and exit criteria

### Phase 1 — Schema and routes (2–3 weeks) ← next

- Migration loader (golang-migrate), applied **in order** into an internal
  model: `ALTER TABLE`, renamed columns, dropped constraints. Not a collection
  of `CREATE TABLE`s.
- Live schema via `pgx`: tables, columns, types, nullability, PKs, unique
  constraints, **explicit foreign keys only** — do not invent relationships
  where no constraint exists.
- Gin adapter: SSA constant propagation on router registration calls. Routes
  get registered in loops, behind helper wrappers, from config — recognizing
  the import path is not route discovery. Handle route groups, nested groups
  and middleware ordering.

**Exit:** ≥95% of corpus routes and all tables/FKs recovered, with misses
listed **explicitly** rather than silently dropped.

### Phase 2 — Call graph to data access (3–4 weeks)

SSA via `ssautil`, CHA→VTA call graph, handler reachability, identify DB call
sites, map endpoint → tables touched with operation type.

**Exit:** correct table set per corpus endpoint; **every** unresolved call site
named with a reason. First genuinely useful output — "GET /students/{id} reads
students, batches" is worth something on its own.

### Phase 3 — Authorization semantics (4–6 weeks, the hard core)

- Classify candidate checks (name patterns + `config.auth_check_funcs`).
- **Prove enforcement via SSA dominance:** does the failure branch dominate
  all paths to the response write?
- **Classify WHERE predicates by parameter provenance:** request-derived vs
  identity-derived, tracked interprocedurally.
- The IDOR signature falls out of this: request-controlled value reaches a
  primary-key predicate, no identity-derived predicate present, no ownership
  check dominating.

**Exit:** every seeded corpus vuln detected, with a **measured** false-positive
rate you can state.

### Phase 4 — Findings and CLI output (1–2 weeks)

Rule engine over stored facts. `--format json|text`. **Consider SARIF** — it
is the CI standard and gets GitHub code-scanning integration nearly free.

**Exit:** actionable findings with `file:line`. No dashboard needed.

### Phase 5 — Runtime verification (3–4 weeks)

- **Developer-supplied credentials only.** Drop auto-registration entirely for
  v1 — a generic CLI cannot guarantee it can create accounts in an arbitrary
  app, and it is a tarpit.
- Safety classifier **defaults to deny for every non-GET method**, with
  per-endpoint opt-in. An endpoint that deletes records is not safe to execute
  merely because the app is running locally. External side effects (email,
  payments) detected statically are imperfect — exclude conservatively.
- Test data may persist in the developer's database; keep a visible test-data
  inventory.
- Outcomes: `pass` / `fail` / `blocked` / `unverified` — and a fifth,
  **`divergent`**:
  - static says protected, runtime says reachable → **highest-confidence
    finding in the whole tool**
  - static says unverified, runtime says blocked → **a coverage gap, not a
    vulnerability**

  Conflating those two will erode trust fast.
- Runtime evidence attaches to the **specific scenario** that observed it. One
  successful request does not establish that all users with that role have
  equivalent access. These statuses must never imply more coverage than was
  tested.

### Phase 6 — Dashboard (2–3 weeks)

React + TypeScript + Cytoscape.js, served from `net/http`, build embedded via
`embed.FS` so it stays a single binary. Two exploration levels: global graph,
and per-endpoint trace including alternate branches and unresolved checks.
Backend sends a **filtered subgraph** on node selection — never the full graph.

### Phase 7 — Breadth

More adapters (Chi, Echo, Fiber, net/http), GORM, git diff between scans, CI
mode.

### Timeline reality

Phases 0–4 give a real, defensible tool: static analysis finding authorization
flaws in one framework and one query style, with code citations. Roughly 11–17
weeks part-time. The full architecture with five adapters is closer to a year.
Ship the narrow version first and let real usage decide which framework comes
second.

---

## 10. Conventions

- **Comments explain *why*, not *what*.** The non-obvious decisions and
  failure modes are the valuable content. Existing comments follow this; match
  them.
- Errors wrap with context: `fmt.Errorf("read %s: %w", path, err)`.
- Stub commands return an explicit not-implemented error naming their phase.
  Never a stub that appears to succeed.
- Keep secrets out of config and out of scan history. The DB URL comes from
  `--db` or `$DATABASE_URL`; bearer tokens are never persisted.
- A missing `.pathwise.yaml` is not an error — it yields a zero `Config` whose
  `DisabledFindings()` reports what that costs.
- Tests assert *properties*, not snapshots: route identity is
  framework-agnostic, scans are immutable, invalid evidence is rejected,
  fingerprints ignore attr ordering.
- Run `make check` (vet + test) before committing.

---

## 11. Landmines

- **`go/packages` requires the target to compile.** Missing dependencies,
  build tags, uncommitted generated files — any of these kills analysis before
  it starts, and it is **the most common way tools like this fail on real
  codebases**. Surface it as `build_error` with a clear diagnostic path in
  Phase 1. Never report it as "no findings".
- **VTA soundness.** Sound only modulo reflection and `unsafe`, given a sound
  initial graph. That caveat is exactly what feeds `unverified` — record
  `reflection` rather than silently ignoring it.
- **`fmt.Sprintf`-built queries** need handling as an explicit `dynamic_sql`
  unresolved case, not a silent skip.
- **`pg_query_go` needs cgo.** `CGO_ENABLED=1`, slow first builds, awkward
  cross-compilation. This conflicts with shipping via `go install` — decide
  early whether to distribute prebuilt binaries.
- **Live DB as schema truth** means `scan` needs a running PostgreSQL, which
  constrains CI and first-run experience. The degraded `--allow-no-db` mode
  marks schema facts `inferred`.
- **Static analysis is not omniscient.** Reflection, interface dispatch,
  dynamically registered routes, complex control flow. Preserve unresolved
  paths; never let them read as resolved.
- **A passing test is not proof of security.** It may not exercise the
  vulnerable branch, or may rest on an incorrect expectation.

---

## 12. Commands

```sh
# First run in a fresh checkout
go mod tidy
make corpus-tidy        # resolves the corpus app's own module

make build              # -> ./pathwise (embeds the prebuilt dashboard; no Node needed)
make ui                 # rebuild the React dashboard into internal/server/assets (needs Node)
make check              # vet + test, run before committing
make test
make fmt

# Live database is authoritative for schema (IDOR needs it for primary keys)
./pathwise scan --project . --db "$DATABASE_URL"
./pathwise scan --project . --db "$DATABASE_URL" --format sarif   # CI output

# Degraded: schema from migrations, facts marked inferred
./pathwise scan --project . --allow-no-db

# Every scan also writes <store-dir>/report.html — a single self-contained page
# (no server) mapping each endpoint to the roles that gate it and the tables it
# reaches. Re-render the latest scan without re-analyzing:
./pathwise report --project . [--out report.html]

./pathwise serve        # interactive dashboard on localhost (endpoints ↔ tables)
./pathwise diff         # phase 7 — returns not-implemented

# Runtime verification: exercise a RUNNING app as supplied identities and
# compare what it did with the static prediction. GET-only by default (non-GET
# is blocked by safety unless opted in with --allow); parameterized paths are
# left unverified. Tokens are never stored. Results persist against the scan
# (unless --no-store) and show in the dashboard's Runtime Tests page.
./pathwise test --project . --base-url http://localhost:8080 \
    --identity student=<token> --identity admin=<token> \
    --allow "POST /batches" [--format json] [--no-store]
```

### Dependencies by phase

Current: `cobra`, `gopkg.in/yaml.v3`, `modernc.org/sqlite`,
`golang.org/x/tools`. The last was added in Phase 1 for the SSA-based Gin
adapter (it was originally penciled in for Phase 2, but route discovery needs
SSA too); pulling `x/tools@latest` for Go 1.26 compatibility also bumped the
`go` directive to 1.26. `pgx/v5` was added in Phase 1 for the live schema
reader. `pg_query_go/v6` (cgo, the real PostgreSQL parser) was added in Phase 3
for WHERE-predicate analysis; it drives `internal/dataflow` and replaced the
Phase 2 keyword table reader. cgo means `CGO_ENABLED=1` and a C toolchain are
required to build, and the first build is slow. `pg_query_go` also drives the
Phase 1 migration-reconstruction path (`internal/schema.Reconstruct`).

## graphify

This project has a knowledge graph at graphify-out/ with god nodes, community structure, and cross-file relationships.

Rules:
- When graphify-out/graph.json exists, run `graphify query "<question>"` before your first grep, search or source-file read for a codebase question, and put this rule in every subagent prompt that explores code. Read raw files only for lines the graph or an earlier read already located. Use `graphify path "<A>" "<B>"` for relationships and `graphify explain "<concept>"` for focused concepts. These return a scoped subgraph, usually much smaller than GRAPH_REPORT.md or raw grep output.
- If graphify-out/wiki/index.md exists, use it for broad navigation instead of raw source browsing.
- Read graphify-out/GRAPH_REPORT.md only for broad architecture review or when query/path/explain do not surface enough context.
- After modifying code, run `graphify update .` to keep the graph current (AST-only, no API cost).
- A committed graphify-out/graph.json is rewritten whole on every build, so a large diff to it in a pull request is expected. Do not flag it as churn or suggest skipping the graph rebuild or its hooks.
