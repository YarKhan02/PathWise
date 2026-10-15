import { useMemo, useState } from "react";
import { FileCode2, X } from "lucide-react";
import type { Scan } from "../lib/types";
import type { View } from "../components/Sidebar";
import { SeverityBadge, ConfidenceBadge, EvidenceBadge, MethodBadge } from "../components/primitives";

export function Findings({ scan, query, go }: { scan: Scan; query: string; go: (v: View, endpoint?: string) => void }) {
  const [sev, setSev] = useState("");
  const [kind, setKind] = useState("");

  const kinds = useMemo(() => [...new Set(scan.findings.map((f) => f.kind))].sort(), [scan]);
  const q = query.trim().toLowerCase();

  const rows = [...scan.findings]
    .filter((f) => (!sev || f.severity === sev) && (!kind || f.kind === kind))
    .filter((f) => !q || `${f.title} ${f.endpoint} ${f.detail}`.toLowerCase().includes(q))
    .sort((a, b) => rank(b.severity) - rank(a.severity));

  const epMethod = (endpoint: string) => scan.endpoints.find((e) => e.id === endpoint)?.method;

  return (
    <div className="page">
      <div className="pagehead">
        <h1>Findings</h1>
        <p>Authorization discrepancies this scan established. Each cites the evidence it rests on; uncertainty is labeled, not hidden.</p>
      </div>

      <div className="toolbar">
        <select className="filter" value={sev} onChange={(e) => setSev(e.target.value)}>
          <option value="">All severities</option>
          <option value="high">High</option>
          <option value="medium">Medium</option>
          <option value="low">Low</option>
        </select>
        <select className="filter" value={kind} onChange={(e) => setKind(e.target.value)}>
          <option value="">All kinds</option>
          {kinds.map((k) => <option key={k} value={k}>{k}</option>)}
        </select>
        {(sev || kind) && <button className="btn ghost" onClick={() => { setSev(""); setKind(""); }}><X size={13} /> Reset</button>}
        <span style={{ marginLeft: "auto", color: "var(--muted)", fontSize: 12.5 }}>{rows.length} shown</span>
      </div>

      <div className="panelcard">
        {rows.length === 0 && (
          <div style={{ padding: 24, color: "var(--muted)", fontSize: 13, textAlign: "center" }}>
            {scan.findings.length === 0 ? "No findings in this scan. Config-dependent checks may be inactive." : "No findings match these filters."}
          </div>
        )}
        {rows.map((f, i) => {
          const m = epMethod(f.endpoint);
          return (
            <div className="findingrow" key={f.id || i} onClick={() => go("graph", f.endpoint)}>
              <SeverityBadge severity={f.severity} />
              <div>
                <div className="ftitle">{f.title}</div>
                <div className="fmeta">
                  {m && <MethodBadge method={m} />}
                  <span className="mono">{f.endpoint}</span>
                  <ConfidenceBadge confidence={f.confidence} />
                  <EvidenceBadge confidence={f.confidence} />
                  <span style={{ opacity: .7 }}>{f.kind}</span>
                </div>
                <div className="fdetail">{f.detail}</div>
                {f.locations && f.locations[0] && (
                  <div className="floc" style={{ fontFamily: "var(--mono)", fontSize: 11, color: "var(--muted)", marginTop: 6 }}>
                    <FileCode2 size={11} style={{ verticalAlign: "-1px", marginRight: 4 }} />
                    {(f.locations[0].file.split("/").pop() ?? "")}:{f.locations[0].start_line}
                  </div>
                )}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

function rank(s: string): number {
  return { high: 3, medium: 2, low: 1, info: 0 }[s] ?? 0;
}
