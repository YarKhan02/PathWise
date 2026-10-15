import { useMemo, useState } from "react";
import { ShieldAlert, X } from "lucide-react";
import type { Scan, RuntimeTest, RuntimeOutcome } from "../lib/types";
import type { View } from "../components/Sidebar";
import { MethodBadge, RoleTag } from "../components/primitives";
import { Unavailable } from "./Unavailable";

// Order and labels for the five outcomes. Divergent leads because it is the
// whole point of runtime verification: a static "protected" a live request
// defeated. Unverified is deliberately not styled as a pass — "could not
// conclude" must never read as "safe".
const OUTCOMES: { key: RuntimeOutcome; label: string; cls: string }[] = [
  { key: "divergent", label: "Divergent", cls: "oc-divergent" },
  { key: "fail", label: "Fail", cls: "oc-fail" },
  { key: "pass", label: "Pass", cls: "oc-pass" },
  { key: "blocked", label: "Blocked", cls: "oc-blocked" },
  { key: "unverified", label: "Unverified", cls: "oc-unverified" },
];
const CLS: Record<RuntimeOutcome, string> = Object.fromEntries(
  OUTCOMES.map((o) => [o.key, o.cls]),
) as Record<RuntimeOutcome, string>;

export function RuntimeTests({ scan, go }: { scan: Scan; go: (v: View, endpoint?: string) => void }) {
  const [outcome, setOutcome] = useState<RuntimeOutcome | "">("");

  const counts = useMemo(() => {
    const c: Record<string, number> = {};
    for (const t of scan.runtimeTests) c[t.outcome] = (c[t.outcome] ?? 0) + 1;
    return c;
  }, [scan]);

  // No run recorded: say so plainly rather than implying the app is untested-safe.
  if (scan.runtimeTests.length === 0) {
    return (
      <Unavailable
        title="Runtime verification"
        reason="No runtime test has been recorded against this scan. Run `pathwise test --base-url <url> --identity role=token` to exercise the running app as each identity and compare what it does with the static prediction."
      />
    );
  }

  const divergences = scan.runtimeTests.filter((t) => t.outcome === "divergent");
  const rows = [...scan.runtimeTests]
    .filter((t) => !outcome || t.outcome === outcome)
    .sort((a, b) => rank(b.outcome) - rank(a.outcome) || a.path.localeCompare(b.path) || a.role.localeCompare(b.role));

  return (
    <div className="page">
      <div className="pagehead">
        <h1>Runtime verification</h1>
        <p>
          What the running application actually did as each identity, compared with what the scan predicted.
          Recorded apart from the static findings and never merged with them
          {scan.runtimeTestedAt ? ` · last run ${fmtTime(scan.runtimeTestedAt)}` : ""}.
        </p>
      </div>

      {divergences.length > 0 && (
        <div className="rt-banner">
          <div className="ic"><ShieldAlert size={20} /></div>
          <div>
            <h3>{divergences.length} divergence{divergences.length > 1 ? "s" : ""} — the highest-confidence signal PathWise can produce</h3>
            <p>The analysis said these endpoints were protected and the running app served them anyway to an identity without the required role.</p>
          </div>
        </div>
      )}

      <div className="toolbar">
        {OUTCOMES.filter((o) => counts[o.key]).map((o) => (
          <button
            key={o.key}
            className={`badge ${o.cls}`}
            style={{ cursor: "pointer", opacity: outcome && outcome !== o.key ? 0.45 : 1 }}
            onClick={() => setOutcome(outcome === o.key ? "" : o.key)}
          >
            {o.label} · {counts[o.key]}
          </button>
        ))}
        {outcome && <button className="btn ghost" onClick={() => setOutcome("")}><X size={13} /> Reset</button>}
        <span style={{ marginLeft: "auto", color: "var(--muted)", fontSize: 12.5 }}>{rows.length} shown</span>
      </div>

      <div className="panelcard">
        {rows.map((t, i) => (
          <Row key={`${t.endpoint}|${t.role}|${i}`} t={t} roles={scan.roles} onOpen={() => go("graph", t.endpoint)} />
        ))}
      </div>
    </div>
  );
}

function Row({ t, roles, onOpen }: { t: RuntimeTest; roles: string[]; onOpen: () => void }) {
  return (
    <div className="rt-row" onClick={onOpen} style={{ cursor: "pointer" }}>
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
  );
}

function rank(o: string): number {
  return { divergent: 4, fail: 3, pass: 2, blocked: 1, unverified: 0 }[o] ?? 0;
}

function fmtTime(iso: string): string {
  const d = new Date(iso);
  return isNaN(d.getTime()) ? iso : d.toLocaleString();
}
