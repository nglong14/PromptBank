"use client";

import type { Technique } from "@/lib/types";

type Props = {
  techniques: Technique[];
  selectedIds: string[];
  onToggle: (ids: string[]) => void;
};

export default function TechniqueToggles({ techniques, selectedIds, onToggle }: Props) {
  function toggle(id: string) {
    if (selectedIds.includes(id)) {
      onToggle(selectedIds.filter((t) => t !== id));
    } else {
      onToggle([...selectedIds, id]);
    }
  }

  return (
    <div>
      <h3>Techniques</h3>
      {techniques.length === 0 ? (
        <p className="muted">Loading techniques...</p>
      ) : (
        <div style={{ display: "flex", flexWrap: "wrap", gap: "0.6rem", marginTop: "0.4rem" }}>
          {techniques.map((t) => {
            const active = selectedIds.includes(t.id);
            return (
              <label
                key={t.id}
                className="card"
                style={{
                  cursor: "pointer",
                  display: "flex",
                  alignItems: "flex-start",
                  gap: "0.5rem",
                  borderColor: active ? "var(--primary)" : undefined,
                  borderWidth: active ? "2px" : undefined,
                  padding: "0.7rem",
                }}
              >
                <input
                  type="checkbox"
                  checked={active}
                  onChange={() => toggle(t.id)}
                  style={{ marginTop: "0.25rem" }}
                />
                <div>
                  <strong>{t.name}</strong>
                  <p className="muted" style={{ fontSize: "0.8rem" }}>{t.description}</p>
                </div>
              </label>
            );
          })}
        </div>
      )}
    </div>
  );
}
