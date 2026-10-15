import { X, FileCode2 } from "lucide-react";
import type { Endpoint, Scan, TableInfo } from "../lib/types";
import {
  MethodBadge, RoleTag, SeverityBadge, EvidenceBadge, EnforcementBadge, ConfidenceBadge,
} from "./primitives";

export type Selection =
  | { kind: "endpoint"; id: string }
  | { kind: "table"; name: string }
  | null;

export function Inspector({
  selection, scan, onClose, onSelect,
}: {
  selection: Selection;
  scan: Scan;
  onClose: () => void;
  onSelect: (s: Selection) => void;
}) {
  if (!selection) return null;
  if (selection.kind === "endpoint") {
    const ep = scan.endpoints.find((e) => e.id === selection.id);
    if (!ep) return null;
    return <EndpointInspector ep={ep} scan={scan} onClose={onClose} />;
  }
  const t = scan.tables.find((x) => x.name === selection.name);
  if (!t) return null;
  return <TableInspector table={t} scan={scan} onClose={onClose} onSelect={onSelect} />;
}

function loc(file?: string, line?: number): string {
  if (!file) return "";
  const base = file.split("/").pop() ?? file;
  return line ? `${base}:${line}` : base;
}

function EndpointInspector({ ep, scan, onClose }: { ep: Endpoint; scan: Scan; onClose: () => void }) {
  return (
    <div className="inspector">
      <div className="ihead">
        <div className="title">
          <MethodBadge method={ep.method} />
          <span className="route mono">{ep.path}</span>
          <button className="iconbtn close" onClick={onClose} aria-label="Close"><X size={14} /></button>
        </div>
        <div className="kv" style={{ marginTop: 8 }}>
          <span className="k">Handler</span>
          <span className="v mono">{ep.handler || "—"}{ep.handlerFile ? `  (${loc(ep.handlerFile, ep.handlerLine)})` : ""}</span>
        </div>
      </div>

      <section className="isec">
        <h3>Authorization path</h3>
        <div className="trace">
          <Step label="request" name={`${ep.method} ${ep.path}`} />
          {ep.middleware.map((m, i) => (
            <Step key={`mw${i}`} label="middleware" name={m.name}
              note={m.args && m.args.length ? `requires role ${m.args.join(", ")}` : undefined} />
          ))}
          <Step label="handler" name={ep.handler || "handler"} />
          {ep.checks.map((c, i) => (
            <Step key={`ck${i}`} label="auth check" name={c.func}
              tone={c.enforced ? undefined : "warn"}
              note={c.enforced ? "enforced (dominates response)" : "does not gate the response"} />
          ))}
          {ep.tables.map((t, i) => (
            <Step key={`tb${i}`} label={t.operations.join(", ").toLowerCase() || "accesses"} name={t.table} mono />
          ))}
          {ep.unresolved && (
            <Step label="unresolved" name="dynamic SQL query" tone="warn" terminal
              note="query built at runtime — tables not recoverable" />
          )}
          {!ep.unresolved && ep.tables.length === 0 && (
            <Step label="data access" name="none observed" terminal />
          )}
        </div>
      </section>

      <section className="isec">
        <h3>Roles &amp; gates</h3>
        {ep.roles.length ? (
          <div className="cellwrap">{ep.roles.map((r) => <RoleTag key={r} role={r} roles={scan.roles} />)}</div>
        ) : (
          <div className="kv"><span className="v" style={{ color: "var(--muted)" }}>
            {ep.middleware.length ? "authenticated, no specific role required" : "no gate — public"}
          </span></div>
        )}
        {ep.checks.map((c, i) => (
          <div className="evline" key={i}>
            <EnforcementBadge enforcement={c.enforcement} />
            <span className="mono" style={{ fontSize: 12 }}>{c.func}</span>
            {c.file && <span className="loc">{loc(c.file, c.line)}</span>}
          </div>
        ))}
      </section>

      {ep.tables.length > 0 && (
        <section className="isec">
          <h3>Data access</h3>
          {ep.tables.map((t) => (
            <div className="evline" key={t.table}>
              <span className="chip">{t.table} <span className="op">{t.operations.join(",")}</span></span>
            </div>
          ))}
        </section>
      )}

      <section className="isec">
        <h3>Findings {ep.findings.length > 0 && <span style={{ color: "var(--muted)" }}>· {ep.findings.length}</span>}</h3>
        {ep.findings.length === 0 && <div style={{ color: "var(--muted)", fontSize: 12.5 }}>No findings on this endpoint.</div>}
        {ep.findings.map((f, i) => (
          <div className="findingcard" key={f.id || i}>
            <div className="cellwrap">
              <SeverityBadge severity={f.severity} />
              <ConfidenceBadge confidence={f.confidence} />
              <EvidenceBadge confidence={f.confidence} />
            </div>
            <div className="ftitle">{f.title}</div>
            <div className="fdetail">{f.detail}</div>
            {f.locations && f.locations[0] && (
              <div className="floc"><FileCode2 size={11} style={{ verticalAlign: "-1px", marginRight: 4 }} />
                {loc(f.locations[0].file, f.locations[0].start_line)}</div>
            )}
          </div>
        ))}
      </section>
    </div>
  );
}

function TableInspector({
  table, scan, onClose, onSelect,
}: {
  table: TableInfo; scan: Scan; onClose: () => void; onSelect: (s: Selection) => void;
}) {
  return (
    <div className="inspector">
      <div className="ihead">
        <div className="title">
          <span className="route mono">{table.name}</span>
          <button className="iconbtn close" onClick={onClose} aria-label="Close"><X size={14} /></button>
        </div>
        <div className="kv" style={{ marginTop: 8 }}>
          <span className="k">Operations</span>
          <span className="v mono">{table.operations.join(", ") || "—"}</span>
        </div>
      </div>
      <section className="isec">
        <h3>Reached by {table.endpoints.length} endpoint{table.endpoints.length === 1 ? "" : "s"}</h3>
        {table.endpoints.map((id) => {
          const ep = scan.endpoints.find((e) => e.id === id);
          return (
            <div className="evline" key={id} style={{ cursor: "pointer" }} onClick={() => onSelect({ kind: "endpoint", id })}>
              {ep && <MethodBadge method={ep.method} />}
              <span className="mono" style={{ fontSize: 12 }}>{ep?.path ?? id}</span>
            </div>
          );
        })}
      </section>
    </div>
  );
}

function Step({
  label, name, note, tone, mono, terminal,
}: {
  label: string; name: string; note?: string; tone?: "warn" | "bad"; mono?: boolean; terminal?: boolean;
}) {
  return (
    <div className={`tstep ${tone ?? ""}`}>
      <div className="rail">
        <span className="node" />
        {!terminal && <span className="line" />}
      </div>
      <div className="body">
        <div className="label">{label}</div>
        <div className={`name ${mono ? "mono" : ""}`}>{name}</div>
        {note && <div style={{ fontSize: 11.5, color: tone ? "var(--sev-medium)" : "var(--muted)" }}>{note}</div>}
      </div>
    </div>
  );
}
