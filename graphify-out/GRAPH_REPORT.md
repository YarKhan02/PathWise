# Graph Report - PathWise  (2026-10-15)

## Corpus Check
- 99 files · ~72,326 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 43 file(s) not represented in the graph (top: .woff2 22, .woff 15, (none) 3)

## Summary
- 1288 nodes · 3611 edges · 90 communities (62 shown, 28 thin omitted)
- Extraction: 93% EXTRACTED · 7% INFERRED · 0% AMBIGUOUS · INFERRED: 267 edges (avg confidence: 0.85)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- golang.org/x/tools/go/ssa.Value
- App.tsx
- testing.T
- isNamedPtr
- cli/scan.go
- basic-gin/store.go
- internal/model
- package.json
- BuildScan
- pgx.go
- compilerOptions
- scans.go
- runtime.go
- analyzeCode
- index-Cafd8W2Y.js
- migrations.go
- context.Context
- runTest
- PathWise
- project.go
- Known-Answer Corpus
- .pathwise.yaml Config and Policy
- e
- Dashboard (React + Cytoscape)
- Phase 7 — Breadth
- pathwise.example.yaml policy template
- Phase 0 — Skeleton and Data Model
- Phase 2 — Call Graph to Data Access
- Phase 4 — Findings and CLI Output
- Phase 6 — Dashboard
- internal/cli
- internal/config
- internal/loader
- internal/project
- brokentest
- embedsql
- ginmodule
- ginspread
- github.com/YarKhan02/pathwise
- pathwise.test/basic-gin
- Nu
- test.go
- re
- h
- Nr
- golang.org/x/tools/go/ssa.Function
- Inspector.tsx
- Classifier
- enforce.go
- Route
- Finding
- kw
- types.ts
- Config
- AccessGraph.tsx
- RenderHTML
- Je
- dependencies
- devDependencies
- $g
- model/facts.go
- Xt
- corpus.go
- dr
- output.go
- primitives.tsx
- lD
- pgx_test.go
- RuntimeTests.tsx
- markers.go
- Ol
- TestResult
- graph.go
- Ps
- pf
- bm
- ka
- vx
- di
- dy
- ga
- ft
- ir
- Ei
- Eo
- jw
- ly
- My
- ct

## God Nodes (most connected - your core abstractions)
1. `e()` - 50 edges
2. `re()` - 46 edges
3. `r()` - 42 edges
4. `kw()` - 40 edges
5. `n()` - 38 edges
6. `a()` - 30 edges
7. `Finding` - 24 edges
8. `Nu()` - 24 edges
9. `yg()` - 22 edges
10. `analyzeFindings()` - 20 edges

## Surprising Connections (you probably didn't know these)
- `login-public-by-design (safe)` --conceptually_related_to--> `Finding: missing_ownership_check`  [INFERRED]
  testdata/apps/basic-gin/expected.yaml → CLAUDE.md
- `main()` --calls--> `NewRootCommand()`  [EXTRACTED]
  cmd/pathwise/main.go → internal/cli/root.go
- `basic-gin expected.yaml` --implements--> `basic-gin Corpus App`  [INFERRED]
  testdata/apps/basic-gin/expected.yaml → CLAUDE.md
- `basic-gin expected.yaml` --references--> `Known-Answer Corpus`  [INFERRED]
  testdata/apps/basic-gin/expected.yaml → CLAUDE.md
- `basic-gin .pathwise.yaml policy` --conceptually_related_to--> `pathwise.example.yaml policy template`  [INFERRED]
  testdata/apps/basic-gin/.pathwise.yaml → configs/pathwise.example.yaml

## Import Cycles
- None detected.

## Hyperedges (group relationships)
- **Evidence Status Kinds** — claude_status_code_confirmed, claude_status_runtime_observed, claude_status_inferred, claude_status_unverified [EXTRACTED 1.00]
- **The Four Findings Taxonomy** — claude_finding_missing_ownership_check, claude_finding_role_check_unenforced, claude_finding_missing_tenant_filter, claude_finding_response_field_exceeds_scope [EXTRACTED 1.00]
- **IDOR Detection Signature** — claude_param_provenance, claude_ssa_dominance, claude_finding_missing_ownership_check, claude_pkg_dataflow [INFERRED 0.85]

## Communities (90 total, 28 thin omitted)

### Community 0 - "golang.org/x/tools/go/ssa.Value"
Cohesion: 0.14
Nodes (19): discoverer, routerNode, constString(), isGinConstructor(), isGinRouterMethod(), joinPaths(), qualifiedName(), sliceElements() (+11 more)

