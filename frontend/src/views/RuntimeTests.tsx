import { useEffect, useMemo, useState } from "react";
import { ShieldAlert, X, Play, Plus, Trash2, Search, ChevronRight, ChevronDown } from "lucide-react";
import type { Scan, RuntimeTest, RuntimeOutcome } from "../lib/types";
import type { View } from "../components/Sidebar";
import { MethodBadge, RoleTag } from "../components/primitives";
import { runTests } from "../lib/api";

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
  const [results, setResults] = useState<RuntimeTest[]>(scan.runtimeTests);
  const [testedAt, setTestedAt] = useState<string | undefined>(scan.runtimeTestedAt);

  // The App fetches /api/scan once, so its `scan` prop is frozen at page load.
  // Re-fetch on mount to recover a run executed earlier this session (the server
  // keeps it in memory), so results don't vanish when navigating away and back.
  useEffect(() => {
    let live = true;
    fetch("api/scan")
      .then((r) => (r.ok ? r.json() : null))
      .then((d: Scan | null) => {
        if (live && d && d.runtimeTests.length) { setResults(d.runtimeTests); setTestedAt(d.runtimeTestedAt); }
      })
      .catch(() => { /* keep whatever the cached prop gave us */ });
    return () => { live = false; };
  }, []);

  const counts = useMemo(() => {
    const c: Record<string, number> = {};
    for (const t of results) c[t.outcome] = (c[t.outcome] ?? 0) + 1;
    return c;
  }, [results]);

  const divergences = results.filter((t) => t.outcome === "divergent");
  const rows = [...results]
    .filter((t) => !outcome || t.outcome === outcome)
    .sort((a, b) => rank(b.outcome) - rank(a.outcome) || a.path.localeCompare(b.path) || a.role.localeCompare(b.role));

  return (
    <div className="page">
      <div className="pagehead">
        <h1>Runtime verification</h1>
        <p>
          Exercise the running application as each identity and compare what it does with what the scan
          predicted. Recorded apart from the static findings and never merged with them
          {testedAt ? ` · last run ${fmtTime(testedAt)}` : ""}.
        </p>
      </div>

      <RunForm scan={scan} onDone={(r) => { setResults(r); setTestedAt(new Date().toISOString()); }} />

      {results.length === 0 ? (
        <p style={{ color: "var(--muted)", fontSize: 13 }}>
          No runtime test has run yet. Fill in the base URL and a token for each identity above, then Run.
        </p>
      ) : (
        <>
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
        </>
      )}
    </div>
  );
}

