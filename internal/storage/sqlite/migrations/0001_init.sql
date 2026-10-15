-- PathWise's own schema. Separate from the analyzed application's PostgreSQL
-- database: this holds analysis results and history only.
--
-- Design rule: scans are immutable. A new scan never overwrites an older
-- one's facts, because historical evidence is what makes revision comparison
-- and finding-lifecycle tracking possible.

CREATE TABLE projects (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    root_path   TEXT NOT NULL,
    module_path TEXT NOT NULL DEFAULT '',
    created_at  TEXT NOT NULL
);

CREATE UNIQUE INDEX idx_projects_root ON projects (root_path);

CREATE TABLE scans (
    id               TEXT PRIMARY KEY,
    project_id       TEXT NOT NULL REFERENCES projects (id),
    git_revision     TEXT NOT NULL DEFAULT '',
    git_dirty        INTEGER NOT NULL DEFAULT 0,
    analyzer_version TEXT NOT NULL,

    -- live | migrations_only | none. Recorded because a finding derived from
    -- an inferred schema is weaker than one derived from the live database,
    -- and a reader must be able to tell which they are looking at.
    schema_source    TEXT NOT NULL DEFAULT 'none',

    -- Finding kinds that could not run, as JSON. A clean scan with checks
    -- disabled is a partial scan, and must never read as an all-clear.
    disabled_checks  TEXT NOT NULL DEFAULT '[]',

    status           TEXT NOT NULL DEFAULT 'running', -- running|complete|failed
    error            TEXT NOT NULL DEFAULT '',
    started_at       TEXT NOT NULL,
    finished_at      TEXT
);

CREATE INDEX idx_scans_project ON scans (project_id, started_at DESC);

-- Facts are the source of truth. Graph nodes and edges below are a projection
-- over this table and may be rebuilt at any time without re-parsing source.
CREATE TABLE facts (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    scan_id           TEXT NOT NULL REFERENCES scans (id),
    kind              TEXT NOT NULL,
    subject           TEXT NOT NULL,
    attrs             TEXT NOT NULL DEFAULT '{}',
    status            TEXT NOT NULL,
    unresolved_reason TEXT NOT NULL DEFAULT '',
    details           TEXT NOT NULL DEFAULT '',
    scenario_id       TEXT NOT NULL DEFAULT '',
    fingerprint       TEXT NOT NULL
);

CREATE UNIQUE INDEX idx_facts_dedup ON facts (scan_id, fingerprint);
CREATE INDEX idx_facts_kind ON facts (scan_id, kind);
CREATE INDEX idx_facts_subject ON facts (scan_id, subject);

CREATE TABLE fact_locations (
    fact_id    INTEGER NOT NULL REFERENCES facts (id) ON DELETE CASCADE,
    file       TEXT NOT NULL,
    start_line INTEGER NOT NULL,
    end_line   INTEGER NOT NULL
);

CREATE INDEX idx_fact_locations_fact ON fact_locations (fact_id);

CREATE TABLE graph_nodes (
    scan_id    TEXT NOT NULL REFERENCES scans (id),
    id         TEXT NOT NULL,
    kind       TEXT NOT NULL,
    label      TEXT NOT NULL,
    stable_key TEXT NOT NULL,
    attrs      TEXT NOT NULL DEFAULT '{}',
    status     TEXT NOT NULL,
    unresolved_reason TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (scan_id, id)
);

CREATE INDEX idx_graph_nodes_stable ON graph_nodes (stable_key);
CREATE INDEX idx_graph_nodes_kind ON graph_nodes (scan_id, kind);

CREATE TABLE graph_edges (
    scan_id    TEXT NOT NULL REFERENCES scans (id),
    id         TEXT NOT NULL,
    kind       TEXT NOT NULL,
    from_node  TEXT NOT NULL,
    to_node    TEXT NOT NULL,
    attrs      TEXT NOT NULL DEFAULT '{}',
    status     TEXT NOT NULL,
    unresolved_reason TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (scan_id, id)
);

CREATE INDEX idx_graph_edges_from ON graph_edges (scan_id, from_node);
CREATE INDEX idx_graph_edges_to ON graph_edges (scan_id, to_node);

CREATE TABLE findings (
    id                 TEXT PRIMARY KEY,
    scan_id            TEXT NOT NULL REFERENCES scans (id),
    kind               TEXT NOT NULL,
    severity           TEXT NOT NULL,
    confidence         TEXT NOT NULL,
    endpoint           TEXT NOT NULL DEFAULT '',
    stable_key         TEXT NOT NULL,
    title              TEXT NOT NULL,
    detail             TEXT NOT NULL DEFAULT '',
    remedy             TEXT NOT NULL DEFAULT '',
    locations          TEXT NOT NULL DEFAULT '[]',
    premises           TEXT NOT NULL DEFAULT '[]',
    runtime_divergence INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_findings_scan ON findings (scan_id, kind);
CREATE INDEX idx_findings_stable ON findings (stable_key);

-- Phase 5 tables. Created now so that the storage schema has one migration
-- history rather than a later rewrite.
CREATE TABLE test_scenarios (
    id            TEXT PRIMARY KEY,
    scan_id       TEXT NOT NULL REFERENCES scans (id),
    endpoint      TEXT NOT NULL,
    identity_ref  TEXT NOT NULL DEFAULT '',
    description   TEXT NOT NULL DEFAULT '',
    expectation   TEXT NOT NULL DEFAULT '',
    safety_class  TEXT NOT NULL DEFAULT 'deny',
    created_at    TEXT NOT NULL
);

CREATE TABLE test_runs (
    id            TEXT PRIMARY KEY,
    scenario_id   TEXT NOT NULL REFERENCES test_scenarios (id),
    request       TEXT NOT NULL DEFAULT '{}',
    status_code   INTEGER,
    response_meta TEXT NOT NULL DEFAULT '{}',

    -- pass | fail | blocked | unverified | divergent
    outcome       TEXT NOT NULL,
    limitation    TEXT NOT NULL DEFAULT '',
    error         TEXT NOT NULL DEFAULT '',
    executed_at   TEXT NOT NULL
);

CREATE TABLE sensitive_fields (
    project_id TEXT NOT NULL REFERENCES projects (id),
    table_name TEXT NOT NULL,
    column_name TEXT NOT NULL,
    note       TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (project_id, table_name, column_name)
);

CREATE TABLE git_changes (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    base_scan   TEXT NOT NULL REFERENCES scans (id),
    head_scan   TEXT NOT NULL REFERENCES scans (id),
    change_type TEXT NOT NULL,
    stable_key  TEXT NOT NULL,
    detail      TEXT NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_git_changes_pair ON git_changes (base_scan, head_scan);
