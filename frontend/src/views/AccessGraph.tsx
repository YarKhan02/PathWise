import { useEffect, useMemo, useRef, useState } from "react";
import cytoscape, { type Core, type ElementDefinition } from "cytoscape";
import { Maximize2, RotateCcw } from "lucide-react";
import type { Scan } from "../lib/types";
import type { Selection } from "../components/Inspector";
import { roleColorVar } from "../lib/api";

const METHOD_COLOR: Record<string, string> = {
  GET: "#2b8a3e", POST: "#1971c2", PUT: "#e8590c", PATCH: "#e8590c", DELETE: "#c92a2a",
};
const ACCENT = "#3a53d0";
const TABLE_FILL = "#eef1fd";
const TABLE_BORDER = "#c3ccf4";
const INK = "#1a1a18";
const MUTED = "#9a9a92";

export function AccessGraph({
  scan, focus, selection, onSelect,
}: {
  scan: Scan; focus: string | null; selection: Selection; onSelect: (s: Selection) => void;
}) {
  const boxRef = useRef<HTMLDivElement>(null);
  const cyRef = useRef<Core | null>(null);
  const [role, setRole] = useState("");

  const elements = useMemo(() => buildElements(scan), [scan]);

  // Build the graph once per scan.
  useEffect(() => {
    if (!boxRef.current) return;
    const cy = cytoscape({
      container: boxRef.current,
      elements,
      wheelSensitivity: 0.25,
      style: stylesheet(scan),
      layout: { name: "preset" },
    });
    cyRef.current = cy;

    cy.on("tap", "node", (ev) => {
      const n = ev.target;
      if (n.data("kind") === "endpoint") onSelect({ kind: "endpoint", id: n.data("ref") });
      else onSelect({ kind: "table", name: n.data("ref") });
    });
    cy.on("tap", (ev) => {
      if (ev.target === cy) onSelect(null);
    });
    cy.fit(undefined, 40);

    return () => cy.destroy();
  }, [elements, scan]);

  // Reflect the current selection as highlight + neighborhood focus.
  useEffect(() => {
    const cy = cyRef.current;
    if (!cy) return;
    cy.batch(() => {
      cy.elements().removeClass("hot faded");
      const id = selection?.kind === "endpoint" ? `e:${selection.id}`
        : selection?.kind === "table" ? `t:${selection.name}` : null;
      if (!id) return;
      const node = cy.getElementById(id);
      if (node.empty()) return;
      const hood = node.closedNeighborhood();
      cy.elements().addClass("faded");
      hood.removeClass("faded").addClass("hot");
    });
  }, [selection]);

  // Role filter: show only endpoints with the role and the tables they reach.
  useEffect(() => {
    const cy = cyRef.current;
    if (!cy) return;
    cy.batch(() => {
      if (!role) {
        cy.elements().style("display", "element");
      } else {
        cy.nodes('[kind="endpoint"]').forEach((n) => {
          const roles: string[] = n.data("roles") || [];
          n.style("display", roles.includes(role) ? "element" : "none");
        });
        cy.nodes('[kind="table"]').forEach((t) => {
          const anyVisible = t.connectedEdges().sources().some((s) => s.style("display") !== "none");
          t.style("display", anyVisible ? "element" : "none");
        });
        cy.edges().forEach((e) => {
          e.style("display", e.source().style("display") !== "none" && e.target().style("display") !== "none" ? "element" : "none");
        });
      }
    });
    cy.fit(cy.elements(":visible"), 40);
  }, [role]);

  // External focus (navigated from a finding) → select and center.
  useEffect(() => {
    const cy = cyRef.current;
    if (!cy || !focus) return;
    const node = cy.getElementById(`e:${focus}`);
    if (node.nonempty()) {
      onSelect({ kind: "endpoint", id: focus });
      cy.animate({ center: { eles: node }, zoom: 1 }, { duration: 250 });
    }
  }, [focus]);

  return (
    <div className="graphcol">
      <div className="graphtoolbar">
        <strong style={{ fontSize: 13 }}>Access graph</strong>
        <span style={{ color: "var(--muted)", fontSize: 12 }}>
          {scan.counts.endpoints} endpoints · {scan.counts.tables} tables
        </span>
        <span style={{ flex: 1 }} />
        <div className="rolebar">
          <button className={`rolebtn ${role === "" ? "active" : ""}`} onClick={() => setRole("")}>All roles</button>
          {scan.roles.map((r) => (
            <button key={r} className={`rolebtn ${role === r ? "active" : ""}`} onClick={() => setRole(r)}>
              <span className="dot" style={{ background: roleColorVar(r, scan.roles) }} /> {r}
            </button>
          ))}
        </div>
        <button className="iconbtn" title="Fit" onClick={() => cyRef.current?.fit(cyRef.current.elements(":visible"), 40)}><Maximize2 size={15} /></button>
        <button className="iconbtn" title="Reset" onClick={() => { setRole(""); onSelect(null); cyRef.current?.fit(undefined, 40); }}><RotateCcw size={15} /></button>
      </div>
      <div id="cy" ref={boxRef} />
      <div className="legend">
        <div className="row"><span className="sw" style={{ background: "#fff", border: `2px solid ${ACCENT}`, borderRadius: 7 }} /> Endpoint</div>
        <div className="row"><span className="sw" style={{ background: TABLE_FILL, border: `1px solid ${TABLE_BORDER}` }} /> Table</div>
        <div className="row"><span className="sw" style={{ background: "transparent", borderBottom: `2px solid ${MUTED}` }} /> reads / writes</div>
      </div>
    </div>
  );
}