// RunForm collects the base URL, an editable identity list (seeded from the
// scan's roles) and the per-endpoint opt-in for non-GET methods, then asks the
// server to run the test. Tokens are typed here and sent to localhost only.
function RunForm({ scan, onDone }: { scan: Scan; onDone: (r: RuntimeTest[]) => void }) {
  const [baseUrl, setBaseUrl] = useState("http://localhost:8080");
  const [ids, setIds] = useState<{ role: string; token: string; on: boolean }[]>(
    scan.roles.length
      ? [...scan.roles.map((role) => ({ role, token: "", on: false })), { role: "anonymous", token: "", on: false }]
      : [{ role: "", token: "", on: false }],
  );
  // The endpoints to run. GET is selected by default (read-only, safe); non-GET
  // starts off and is an explicit opt-in, because it can mutate or delete data.
  const [allow, setAllow] = useState<Set<string>>(() => {
    const s = new Set<string>();
    for (const e of scan.endpoints) {
      if (e.method.toUpperCase() === "GET") s.add(`GET ${e.path}`);
    }
    return s;
  });
  const [allowQuery, setAllowQuery] = useState("");
  const [methodFilter, setMethodFilter] = useState<Set<string>>(new Set());
  const [running, setRunning] = useState(false);
  const [error, setError] = useState("");

  // Every distinct endpoint, as "METHOD /path".
  const allEps = useMemo(() => {
    const seen = new Set<string>();
    for (const e of scan.endpoints) seen.add(`${e.method.toUpperCase()} ${e.path}`);
    return [...seen].sort();
  }, [scan.endpoints]);

  // The methods present, with a count each, for the filter chips.
  const methodCounts = useMemo(() => {
    const c: Record<string, number> = {};
    for (const k of allEps) {
      const m = k.split(" ", 1)[0];
      c[m] = (c[m] ?? 0) + 1;
    }
    return c;
  }, [allEps]);
  const methods = useMemo(() => Object.keys(methodCounts).sort(), [methodCounts]);

  const shownEps = useMemo(() => {
    const q = allowQuery.trim().toLowerCase();
    return allEps.filter(
      (k) =>
        (methodFilter.size === 0 || methodFilter.has(k.split(" ", 1)[0])) &&
        (!q || k.toLowerCase().includes(q)),
    );
  }, [allEps, allowQuery, methodFilter]);

  const allShownSelected = shownEps.length > 0 && shownEps.every((k) => allow.has(k));
  const toggleAllShown = () =>
    setAllow((s) => {
      const next = new Set(s);
      if (allShownSelected) shownEps.forEach((k) => next.delete(k));
      else shownEps.forEach((k) => next.add(k));
      return next;
    });
  const toggleMethod = (m: string) =>
    setMethodFilter((s) => {
      const next = new Set(s);
      if (next.has(m)) next.delete(m);
      else next.add(m);
      return next;
    });

  const setId = (i: number, patch: Partial<{ role: string; token: string; on: boolean }>) =>
    setIds((xs) => xs.map((x, j) => (j === i ? { ...x, ...patch } : x)));

  const run = async () => {
    setError("");
    setRunning(true);
    try {
      // Only ticked identities with a role and token run; the role each is
      // predicted-allowed for is decided per endpoint by the scan's gates.
      // Token may be empty: an identity with no token runs unauthenticated,
      // to catch endpoints reachable without any credential.
      const identities = ids
        .filter((x) => x.on && x.role.trim())
        .map(({ role, token }) => ({ role, token }));
      const r = await runTests({ baseUrl: baseUrl.trim(), identities, endpoints: [...allow] });
      onDone(r);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setRunning(false);
    }
  };

  return (
    <div className="rt-form panelcard">
      <label className="rt-field">
        <span>Base URL</span>
        <input value={baseUrl} onChange={(e) => setBaseUrl(e.target.value)} placeholder="http://localhost:8080" />
      </label>

      <div className="rt-field">
        <span>Identities <em>(role matched against the scan's gates; token is a bearer credential, never stored — leave it empty to run unauthenticated)</em></span>
        {ids.map((id, i) => (
          <div key={i} className={`rt-idrow ${id.on ? "" : "off"}`}>
            <input type="checkbox" title="run as this identity" checked={id.on} onChange={(e) => setId(i, { on: e.target.checked })} />
            <input className="rt-role" value={id.role} placeholder="role" onChange={(e) => setId(i, { role: e.target.value })} />
            <input className="rt-token" type="password" value={id.token} placeholder="token (empty = unauthenticated)" onChange={(e) => setId(i, { token: e.target.value })} />
            <button className="btn ghost" title="remove" onClick={() => setIds((xs) => xs.filter((_, j) => j !== i))}><Trash2 size={13} /></button>
          </div>
        ))}
        <button className="btn ghost" style={{ alignSelf: "start" }} onClick={() => setIds((xs) => [...xs, { role: "", token: "", on: true }])}>
          <Plus size={13} /> Add identity
        </button>
      </div>

      {allEps.length > 0 && (
        <div className="rt-field">
          <span>Endpoints to run <em>(GET is on by default — read-only and safe; non-GET is off, ticking one opts it in and it can mutate or delete data in the running app)</em></span>
          <div className="rt-allowmethods">
            {methods.map((m) => (
              <button key={m} className={`rolebtn ${methodFilter.has(m) ? "active" : ""}`} onClick={() => toggleMethod(m)}>
                {m} <span className="rt-mcount">{methodCounts[m]}</span>
              </button>
            ))}
            {methodFilter.size > 0 && <button className="btn ghost" onClick={() => setMethodFilter(new Set())}>All methods</button>}
          </div>
          <div className="rt-allowhead">
            <div className="search">
              <Search size={13} />
              <input value={allowQuery} onChange={(e) => setAllowQuery(e.target.value)} placeholder="filter by path…" />
            </div>
            <button className="btn" onClick={toggleAllShown} disabled={shownEps.length === 0}>
              {allShownSelected ? "Deselect" : "Select"} {shownEps.length} shown
            </button>
            <span className="rt-selcount">{allow.size} selected</span>
            {allow.size > 0 && <button className="btn ghost" onClick={() => setAllow(new Set())}><X size={13} /> Clear</button>}
          </div>
          <div className="rt-allowbox">
            {shownEps.map((key) => (
              <label key={key} className={`rt-check ${key.startsWith("GET ") ? "" : "mut"}`}>
                <input
                  type="checkbox"
                  checked={allow.has(key)}
                  onChange={(e) =>
                    setAllow((s) => {
                      const next = new Set(s);
                      if (e.target.checked) next.add(key);
                      else next.delete(key);
                      return next;
                    })
                  }
                />
                <span className="route">{key}</span>
              </label>
            ))}
            {shownEps.length === 0 && <div className="rt-allowempty">no endpoints match "{allowQuery}"</div>}
          </div>
        </div>
      )}

      <div className="rt-run">
        <button className="btn primary" onClick={run} disabled={running}>
          <Play size={13} /> {running ? "Running…" : "Run runtime test"}
        </button>
        {error && <span className="rt-err">{error}</span>}
      </div>
    </div>
  );
}

function Row({ t, roles, onOpen }: { t: RuntimeTest; roles: string[]; onOpen: () => void }) {
  const [open, setOpen] = useState(false);
  return (
    <div className="rt-rowwrap">
      <div className="rt-row" onClick={onOpen} style={{ cursor: "pointer" }}>
        <MethodBadge method={t.method} />
        <div>
          <div className="rt-path">{t.path}</div>
          <div className="rt-meta">
            <RoleTag role={t.role} roles={roles} />
            <span className="badge conf">predicted {t.expectation}</span>
            {t.status ? <span className="badge conf">HTTP {t.status}</span> : null}
            {t.body ? (
              <button
                className="btn ghost rt-bodybtn"
                onClick={(e) => { e.stopPropagation(); setOpen((o) => !o); }}
              >
                {open ? <ChevronDown size={12} /> : <ChevronRight size={12} />} response
              </button>
            ) : null}
            <span className="rt-detail">{t.detail}</span>
          </div>
        </div>
        <span className={`badge ${CLS[t.outcome]}`}>{t.outcome}</span>
      </div>
      {open && t.body && <pre className="rt-body">{prettyBody(t.body)}</pre>}
    </div>
  );
}

// prettyBody indents JSON for readability; non-JSON (or truncated) bodies are
// shown verbatim.
function prettyBody(body: string): string {
  try {
    return JSON.stringify(JSON.parse(body), null, 2);
  } catch {
    return body;
  }
}

function rank(o: string): number {
  return { divergent: 4, fail: 3, pass: 2, blocked: 1, unverified: 0 }[o] ?? 0;
}

function fmtTime(iso: string): string {
  const d = new Date(iso);
  return isNaN(d.getTime()) ? iso : d.toLocaleString();
}
