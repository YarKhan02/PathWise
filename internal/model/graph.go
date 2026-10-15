package model

// NodeKind enumerates the authorization graph's node types.
type NodeKind string

const (
	NodeRole         NodeKind = "role"
	NodeIdentity     NodeKind = "identity"
	NodeEndpoint     NodeKind = "endpoint"
	NodeMiddleware   NodeKind = "middleware"
	NodeAuthCheck    NodeKind = "auth_check"
	NodeFunction     NodeKind = "function"
	NodeTable        NodeKind = "table"
	NodeRelationship NodeKind = "db_relationship"
	NodeQuery        NodeKind = "query"
	NodeResponseFld  NodeKind = "response_field"
	NodeScenario     NodeKind = "test_scenario"
)

// EdgeKind enumerates meaningful relationships between nodes.
type EdgeKind string

const (
	EdgeUsesMiddleware EdgeKind = "uses_middleware"
	EdgeCallsFunction  EdgeKind = "calls_function"
	EdgeChecksRole     EdgeKind = "checks_role"
	EdgeChecksOwner    EdgeKind = "checks_ownership"
	EdgeQueriesTable   EdgeKind = "queries_table"
	EdgeFiltersRecords EdgeKind = "filters_records"
	EdgeReturnsField   EdgeKind = "returns_field"
	EdgeVerifiedByTest EdgeKind = "verified_by_test"
)

// Operation is the effect a query has on a table.
type Operation string

const (
	OpSelect Operation = "select"
	OpInsert Operation = "insert"
	OpUpdate Operation = "update"
	OpDelete Operation = "delete"
)

// Node is a vertex in the projected authorization graph.
type Node struct {
	// ID is unique within a scan.
	ID string `json:"id"`

	Kind  NodeKind `json:"kind"`
	Label string   `json:"label"`

	// StableKey identifies the same logical entity across scans and git
	// revisions. Graph IDs are not comparable between scans because source
	// positions move; StableKey is what the diff engine joins on.
	StableKey string `json:"stable_key"`

	Attrs    map[string]any `json:"attrs,omitempty"`
	Evidence Evidence       `json:"evidence"`
}

// Edge is a relationship in the projected graph.
type Edge struct {
	ID   string   `json:"id"`
	Kind EdgeKind `json:"kind"`
	From string   `json:"from"`
	To   string   `json:"to"`

	Attrs    map[string]any `json:"attrs,omitempty"`
	Evidence Evidence       `json:"evidence"`
}

// Graph is a projection over the facts of a single scan. It is rebuilt from
// facts rather than mutated in place, so changing an inference rule never
// requires re-analyzing source.
type Graph struct {
	ScanID string `json:"scan_id"`
	Nodes  []Node `json:"nodes"`
	Edges  []Edge `json:"edges"`
}