function buildElements(scan: Scan): ElementDefinition[] {
  const els: ElementDefinition[] = [];
  const stepE = 46, stepT = 46;
  const leftX = 0, rightX = 640;

  const sortedE = [...scan.endpoints].sort((a, b) => a.id.localeCompare(b.id));
  sortedE.forEach((e, i) => {
    els.push({
      data: {
        id: `e:${e.id}`, kind: "endpoint", ref: e.id,
        label: `${e.method} ${e.path}`, method: e.method, roles: e.roles,
        flagged: e.findings.length > 0,
      },
      position: { x: leftX, y: i * stepE },
    });
  });

  scan.tables.forEach((t, i) => {
    els.push({
      data: { id: `t:${t.name}`, kind: "table", ref: t.name, label: t.name },
      position: { x: rightX, y: i * stepT + (sortedE.length - scan.tables.length) * stepE / 2 },
    });
  });

  for (const e of sortedE) {
    for (const t of e.tables) {
      els.push({ data: { id: `e:${e.id}->t:${t.table}`, source: `e:${e.id}`, target: `t:${t.table}`, op: t.operations.join(",") } });
    }
  }
  return els;
}

// Returns an untyped stylesheet: Cytoscape supports function mappers at
// runtime (border color by HTTP method), which its strict TS types reject.
function stylesheet(_scan: Scan): cytoscape.Stylesheet[] {
  return ([
    {
      selector: 'node[kind="endpoint"]',
      style: {
        shape: "round-rectangle",
        "background-color": "#ffffff",
        "border-width": 2,
        "border-color": (n: cytoscape.NodeSingular) => METHOD_COLOR[n.data("method")] ?? "#5c636e",
        label: "data(label)",
        "font-family": "IBM Plex Mono, monospace",
        "font-size": 10,
        color: INK,
        "text-valign": "center",
        "text-halign": "right",
        "text-margin-x": 6,
        width: 12, height: 12,
        "text-max-width": "260px",
      },
    },
    {
      selector: 'node[kind="endpoint"][?flagged]',
      style: { "border-color": "#c92a2a", width: 15, height: 15 },
    },
    {
      selector: 'node[kind="table"]',
      style: {
        shape: "round-rectangle",
        "background-color": TABLE_FILL,
        "border-width": 1,
        "border-color": TABLE_BORDER,
        label: "data(label)",
        "font-family": "IBM Plex Mono, monospace",
        "font-size": 11,
        color: "#2b3fa6",
        "text-valign": "center",
        "text-halign": "left",
        "text-margin-x": -8,
        padding: "7px",
        width: "label", height: 20,
      },
    },
    {
      selector: "edge",
      style: {
        width: 1.4,
        "line-color": "#d6d6d0",
        "curve-style": "bezier",
        "target-arrow-shape": "triangle",
        "target-arrow-color": "#d6d6d0",
        "arrow-scale": 0.7,
        opacity: 0.75,
      },
    },
    { selector: ".faded", style: { opacity: 0.12 } },
    {
      selector: "node.hot",
      style: { "border-color": ACCENT, "border-width": 3, "z-index": 10 },
    },
    {
      selector: "edge.hot",
      style: { "line-color": ACCENT, "target-arrow-color": ACCENT, width: 2.4, opacity: 1, label: "data(op)",
        "font-family": "IBM Plex Mono, monospace", "font-size": 9, color: MUTED, "text-background-color": "#fff", "text-background-opacity": 1, "text-background-padding": "2px" },
    },
  ] as unknown) as cytoscape.Stylesheet[];
}
