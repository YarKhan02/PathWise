import { useMemo, useState } from "react";
import { X } from "lucide-react";
import type { Scan, Endpoint } from "../lib/types";
import type { Selection } from "../components/Inspector";
import { MethodBadge, RoleTag, SeverityBadge } from "../components/primitives";

export function Endpoints({
  scan, query, selection, onSelect,
}: {
  scan: Scan; query: string; selection: Selection; onSelect: (s: Selection) => void;
}) {
  const [method, setMethod] = useState("");
  const [role, setRole] = useState("");

  const methods = useMemo(
    () => [...new Set(scan.endpoints.map((e) => e.method))].sort(),
    [scan],
  );

  const q = query.trim().toLowerCase();
  const rows = scan.endpoints.filter((e) => {
    if (method && e.method !== method) return false;
    if (role && !(role === "__public" ? e.roles.length === 0 : e.roles.includes(role))) return false;
    if (q && !searchText(e).includes(q)) return false;
    return true;
  });

  return (
    <div className="page">
      <div className="pagehead">
        <h1>Endpoints</h1>
        <p>{scan.endpoints.length} routes discovered. Select one to trace its authorization path and data access.</p>
      </div>

      <div className="toolbar">
        <select className="filter" value={method} onChange={(e) => setMethod(e.target.value)}>
          <option value="">All methods</option>
          {methods.map((m) => <option key={m} value={m}>{m}</option>)}
        </select>
        <select className="filter" value={role} onChange={(e) => setRole(e.target.value)}>
          <option value="">All roles</option>
          {scan.roles.map((r) => <option key={r} value={r}>{r}</option>)}
          <option value="__public">public (no gate)</option>
        </select>
        {(method || role) && (
          <button className="btn ghost" onClick={() => { setMethod(""); setRole(""); }}>
            <X size={13} /> Reset
          </button>
        )}
        <span style={{ marginLeft: "auto", color: "var(--muted)", fontSize: 12.5 }}>{rows.length} shown</span>
      </div>

      <div className="panelcard">
        <table className="grid">
          <thead>
            <tr>
              <th style={{ width: 70 }}>Method</th>
              <th>Route</th>
              <th style={{ width: 180 }}>Roles</th>
              <th style={{ width: 220 }}>Data access</th>
              <th style={{ width: 90 }}>Findings</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((e) => (
              <tr key={e.id}
                className={selection?.kind === "endpoint" && selection.id === e.id ? "sel" : ""}
                onClick={() => onSelect({ kind: "endpoint", id: e.id })}>
                <td><MethodBadge method={e.method} /></td>
                <td><span className="route" title={`${e.handler}`}>{e.path}</span></td>
                <td>
                  <div className="cellwrap">
                    {e.roles.length === 0
                      ? <span className="dash">{e.middleware.length ? "authenticated" : "public"}</span>
                      : e.roles.map((r) => <RoleTag key={r} role={r} roles={scan.roles} />)}
                  </div>
                </td>
                <td>
                  <div className="cellwrap">
                    {e.tables.length === 0 && !e.unresolved && <span className="dash">—</span>}
                    {e.tables.map((t) => (
                      <span className="chip" key={t.table}>{t.table} <span className="op">{t.operations.join(",")}</span></span>
                    ))}
                    {e.unresolved && <span className="badge sev-medium">dynamic</span>}
                  </div>
                </td>
                <td>{e.findings.length ? <SeverityBadge severity={topSeverity(e)} /> : <span className="dash">—</span>}</td>
              </tr>
            ))}
            {rows.length === 0 && (
              <tr><td colSpan={5}><div style={{ padding: 24, textAlign: "center", color: "var(--muted)" }}>No endpoints match these filters.</div></td></tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function searchText(e: Endpoint): string {
  return [e.method, e.path, e.handler, ...e.roles, ...e.tables.map((t) => t.table)].join(" ").toLowerCase();
}

function topSeverity(e: Endpoint): string {
  const order = ["high", "medium", "low", "info"];
  for (const s of order) if (e.findings.some((f) => f.severity === s)) return s;
  return "low";
}