### Community 1 - "App.tsx"
Cohesion: 0.22
Nodes (13): App(), Selection, Item, primary, shortScan(), Sidebar(), verify, View (+5 more)

### Community 2 - "testing.T"
Cohesion: 0.14
Nodes (24): Facts(), buildErrorFact(), CodeConfirmed(), Evidence, UnresolvedReason, Inferred(), Unresolved(), NormalizeRoute() (+16 more)

### Community 3 - "isNamedPtr"
Cohesion: 0.18
Nodes (10): isGinEnginePtr(), isNamedPtr(), IsRequestSource(), IsResponseWrite(), isRouterGroupPtr(), ResponseBody(), isStringType(), jsonFields() (+2 more)

### Community 5 - "basic-gin/store.go"
Cohesion: 0.08
Nodes (38): Batch, Student, StudentProfile, User, Handler, Module, Auth(), main() (+30 more)

### Community 6 - "internal/model"
Cohesion: 0.05
Nodes (37): auth_check_candidate Fact, auth_check_enforced Fact, CHA Refined by VTA Call Graph, Evidence Model, Fact (Source of Truth), Finding: missing_ownership_check, Finding: role_check_unenforced, Finding (Severity + Confidence) (+29 more)

### Community 7 - "package.json"
Cohesion: 0.10
Nodes (18): name, private, scripts, build, dev, preview, type, version (+10 more)

### Community 8 - "BuildScan"
Cohesion: 0.12
Nodes (9): BuildScan(), nonNil(), nonNilFindings(), sortedSet(), Handler(), Serve(), serveIndex(), TestHandlerServesScanAndPage() (+1 more)

### Community 9 - "pgx.go"
Cohesion: 0.28
Nodes (4): collect(), readKeyColumns(), ReadLive(), uniqueGroup

### Community 10 - "compilerOptions"
Cohesion: 0.11
Nodes (18): compilerOptions, allowImportingTsExtensions, isolatedModules, jsx, lib, module, moduleDetection, moduleResolution (+10 more)

### Community 11 - "scans.go"
Cohesion: 0.12
Nodes (13): codeAnalysis, keyColumns(), readSchema(), Fact, FactKind, nonNilStrings(), boolToInt(), Scan (+5 more)

### Community 12 - "runtime.go"
Cohesion: 0.23
Nodes (16): parseAllow(), parseIdentities(), classify(), expectAllowed(), Endpoint, Identity, Result, Key() (+8 more)

### Community 13 - "analyzeCode"
Cohesion: 0.12
Nodes (25): Discover(), corpusApp(), equalStrings(), TestDiscoverBasicGinRoutes(), TestMiddlewareOrderPreserved(), TestRouterParameterModulePattern(), TestUnresolvableHandlerIsMiss(), BuildCallGraph() (+17 more)

### Community 14 - "index-Cafd8W2Y.js"
Cohesion: 0.02
Nodes (16): ax, c0, fD, fx, ix, lx, mx, ox (+8 more)

### Community 15 - "migrations.go"
Cohesion: 0.12
Nodes (22): EqPredicate, columnName(), Query, operatorIs(), paramNumber(), Parse(), TestDynamicSQLUnparsable(), TestParseCorpusQueries() (+14 more)

### Community 16 - "context.Context"
Cohesion: 0.29
Nodes (6): loadReport(), writeReport(), endpointsForRuntime(), Report, Store, Open()

### Community 17 - "runTest"
Cohesion: 0.29
Nodes (15): globalFlags, testFlags, main(), newReportCommand(), runReport(), NewRootCommand(), notImplemented(), newScanCommand() (+7 more)

### Community 18 - "PathWise"
Cohesion: 0.17
Nodes (8): Runtime divergent Outcome, Graph and Evidence Storage Responsibility, PathWise, Phase 5 — Runtime Verification, internal/runtime, Runtime Verification Responsibility, Scan Pipeline, Static Analysis Responsibility

### Community 19 - "project.go"
Cohesion: 0.31
Nodes (7): describeRevision(), findModuleRoot(), gitState(), Project, readModulePath(), Resolve(), runGit()

### Community 20 - "Known-Answer Corpus"
Cohesion: 0.40
Nodes (5): basic-gin Corpus App, Known-Answer Corpus, Corpus Markers (pathwise: comments), internal/corpus, basic-gin expected.yaml

### Community 21 - ".pathwise.yaml Config and Policy"
Cohesion: 0.33
Nodes (5): .pathwise.yaml Config and Policy, Finding: missing_tenant_filter, Finding: response_field_exceeds_scope, scope-profile-national-id, tenant-list-batches

