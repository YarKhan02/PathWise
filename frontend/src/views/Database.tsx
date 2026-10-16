import type { Scan } from "../lib/types";
import type { Selection } from "../components/Inspector";

export function Database({
  scan, query, selection, onSelect,
}: {
  scan: Scan; query: string; selection: Selection; onSelect: (s: Selection) => void;
}) {
  const q = query.trim().toLowerCase();
  const rows = scan.tables.filter((t) => !q || t.name.toLowerCase().includes(q));

  return (
    <div className="page">
      <div className="pagehead">
        <h1>Database</h1>
        <p>{scan.tables.length} tables reached by the scanned endpoints. Select a table to see which endpoints read or write it.</p>
      </div>

      {scan.tables.length === 0 ? (
        <div className="panelcard"><div style={{ padding: 24, color: "var(--muted)", fontSize: 13 }}>
          No tables were resolved. Queries built at runtime, or an unparsed query layer, leave the table set unknown.
        </div></div>
      ) : (
        <div className="panelcard">
          <table className="grid">
            <thead>
              <tr>
                <th>Table</th>
                <th style={{ width: 90 }}>Columns</th>
                <th style={{ width: 160 }}>Operations</th>
                <th style={{ width: 120 }}>Endpoints</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((t) => (
                <tr key={t.name}
                  className={selection?.kind === "table" && selection.name === t.name ? "sel" : ""}
                  onClick={() => onSelect({ kind: "table", name: t.name })}>
                  <td><span className="route">{t.name}</span></td>
                  <td style={{ color: "var(--muted)" }}>{t.columns.length || "—"}</td>
                  <td><div className="cellwrap">{t.operations.map((o) => <span className="chip" key={o}>{o.toLowerCase()}</span>)}</div></td>
                  <td style={{ color: "var(--muted)" }}>{t.endpoints.length}</td>
                </tr>
              ))}
              {rows.length === 0 && (
                <tr><td colSpan={4}><div style={{ padding: 24, textAlign: "center", color: "var(--muted)" }}>No tables match “{query}”.</div></td></tr>
              )}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
