# Graph Report - PathWise  (2026-10-14)

## Corpus Check
- Corpus is ~17,707 words - fits in a single context window. You may not need a graph.

## Summary
- 223 nodes · 517 edges · 17 communities (11 shown, 6 thin omitted)
- Extraction: 90% EXTRACTED · 10% INFERRED · 0% AMBIGUOUS · INFERRED: 50 edges (avg confidence: 0.86)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- Test Fixture Gin App
- SQLite Storage Layer
- CLI & Project Resolution
- Test Suites
- Known-Answer Corpus Loader
- Evidence & Graph Model
- Policy Configuration
- Call-Graph Analysis & Role Findings
- Tenancy & Scope Findings
- Corpus Marker Parsing
- Finding Taxonomy
- Evidence Honesty Principle
- Auth Check States
- Route Normalization
- Module Root
- Test Package
- Public-Endpoint Control

## God Nodes (most connected - your core abstractions)
1. `Store` - 17 edges
2. `runScan()` - 11 edges
3. `Evidence` - 11 edges
4. `Config` - 10 edges
5. `App` - 10 edges
6. `FindingKind` - 10 edges
7. `Scan` - 10 edges
8. `newTestStore()` - 9 edges
9. `Open()` - 8 edges
10. `CurrentUser()` - 8 edges

## Surprising Connections (you probably didn't know these)
- `main()` --calls--> `NewRootCommand()`  [EXTRACTED]
  cmd/pathwise/main.go → internal/cli/root.go
- `ownership-enforced negative control` --conceptually_related_to--> `missing_ownership_check finding`  [INFERRED]
  testdata/apps/basic-gin/expected.yaml → CLAUDE.md
- `dynamic-sql-search expectation` --references--> `EvidenceStatus`  [INFERRED]
  testdata/apps/basic-gin/expected.yaml → CLAUDE.md
- `basic-gin corpus app` --references--> `Known-answer corpus`  [INFERRED]
  testdata/apps/basic-gin/expected.yaml → CLAUDE.md
- `basic-gin .pathwise.yaml policy` --conceptually_related_to--> `pathwise.example.yaml policy template`  [INFERRED]
  testdata/apps/basic-gin/.pathwise.yaml → configs/pathwise.example.yaml

## Import Cycles
- None detected.

## Hyperedges (group relationships)
- **Fixed four-finding taxonomy** — claude_missing_ownership_check, claude_role_check_unenforced, claude_missing_tenant_filter, claude_response_field_exceeds_scope [EXTRACTED 1.00]
- **Evidence honesty shared model** — claude_evidencestatus, claude_paramprovenance, claude_auth_check_candidate, claude_auth_check_enforced, claude_facts_truth [INFERRED 0.75]
- **basic-gin known-answer set** — testdata_apps_basic_gin_expected_basic_gin, testdata_apps_basic_gin_expected_idor_get_student, testdata_apps_basic_gin_expected_dynamic_sql_search, testdata_apps_basic_gin_expected_ownership_enforced, testdata_apps_basic_gin_expected_login_public_by_design [EXTRACTED 1.00]

## Communities (17 total, 6 thin omitted)

### Community 0 - "Test Fixture Gin App"
Cohesion: 0.11
Nodes (29): Batch, Student, StudentProfile, User, CanAccessStudent(), CurrentUser(), HasRole(), IsAdmin() (+21 more)

### Community 1 - "SQLite Storage Layer"
Cohesion: 0.11
Nodes (12): Fact, FactKind, boolToInt(), newID(), nowRFC3339(), Open(), ParamProvenance, Project (+4 more)

### Community 2 - "CLI & Project Resolution"
Cohesion: 0.14
Nodes (18): globalFlags, main(), NewRootCommand(), notImplemented(), describeRevision(), newScanCommand(), reasonDisabled(), reportCoverage() (+10 more)

### Community 3 - "Test Suites"
Cohesion: 0.18
Nodes (20): corpusRoot(), TestBasicGinExpectations(), TestCorpusCoversEveryFindingKind(), TestCorpusHasNegativeControls(), TestCorpusLoads(), TestMarkerParsing(), TestMarkersIgnoreNonComments(), CodeConfirmed() (+12 more)

### Community 4 - "Known-Answer Corpus Loader"
Cohesion: 0.22
Nodes (13): App, ExpectedFinding, ExpectedForeignKey, ExpectedRoute, ExpectedSafe, ExpectedSchema, ExpectedTable, ExpectedUnresolved (+5 more)

### Community 5 - "Evidence & Graph Model"
Cohesion: 0.24
Nodes (12): SourceLocation, UnresolvedReason, Inferred(), Unresolved(), Edge, EdgeKind, Evidence, EvidenceStatus (+4 more)

### Community 6 - "Policy Configuration"
Cohesion: 0.21
Nodes (4): EndpointScope, Tenancy, Config, Load()

### Community 7 - "Call-Graph Analysis & Role Findings"
Cohesion: 0.22
Nodes (6): missing_ownership_check finding, role_check_unenforced finding, idor-get-student expectation, ownership-enforced negative control, role-discarded-admin-report expectation, role-nondominating-dashboard expectation

### Community 8 - "Tenancy & Scope Findings"
Cohesion: 0.36
Nodes (9): Known-answer corpus, missing_tenant_filter finding, PathWise, response_field_exceeds_scope finding, pathwise.example.yaml policy template, basic-gin corpus app, scope-profile-national-id expectation, tenant-list-batches expectation (+1 more)

### Community 9 - "Corpus Marker Parsing"
Cohesion: 0.62
Nodes (5): Marker, MarkerKind, parseMarker(), scanFile(), ScanMarkers()

### Community 10 - "Finding Taxonomy"
Cohesion: 0.48
Nodes (5): AllFindingKinds(), FindingKind, Confidence, Finding, Severity

## Knowledge Gaps
- **12 isolated node(s):** `github.com/YarKhan02/pathwise`, `pathwise.test/basic-gin`, `PathWise`, `auth_check_candidate fact`, `auth_check_enforced fact` (+7 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 30 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **6 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `runScan()` connect `CLI & Project Resolution` to `SQLite Storage Layer`, `Policy Configuration`?**
  _High betweenness centrality (0.054) - this node is a cross-community bridge._
- **What connects `github.com/YarKhan02/pathwise`, `pathwise.test/basic-gin`, `PathWise` to the rest of the system?**
  _12 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Test Fixture Gin App` be split into smaller, more focused modules?**
  _Cohesion score 0.1076923076923077 - nodes in this community are weakly interconnected._
- **Why does `Store` connect `SQLite Storage Layer` to `Test Suites`?**
  _High betweenness centrality (0.042) - this node is a cross-community bridge._
- **Should `SQLite Storage Layer` be split into smaller, more focused modules?**
  _Cohesion score 0.11095305832147938 - nodes in this community are weakly interconnected._
- **Why does `Config` connect `Policy Configuration` to `CLI & Project Resolution`?**
  _High betweenness centrality (0.027) - this node is a cross-community bridge._
- **Should `CLI & Project Resolution` be split into smaller, more focused modules?**
  _Cohesion score 0.1431451612903226 - nodes in this community are weakly interconnected._