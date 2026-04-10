"use client";

import type { Framework } from "@/lib/types";

type Props = {
  frameworks: Framework[];
  selectedId: string;
  onSelect: (id: string) => void;
};

export default function FrameworkSelector({ frameworks, selectedId, onSelect }: Props) {
  return (
    <div className="stack">
      <h3>Framework</h3>
      {frameworks.length === 0 ? (
        <p className="muted">Loading frameworks...</p>
      ) : (
        <div style={{ display: "grid", gap: "0.6rem", gridTemplateColumns: "repeat(auto-fit, minmax(250px, 1fr))" }}>
          {frameworks.map((fw) => {
            const isActive = fw.id === selectedId;
            return (
              <button
                key={fw.id}
                type="button"
                className="card"
                onClick={() => onSelect(fw.id)}
                style={{
                  textAlign: "left",
                  cursor: "pointer",
                  borderColor: isActive ? "var(--primary)" : undefined,
                  borderWidth: isActive ? "2px" : undefined,
                }}
              >
                <strong>{fw.name}</strong>
                <p className="muted" style={{ fontSize: "0.85rem" }}>{fw.description}</p>
                <p style={{ fontSize: "0.8rem", marginTop: "0.3rem" }}>
                  Slots:{" "}
                  {fw.slots.map((s) => (
                    <span key={s.name} style={{ marginRight: "0.4rem" }}>
                      {s.name}
                      {s.required ? "*" : ""}
                    </span>
                  ))}
                </p>
              </button>
            );
          })}
        </div>
      )}
    </div>
  );
}
