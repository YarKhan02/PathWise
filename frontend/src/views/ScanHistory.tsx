import { useEffect, useMemo, useState } from "react";
import { ChevronRight, ChevronDown, ShieldAlert } from "lucide-react";
import type { Scan, TestRun, RuntimeOutcome } from "../lib/types";
import { MethodBadge, RoleTag } from "../components/primitives";

// ScanHistory lists every runtime verification run recorded for the project,
// newest first — the durable record (append-only in the store) so a run is never
// lost on a page refresh or a re-scan. Response bodies are not stored, so history
// shows outcomes only.

const OUTCOMES: { key: RuntimeOutcome; label: string; cls: string }[] = [
  { key: "divergent", label: "divergent", cls: "oc-divergent" },
  { key: "fail", label: "fail", cls: "oc-fail" },
  { key: "pass", label: "pass", cls: "oc-pass" },
  { key: "blocked", label: "blocked", cls: "oc-blocked" },
  { key: "unverified", label: "unverified", cls: "oc-unverified" },
];
const CLS = Object.fromEntries(OUTCOMES.map((o) => [o.key, o.cls])) as Record<RuntimeOutcome, string>;

export function ScanHistory({ scan, go }: { scan: Scan; go: (v: "graph", endpoint?: string) => void }) {
  const [history, setHistory] = useState<TestRun[]>(scan.history ?? []);

  // The App caches /api/scan at page load, so re-fetch on mount to pick up runs
  // executed this session (the server keeps them durably in the store).
  useEffect(() => {
    let live = true;
    fetch("api/scan")
      .then((r) => (r.ok ? r.json() : null))
      .then((d: Scan | null) => { if (live && d?.history) setHistory(d.history); })
      .catch(() => { /* keep the cached history */ });
    return () => { live = false; };
  }, []);

  if (history.length === 0) {
    return (
      <div className="page">
        <div className="pagehead"><h1>Scan history</h1></div>
        <div className="panelcard"><div style={{ padding: 24, color: "var(--muted)", fontSize: 13 }}>
          No runtime runs recorded yet. Run a verification on the Runtime Tests page — every run is saved here,
          across scans, and survives refreshes.
        </div></div>
      </div>
    );
  }

  return (
    <div className="page">
      <div className="pagehead">
        <h1>Scan history</h1>
        <p>{history.length} runtime run{history.length === 1 ? "" : "s"} recorded for this project, newest first. Each run is kept as observed — never merged into the static findings.</p>
      </div>
      <div className="hist-list">
        {history.map((run, i) => <RunCard key={`${run.scanId}|${run.executedAt}|${i}`} run={run} defaultOpen={i === 0} roles={scan.roles} go={go} />)}
      </div>
    </div>
  );
}

function RunCard({ run, defaultOpen, roles, go }: { run: TestRun; defaultOpen: boolean; roles: string[]; go: (v: "graph", endpoint?: string) => void }) {
  const [open, setOpen] = useState(defaultOpen);
  const counts = useMemo(() => {
    const c: Record<string, number> = {};
    for (const r of run.results) c[r.outcome] = (c[r.outcome] ?? 0) + 1;
    return c;
  }, [run]);
  const divergent = counts["divergent"] ?? 0;
  const rows = [...run.results].sort((a, b) => rank(b.outcome) - rank(a.outcome) || a.path.localeCompare(b.path) || a.role.localeCompare(b.role));

  return (
    <div className="hist-run">
      <div className="hist-head" onClick={() => setOpen((o) => !o)}>
        {open ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
        <span className="hist-time">{fmtTime(run.executedAt)}</span>
        {run.revision && <span className="hist-rev" title="git revision">{run.revision.slice(0, 10)}</span>}
        {divergent > 0 && <span className="hist-diverge"><ShieldAlert size={12} /> {divergent} divergent</span>}
        <span className="hist-badges">
          {OUTCOMES.filter((o) => counts[o.key]).map((o) => (
            <span key={o.key} className={`badge ${o.cls}`}>{counts[o.key]} {o.label}</span>
          ))}
        </span>
        <span className="hist-total">{run.results.length}</span>
      </div>
      {open && (
        <div className="hist-rows">
          {rows.map((t, i) => (
            <div className="rt-row" key={`${t.endpoint}|${t.role}|${i}`} style={{ cursor: "pointer" }} onClick={() => go("graph", t.endpoint)}>
              <MethodBadge method={t.method} />
              <div>
                <div className="rt-path">{t.path}</div>
                <div className="rt-meta">
                  <RoleTag role={t.role} roles={roles} />
                  <span className="badge conf">predicted {t.expectation}</span>
                  {t.status ? <span className="badge conf">HTTP {t.status}</span> : null}
                  <span className="rt-detail">{t.detail}</span>
                </div>
              </div>
              <span className={`badge ${CLS[t.outcome]}`}>{t.outcome}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function rank(o: string): number {
  return { divergent: 4, fail: 3, pass: 2, blocked: 1, unverified: 0 }[o] ?? 0;
}
function fmtTime(iso: string): string {
  const d = new Date(iso);
  return isNaN(d.getTime()) ? iso : d.toLocaleString();
}
