# Graph Report - PathWise  (2026-10-15)

## Corpus Check
- 89 files · ~63,403 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 43 file(s) not represented in the graph (top: .woff2 22, .woff 15, (none) 3)

## Summary
- 683 nodes · 1784 edges · 41 communities (21 shown, 20 thin omitted)
- Extraction: 96% EXTRACTED · 4% INFERRED · 0% AMBIGUOUS · INFERRED: 63 edges (avg confidence: 0.85)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- Gin Route Adapter
- Dashboard React UI
- SSA Loader & Packages
- Config Policy & Scan Analysis
- Schema Extraction (AST/types)
- Corpus Gin App
- Evidence & Fact Model
- Frontend Dependencies
- Report Projection
- Live Schema Reader (pgx)
- TypeScript Config
- SQLite Scan Storage
- Dashboard HTTP Server
- Corpus Loader & Validation
- CLI Entry & Commands
- SQL Parsing (pg_query_go)
- Report & Serve Wiring
- Cobra Command Tree
- Project Principles & Phases
- Project & Git Resolution
- Known-Answer Corpus
- Config-Gated Findings
- Embed-SQL Fixture
- Dashboard (React + Cytoscape
- Phase 7 — Breadth
- pathwise.example.yaml policy
- Phase 0 — Skeleton and Data 
- Phase 2 — Call Graph to Data
- Phase 4 — Findings and CLI O
- Phase 6 — Dashboard
- internal/cli
- internal/config
- internal/loader
- internal/project
- brokentest
- embedsql
- ginmodule
- ginspread
- github.com/YarKhan02/pathwis
- pathwise.test/basic-gin

## God Nodes (most connected - your core abstractions)
1. `Finding` - 18 edges
2. `compilerOptions` - 17 edges
3. `AnalyzeReachableDB()` - 16 edges
4. `Config` - 16 edges
5. `discoverer` - 15 edges
6. `runScan()` - 15 edges
7. `analyzeCode()` - 15 edges
8. `Fact` - 14 edges
9. `Discover()` - 13 edges
10. `analyzeFindings()` - 13 edges

## Surprising Connections (you probably didn't know these)
- `main()` --calls--> `NewRootCommand()`  [EXTRACTED]
  cmd/pathwise/main.go → internal/cli/root.go
- `login-public-by-design (safe)` --conceptually_related_to--> `Finding: missing_ownership_check`  [INFERRED]
  testdata/apps/basic-gin/expected.yaml → CLAUDE.md
- `basic-gin .pathwise.yaml policy` --conceptually_related_to--> `pathwise.example.yaml policy template`  [INFERRED]
  testdata/apps/basic-gin/.pathwise.yaml → configs/pathwise.example.yaml
- `idor-get-student` --implements--> `Finding: missing_ownership_check`  [EXTRACTED]
  testdata/apps/basic-gin/expected.yaml → CLAUDE.md
- `ownership-enforced (safe)` --references--> `SSA Dominance Analysis`  [EXTRACTED]
  testdata/apps/basic-gin/expected.yaml → CLAUDE.md

## Import Cycles
- None detected.

## Hyperedges (group relationships)
- **The Four Findings Taxonomy** — claude_finding_missing_ownership_check, claude_finding_role_check_unenforced, claude_finding_missing_tenant_filter, claude_finding_response_field_exceeds_scope [EXTRACTED 1.00]
- **Evidence Status Kinds** — claude_status_code_confirmed, claude_status_runtime_observed, claude_status_inferred, claude_status_unverified [EXTRACTED 1.00]
- **IDOR Detection Signature** — claude_param_provenance, claude_ssa_dominance, claude_finding_missing_ownership_check, claude_pkg_dataflow [INFERRED 0.85]

## Communities (41 total, 20 thin omitted)

### Community 0 - "Gin Route Adapter"
Cohesion: 0.07
Nodes (47): Enforcement, discoverer, routerNode, constString(), isGinConstructor(), isGinEnginePtr(), isGinRouterMethod(), isNamedPtr() (+39 more)

### Community 1 - "Dashboard React UI"
Cohesion: 0.09
Nodes (52): App(), EndpointInspector(), Inspector(), loc(), Selection, Step(), TableInspector(), ConfidenceBadge() (+44 more)

### Community 2 - "SSA Loader & Packages"
Cohesion: 0.07
Nodes (48): Discover(), corpusApp(), equalStrings(), TestDiscoverBasicGinRoutes(), TestMiddlewareOrderPreserved(), TestRouterParameterModulePattern(), TestUnresolvableHandlerIsMiss(), BuildCallGraph() (+40 more)

### Community 3 - "Config Policy & Scan Analysis"
Cohesion: 0.06
Nodes (41): Unresolved, codeAnalysis, EndpointScope, Tenancy, ExpectedUnresolved, Middleware, Miss, Result (+33 more)

### Community 4 - "Schema Extraction (AST/types)"
Cohesion: 0.16
Nodes (10): ExpectedForeignKey, ExpectedSchema, ExpectedTable, Edge, EdgeKind, Graph, Node, NodeKind (+2 more)

### Community 5 - "Corpus Gin App"
Cohesion: 0.08
Nodes (38): Batch, Student, StudentProfile, User, Handler, Module, Auth(), main() (+30 more)

### Community 6 - "Evidence & Fact Model"
Cohesion: 0.05
Nodes (37): auth_check_candidate Fact, auth_check_enforced Fact, CHA Refined by VTA Call Graph, Evidence Model, Fact (Source of Truth), Finding: missing_ownership_check, Finding: role_check_unenforced, Finding (Severity + Confidence) (+29 more)

### Community 7 - "Frontend Dependencies"
Cohesion: 0.06
Nodes (32): dependencies, cytoscape, @fontsource/ibm-plex-mono, @fontsource-variable/inter, lucide-react, react, react-dom, devDependencies (+24 more)

### Community 8 - "Report Projection"
Cohesion: 0.10
Nodes (21): attrInt(), attrStr(), attrStrings(), Build(), contains(), sortedKeys(), BuildScan(), nonNil() (+13 more)

### Community 9 - "Live Schema Reader (pgx)"
Cohesion: 0.14
Nodes (15): collect(), readKeyColumns(), ReadLive(), applyMigration(), columnNames(), keys(), resetAndExec(), sameSet() (+7 more)

### Community 10 - "TypeScript Config"
Cohesion: 0.11
Nodes (18): compilerOptions, allowImportingTsExtensions, isolatedModules, jsx, lib, module, moduleDetection, moduleResolution (+10 more)

### Community 11 - "SQLite Scan Storage"
Cohesion: 0.16
Nodes (10): readSchema(), nonNilLocations(), nonNilStrings(), boolToInt(), Scan, SchemaSource, newID(), nowRFC3339() (+2 more)

### Community 12 - "Dashboard HTTP Server"
Cohesion: 0.16
Nodes (4): Handler(), Serve(), serveIndex(), TestHandlerServesScanAndPage()

### Community 13 - "Corpus Loader & Validation"
Cohesion: 0.18
Nodes (12): App, ExpectedFinding, ExpectedRoute, ExpectedSafe, Marker, MarkerKind, contains(), TestMarkerParsing() (+4 more)

### Community 15 - "SQL Parsing (pg_query_go)"
Cohesion: 0.26
Nodes (8): EqPredicate, columnName(), Query, operatorIs(), paramNumber(), Parse(), TestDynamicSQLUnparsable(), TestParseCorpusQueries()

### Community 16 - "Report & Serve Wiring"
Cohesion: 0.27
Nodes (6): loadReport(), writeReport(), RenderHTML(), Report, Store, Open()

### Community 17 - "Cobra Command Tree"
Cohesion: 0.37
Nodes (11): globalFlags, newReportCommand(), runReport(), NewRootCommand(), notImplemented(), newScanCommand(), runScan(), newServeCommand() (+3 more)

### Community 18 - "Project Principles & Phases"
Cohesion: 0.17
Nodes (8): Runtime divergent Outcome, Graph and Evidence Storage Responsibility, PathWise, Phase 5 — Runtime Verification, internal/runtime, Runtime Verification Responsibility, Scan Pipeline, Static Analysis Responsibility

### Community 19 - "Project & Git Resolution"
Cohesion: 0.31
Nodes (7): describeRevision(), findModuleRoot(), gitState(), Project, readModulePath(), Resolve(), runGit()

### Community 20 - "Known-Answer Corpus"
Cohesion: 0.40
Nodes (5): basic-gin Corpus App, Known-Answer Corpus, Corpus Markers (pathwise: comments), internal/corpus, basic-gin expected.yaml

### Community 21 - "Config-Gated Findings"
Cohesion: 0.33
Nodes (5): .pathwise.yaml Config and Policy, Finding: missing_tenant_filter, Finding: response_field_exceeds_scope, scope-profile-national-id, tenant-list-batches

## Knowledge Gaps
- **41 isolated node(s):** `@types/cytoscape`, `@types/react`, `@types/react-dom`, `typescript`, `github.com/YarKhan02/pathwise` (+36 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 156 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **20 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Finding` connect `Config Policy & Scan Analysis` to `Gin Route Adapter`, `Report Projection`, `SQLite Scan Storage`, `Schema Extraction (AST/types)`?**
  _High betweenness centrality (0.013) - this node is a cross-community bridge._
- **What connects `@types/cytoscape`, `@types/react`, `@types/react-dom` to the rest of the system?**
  _41 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Gin Route Adapter` be split into smaller, more focused modules?**
  _Cohesion score 0.06564364876385337 - nodes in this community are weakly interconnected._
- **Why does `runScan()` connect `Cobra Command Tree` to `SSA Loader & Packages`, `Config Policy & Scan Analysis`, `Schema Extraction (AST/types)`, `SQLite Scan Storage`, `Report & Serve Wiring`, `Project & Git Resolution`?**
  _High betweenness centrality (0.013) - this node is a cross-community bridge._
- **Should `Dashboard React UI` be split into smaller, more focused modules?**
  _Cohesion score 0.08846153846153847 - nodes in this community are weakly interconnected._
- **Why does `Config` connect `Config Policy & Scan Analysis` to `Gin Route Adapter`, `SSA Loader & Packages`, `Schema Extraction (AST/types)`?**
  _High betweenness centrality (0.011) - this node is a cross-community bridge._
- **Should `SSA Loader & Packages` be split into smaller, more focused modules?**
  _Cohesion score 0.07320024198427103 - nodes in this community are weakly interconnected._