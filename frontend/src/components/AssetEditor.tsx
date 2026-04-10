"use client";

import type { Assets, AssetExample, FieldQuality } from "@/lib/types";

type Props = {
  assets: Assets;
  fieldReport: Record<string, FieldQuality> | null;
  onChange: (assets: Assets) => void;
};

const ASSET_FIELDS: { key: keyof Omit<Assets, "examples">; label: string; rows?: number }[] = [
  { key: "persona", label: "Persona", rows: 2 },
  { key: "context", label: "Context", rows: 3 },
  { key: "tone", label: "Tone" },
  { key: "constraints", label: "Constraints", rows: 3 },
  { key: "goal", label: "Goal", rows: 2 },
];

function qualityBadge(quality: FieldQuality | undefined) {
  if (!quality) return null;
  const colors: Record<FieldQuality, string> = {
    empty: "var(--danger)",
    weak: "#d97706",
    complete: "#16a34a",
  };
  return (
    <span style={{ color: colors[quality], fontSize: "0.8rem", marginLeft: "0.5rem" }}>
      {quality}
    </span>
  );
}

export default function AssetEditor({ assets, fieldReport, onChange }: Props) {
  function updateField(key: keyof Omit<Assets, "examples">, value: string) {
    onChange({ ...assets, [key]: value });
  }

  function updateExample(index: number, field: keyof AssetExample, value: string) {
    const updated = [...assets.examples];
    updated[index] = { ...updated[index], [field]: value };
    onChange({ ...assets, examples: updated });
  }

  function addExample() {
    onChange({ ...assets, examples: [...assets.examples, { input: "", output: "" }] });
  }

  function removeExample(index: number) {
    onChange({ ...assets, examples: assets.examples.filter((_, i) => i !== index) });
  }

  return (
    <div className="stack">
      {ASSET_FIELDS.map(({ key, label, rows }) => {
        const value = assets[key];
        return (
          <label className="field" key={key}>
            <span>
              {label}
              {qualityBadge(fieldReport?.[key])}
              <span className="muted" style={{ float: "right", fontSize: "0.8rem" }}>
                {value.length} chars
              </span>
            </span>
            {rows && rows > 1 ? (
              <textarea
                className="textarea"
                rows={rows}
                value={value}
                onChange={(e) => updateField(key, e.target.value)}
              />
            ) : (
              <input
                className="input"
                value={value}
                onChange={(e) => updateField(key, e.target.value)}
              />
            )}
          </label>
        );
      })}

      <div>
        <span>
          Examples
          {qualityBadge(fieldReport?.examples)}
        </span>
        {assets.examples.map((ex, i) => (
          <div key={i} className="card" style={{ marginTop: "0.5rem" }}>
            <div className="row">
              <strong>Example {i + 1}</strong>
              <button type="button" className="btn btn-secondary" onClick={() => removeExample(i)}>
                Remove
              </button>
            </div>
            <label className="field">
              Input
              <input
                className="input"
                value={ex.input}
                onChange={(e) => updateExample(i, "input", e.target.value)}
              />
            </label>
            <label className="field">
              Output
              <input
                className="input"
                value={ex.output}
                onChange={(e) => updateExample(i, "output", e.target.value)}
              />
            </label>
          </div>
        ))}
        <button
          type="button"
          className="btn btn-secondary"
          style={{ marginTop: "0.5rem" }}
          onClick={addExample}
        >
          + Add example
        </button>
      </div>
    </div>
  );
}