### Community 22 - "e"
Cohesion: 0.06
Nodes (74): a0(), b(), ba(), bf(), Bi(), bv(), bx(), ck() (+66 more)

### Community 41 - "Nu"
Cohesion: 0.06
Nodes (40): Al(), Ap(), Bs(), Ea(), eg(), ev(), fv(), g1() (+32 more)

### Community 43 - "re"
Cohesion: 0.09
Nodes (32): ac(), ag(), am(), av(), Cg(), Dw(), Fp(), G0() (+24 more)

### Community 44 - "h"
Cohesion: 0.14
Nodes (25): j2(), b(), g(), m(), p(), v(), x(), Ny() (+17 more)

### Community 45 - "Nr"
Cohesion: 0.11
Nodes (29): Ao(), Au(), Aw(), cm(), Dn(), fs(), gn(), gw() (+21 more)

### Community 46 - "golang.org/x/tools/go/ssa.Function"
Cohesion: 0.30
Nodes (11): realFunc(), underlyingCallee(), AnalyzeReachableDB(), DBAccess, reachableFuncs(), EmbedIndex, NewClassifier(), analyzeDataAccess() (+3 more)

### Community 47 - "Inspector.tsx"
Cohesion: 0.31
Nodes (12): EndpointInspector(), Inspector(), loc(), Step(), TableInspector(), EnforcementBadge(), MethodBadge(), RoleTag() (+4 more)

### Community 48 - "Classifier"
Cohesion: 0.33
Nodes (6): Provenance, combine(), Classifier, identityOrBase(), peel(), qualifiedName()

### Community 49 - "enforce.go"
Cohesion: 0.23
Nodes (14): Enforcement, at(), dbQueryArg(), isSQLQuerier(), CheckEnforcement(), classify(), condUsesIdentity(), controllingIf() (+6 more)

### Community 50 - "Route"
Cohesion: 0.22
Nodes (8): Unresolved, Middleware, Miss, Result, Route, rootsFor(), routeNamed(), dataAccessFacts()

### Community 51 - "Finding"
Cohesion: 0.13
Nodes (13): reasonDisabled(), reportCoverage(), roleCheckFinding(), SourceLocation, AllFindingKinds(), Finding, FindingKind, TestRequiresConfig() (+5 more)

### Community 52 - "kw"
Cohesion: 0.12
Nodes (28): cc(), Cw(), $d(), dg(), Dp(), Ff(), ic(), ii() (+20 more)

### Community 53 - "types.ts"
Cohesion: 0.20
Nodes (9): Check, Counts, Finding, Location, Middleware, RuntimeOutcome, RuntimeTest, TableInfo (+1 more)

### Community 54 - "Config"
Cohesion: 0.15
Nodes (13): EndpointScope, Tenancy, ReachedQuery, Config, hasIdentityPredicate(), idorFinding(), isKeyColumn(), OwnershipCheck() (+5 more)

### Community 55 - "AccessGraph.tsx"
Cohesion: 0.36
Nodes (7): Load, roleColorVar(), AccessGraph(), buildElements(), METHOD_COLOR, stylesheet(), react

### Community 57 - "Je"
Cohesion: 0.11
Nodes (27): $c(), Cn(), Cv(), Er(), Fa(), fc(), fo(), e() (+19 more)

### Community 58 - "dependencies"
Cohesion: 0.29
Nodes (7): dependencies, cytoscape, @fontsource/ibm-plex-mono, @fontsource-variable/inter, lucide-react, react, react-dom

### Community 59 - "devDependencies"
Cohesion: 0.29
Nodes (7): devDependencies, @types/cytoscape, @types/react, @types/react-dom, typescript, vite, @vitejs/plugin-react

### Community 60 - "$g"
Cohesion: 0.14
Nodes (26): Af(), As(), bw(), dl(), dv(), Ef(), $f(), $g() (+18 more)

### Community 61 - "model/facts.go"
Cohesion: 0.12
Nodes (18): TestDiscoverRoutesReportsBuildError(), attrInt(), attrStr(), attrStrings(), Build(), contains(), sortedKeys(), ParamProvenance (+10 more)

### Community 62 - "Xt"
Cohesion: 0.14
Nodes (24): d1(), dm(), em(), f1(), fw(), gc(), hg(), _i() (+16 more)

### Community 63 - "corpus.go"
Cohesion: 0.18
Nodes (16): App, ExpectedFinding, ExpectedForeignKey, ExpectedRoute, ExpectedSafe, ExpectedSchema, ExpectedTable, ExpectedUnresolved (+8 more)

