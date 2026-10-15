import { roleColorVar } from "../lib/api";
import type { Enforcement } from "../lib/types";

export function MethodBadge({ method }: { method: string }) {
  const m = method.toLowerCase();
  const known = ["get", "post", "put", "patch", "delete"].includes(m);
  return <span className={`method m-${known ? m : "other"}`}>{method}</span>;
}

export function RoleTag({ role, roles }: { role: string; roles: string[] }) {
  return (
    <span className="roletag">
      <span className="dot" style={{ background: roleColorVar(role, roles) }} />
      {role}
    </span>
  );
}

export function SeverityBadge({ severity }: { severity: string }) {
  return <span className={`badge sev-${severity}`}>{severity}</span>;
}

export function ConfidenceBadge({ confidence }: { confidence: string }) {
  return <span className="badge conf">{confidence}</span>;
}

const enforcementLabel: Record<Enforcement, string> = {
  enforced: "enforced",
  discarded: "result discarded",
  non_dominating: "non-dominating",
  result_not_gating: "not gating",
};

export function EnforcementBadge({ enforcement }: { enforcement: Enforcement }) {
  const ok = enforcement === "enforced";
  return (
    <span className={`badge ${ok ? "ev-code" : "sev-medium"}`}>
      {enforcementLabel[enforcement] ?? enforcement}
    </span>
  );
}

// evidenceForConfidence maps a finding's confidence to the evidence category it
// rests on, so the UI speaks the same evidence language everywhere.
export function evidenceLabel(confidence: string): { cls: string; text: string } {
  switch (confidence) {
    case "proven":
      return { cls: "ev-code", text: "code confirmed" };
    case "likely":
      return { cls: "ev-inferred", text: "inferred" };
    default:
      return { cls: "ev-unverified", text: "unverified" };
  }
}

export function EvidenceBadge({ confidence }: { confidence: string }) {
  const e = evidenceLabel(confidence);
  return <span className={`badge ev ${e.cls}`}>{e.text}</span>;
}
