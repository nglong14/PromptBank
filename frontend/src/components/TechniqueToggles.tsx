"use client";

import { useState } from "react";
import { llmSuggestTechnique } from "@/lib/api";
import { getToken } from "@/lib/auth";
import type { Assets, Technique } from "@/lib/types";

const techniqueAssetHints: Record<string, { field: keyof Assets; label: string }> = {
  "few-shot": { field: "examples", label: "No examples set — add some or use Suggest." },
  "role-priming": { field: "persona", label: "No persona set — write one or use Suggest." },
  "constraints-first": { field: "constraints", label: "No constraints set — add some or use Suggest." },
};

type Props = {
  techniques: Technique[];
  selectedIds: string[];
  onToggle: (ids: string[]) => void;
  assets: Assets;
  onAssetsChange: (assets: Assets) => void;
};

export default function TechniqueToggles({ techniques, selectedIds, onToggle, assets, onAssetsChange }: Props) {
  const [suggestingId, setSuggestingId] = useState<string | null>(null);
  const [suggestError, setSuggestError] = useState("");

  function toggle(id: string) {
    if (selectedIds.includes(id)) {
      onToggle(selectedIds.filter((t) => t !== id));
    } else {
      onToggle([...selectedIds, id]);
    }
  }

  async function onSuggest(techniqueId: string) {
    const token = getToken();
    if (!token) return;

    setSuggestingId(techniqueId);
    setSuggestError("");
    try {
      const res = await llmSuggestTechnique(token, techniqueId, assets);
      onAssetsChange(res.assets);
    } catch (err) {
      setSuggestError(err instanceof Error ? err.message : "Suggestion failed");
    } finally {
      setSuggestingId(null);
    }
  }

  function needsHint(techniqueId: string): boolean {
    const hint = techniqueAssetHints[techniqueId];
    if (!hint) return false;
    const val = assets[hint.field];
    if (Array.isArray(val)) return val.length === 0;
    return !val;
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
            const showHint = active && needsHint(t.id);
            const hint = techniqueAssetHints[t.id];
            const isSuggesting = suggestingId === t.id;
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
                  minWidth: "220px",
                  flex: "1 1 220px",
                }}
              >
                <input
                  type="checkbox"
                  checked={active}
                  onChange={() => toggle(t.id)}
                  style={{ marginTop: "0.25rem" }}
                />
                <div style={{ flex: 1 }}>
                  <strong>{t.name}</strong>
                  <p className="muted" style={{ fontSize: "0.8rem", margin: "0.15rem 0" }}>{t.description}</p>
                  {active && hint && (
                    <div style={{ marginTop: "0.4rem", display: "flex", flexWrap: "wrap", gap: "0.3rem", alignItems: "center" }}>
                      {showHint && (
                        <span style={{ fontSize: "0.78rem", color: "var(--warning)" }}>
                          {hint.label}
                        </span>
                      )}
                      <button
                        type="button"
                        className="btn btn-secondary"
                        style={{ fontSize: "0.75rem", padding: "0.2rem 0.5rem" }}
                        disabled={isSuggesting || suggestingId !== null}
                        onClick={(e) => {
                          e.preventDefault();
                          e.stopPropagation();
                          void onSuggest(t.id);
                        }}
                      >
                        {isSuggesting ? "Suggesting..." : "Suggest with AI"}
                      </button>
                    </div>
                  )}
                </div>
              </label>
            );
          })}
        </div>
      )}
      {suggestError && (
        <p className="error" style={{ marginTop: "0.4rem", fontSize: "0.85rem" }}>{suggestError}</p>
      )}
    </div>
  );
}
