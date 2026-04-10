"use client";

import type { SlotDiagnostic } from "@/lib/types";

type Props = {
  diagnostics: SlotDiagnostic[];
};

export default function DiagnosticPanel({ diagnostics }: Props) {
  if (diagnostics.length === 0) {
    return <p className="success">All slot checks passed.</p>;
  }

  return (
    <div>
      <h3>Slot Diagnostics</h3>
      <ul className="list" style={{ marginTop: "0.4rem" }}>
        {diagnostics.map((d, i) => (
          <li
            key={i}
            style={{
              padding: "0.4rem 0.6rem",
              borderLeft: `3px solid ${d.severity === "error" ? "var(--danger)" : "#d97706"}`,
              background: d.severity === "error" ? "#fef2f2" : "#fffbeb",
              borderRadius: "4px",
              fontSize: "0.9rem",
            }}
          >
            <strong style={{ color: d.severity === "error" ? "var(--danger)" : "#92400e" }}>
              {d.severity === "error" ? "Error" : "Warning"}
            </strong>
            {" — "}
            {d.message}
          </li>
        ))}
      </ul>
    </div>
  );
}
