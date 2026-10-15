import { ArrowRight, ShieldAlert, CircleHelp } from "lucide-react";
import type { Scan } from "../lib/types";
import type { View } from "../components/Sidebar";
import { SeverityBadge, EvidenceBadge } from "../components/primitives";

export function Overview({ scan, go }: { scan: Scan; go: (v: View, endpoint?: string) => void }) {
  const c = scan.counts;
  const sev = c.severity || {};
  const topFindings = [...scan.findings]
    .sort((a, b) => sevRank(b.severity) - sevRank(a.severity))
    .slice(0, 5);

  return (
    <div className="page">
      <div className="pagehead">
        <h1>Authorization overview</h1>
        <p>What the current scan established about {scan.project}. Every number below comes from the scan — nothing is estimated.</p>
      </div>

      <div className="metrics">
        <Metric v={c.endpoints} l="Endpoints" />
        <Metric v={c.roles} l="Roles" />
        <Metric v={c.tables} l="Tables reached" />
        <div className="metric">
          <div className="v">{c.findings}</div>
          <div className="l">Findings</div>
          <div className="sub">
            {sev.high ? <SeverityBadge severity="high" /> : null}
            {sev.high ? <span style={{ fontSize: 12, color: "var(--muted)" }}>{sev.high}</span> : null}
            {sev.medium ? <SeverityBadge severity="medium" /> : null}
            {sev.medium ? <span style={{ fontSize: 12, color: "var(--muted)" }}>{sev.medium}</span> : null}
            {!c.findings && <span style={{ fontSize: 12, color: "var(--muted)" }}>none</span>}
          </div>
        </div>
        <Metric v={c.unresolved} l="Unresolved paths" hint="endpoints reaching a query built at runtime" />
      </div>

      <div style={{ display: "grid", gridTemplateColumns: "1.4fr 1fr", gap: 16, alignItems: "start" }}>
        <section className="panelcard">
          <header>
            <ShieldAlert size={15} strokeWidth={1.8} />
            <h2>Findings to investigate</h2>
            <span className="count">{scan.findings.length}</span>
            <button className="btn ghost" style={{ marginLeft: "auto" }} onClick={() => go("findings")}>
              All findings <ArrowRight size={13} />
            </button>
          </header>
          {topFindings.length === 0 && (
            <div style={{ padding: 22, color: "var(--muted)", fontSize: 13 }}>
              No findings in this scan. Note that checks needing configuration may be inactive.
            </div>
          )}
          {topFindings.map((f, i) => (
            <div className="findingrow" key={f.id || i} onClick={() => go("graph", f.endpoint)}>
              <SeverityBadge severity={f.severity} />
              <div>
                <div className="ftitle">{f.title}</div>
                <div className="fmeta"><span className="mono">{f.endpoint}</span><EvidenceBadge confidence={f.confidence} /></div>
              </div>
              <ArrowRight size={15} color="var(--faint)" />
            </div>
          ))}
        </section>

        <section className="panelcard">
          <header><CircleHelp size={15} strokeWidth={1.8} /><h2>Verification status</h2></header>
          <div style={{ padding: "14px 16px", fontSize: 13, color: "var(--ink-2)" }}>
            <p style={{ marginTop: 0 }}>
              Conclusions here are from <b>static analysis</b> of source and types. Runtime verification
              against a running app is not part of this scan.
            </p>
            <div className="evline"><EvidenceBadge confidence="proven" /><span style={{ color: "var(--muted)" }}>proven from source &amp; control flow</span></div>
            <div className="evline"><EvidenceBadge confidence="likely" /><span style={{ color: "var(--muted)" }}>rests on a heuristic</span></div>
            <div className="evline"><EvidenceBadge confidence="possible" /><span style={{ color: "var(--muted)" }}>a premise is unresolved</span></div>
            <p style={{ marginBottom: 0, color: "var(--muted)" }}>
              {c.unresolved > 0
                ? `${c.unresolved} endpoint${c.unresolved === 1 ? "" : "s"} reach a query built at runtime; their data access is not fully known.`
                : "No unresolved query paths in this scan."}
            </p>
          </div>
        </section>
      </div>
    </div>
  );
}

function Metric({ v, l, hint }: { v: number; l: string; hint?: string }) {
  return (
    <div className="metric" title={hint}>
      <div className="v">{v}</div>
      <div className="l">{l}</div>
    </div>
  );
}

function sevRank(s: string): number {
  return { high: 3, medium: 2, low: 1, info: 0 }[s] ?? 0;
}