### Community 64 - "dr"
Cohesion: 0.15
Nodes (17): bc(), bg(), Dc(), dr(), gg(), hf(), hv(), If() (+9 more)

### Community 65 - "output.go"
Cohesion: 0.25
Nodes (14): Severity, SARIF(), sarifLevel(), sarifArtifact, sarifDriver, sarifLocation, sarifLog, sarifPhysical (+6 more)

### Community 66 - "primitives.tsx"
Cohesion: 0.30
Nodes (11): ConfidenceBadge(), enforcementLabel, EvidenceBadge(), evidenceLabel(), SeverityBadge(), Enforcement, Findings(), rank() (+3 more)

### Community 67 - "lD"
Cohesion: 0.15
Nodes (11): ad(), cD(), ex, Hn(), lD(), nx, uu(), Vd() (+3 more)

### Community 68 - "pgx_test.go"
Cohesion: 0.32
Nodes (9): keysOf(), TestReconstructMatchesCorpus(), applyMigration(), columnNames(), keys(), resetAndExec(), sameSet(), TestReadLiveCompositeForeignKey() (+1 more)

### Community 69 - "RuntimeTests.tsx"
Cohesion: 0.33
Nodes (8): CLS, fmtTime(), OUTCOMES, rank(), Row(), RuntimeTests(), Unavailable(), lucide-react

### Community 70 - "markers.go"
Cohesion: 0.42
Nodes (7): Marker, MarkerKind, TestMarkerParsing(), TestMarkersIgnoreNonComments(), parseMarker(), scanFile(), ScanMarkers()

### Community 71 - "Ol"
Cohesion: 0.25
Nodes (8): bp(), es(), Hd(), kp(), Ol(), Q1(), X1(), Y0()

### Community 72 - "TestResult"
Cohesion: 0.33
Nodes (4): toStoredResults(), TestResult, splitEndpoint(), StoredTestResult

### Community 73 - "graph.go"
Cohesion: 0.48
Nodes (6): Edge, EdgeKind, Graph, Node, NodeKind, Operation

### Community 74 - "Ps"
Cohesion: 0.29
Nodes (7): B0(), df(), hu(), L0(), M0(), Ps(), Zo()

### Community 75 - "pf"
Cohesion: 0.40
Nodes (5): cp(), pf(), q0(), U0(), W0()

### Community 76 - "bm"
Cohesion: 0.50
Nodes (4): bm(), br(), km(), qv()

### Community 77 - "ka"
Cohesion: 0.67
Nodes (3): Ah(), ka(), Oh()

### Community 78 - "vx"
Cohesion: 0.67
Nodes (3): cx, hx(), vx()

### Community 80 - "dy"
Cohesion: 0.67
Nodes (3): dy(), Qk(), vy()

### Community 81 - "ga"
Cohesion: 0.67
Nodes (3): Fn(), ga(), ti()

### Community 82 - "ft"
Cohesion: 0.67
Nodes (3): ft(), wm(), Yw

### Community 83 - "ir"
Cohesion: 0.67
Nodes (3): ir(), e(), is()

## Knowledge Gaps
- **41 isolated node(s):** `@types/cytoscape`, `@types/react`, `@types/react-dom`, `typescript`, `github.com/YarKhan02/pathwise` (+36 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 267 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **28 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Finding` connect `Finding` to `output.go`, `testing.T`, `cli/scan.go`, `BuildScan`, `scans.go`, `golang.org/x/tools/go/ssa.Function`, `runTest`, `Config`, `model/facts.go`?**
  _High betweenness centrality (0.006) - this node is a cross-community bridge._
- **Are the 24 inferred relationships involving `e()` (e.g. with `ac()` and `bv()`) actually correct?**
  _`e()` has 24 INFERRED edges - model-reasoned connections that need verification._
- **What connects `@types/cytoscape`, `@types/react`, `@types/react-dom` to the rest of the system?**
  _41 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `golang.org/x/tools/go/ssa.Value` be split into smaller, more focused modules?**
  _Cohesion score 0.1431451612903226 - nodes in this community are weakly interconnected._
- **Why does `Loaded` connect `analyzeCode` to `cli/scan.go`, `golang.org/x/tools/go/ssa.Function`?**
  _High betweenness centrality (0.006) - this node is a cross-community bridge._
- **Are the 21 inferred relationships involving `r()` (e.g. with `bv()` and `bw()`) actually correct?**
  _`r()` has 21 INFERRED edges - model-reasoned connections that need verification._
- **Should `testing.T` be split into smaller, more focused modules?**
  _Cohesion score 0.13548387096774195 - nodes in this community are weakly interconnected._