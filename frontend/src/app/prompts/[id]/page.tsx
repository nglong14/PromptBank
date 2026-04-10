"use client";

import { FormEvent, useCallback, useEffect, useState } from "react";
import { useParams } from "next/navigation";
import ProtectedPage from "@/components/ProtectedPage";
import AssetEditor from "@/components/AssetEditor";
import FrameworkSelector from "@/components/FrameworkSelector";
import TechniqueToggles from "@/components/TechniqueToggles";
import DiagnosticPanel from "@/components/DiagnosticPanel";
import {
  compose,
  createPromptVersion,
  getPrompt,
  listFrameworks,
  listPromptVersions,
  listTechniques,
  normalizeAssets,
  updatePrompt,
} from "@/lib/api";
import { getToken } from "@/lib/auth";
import type {
  Assets,
  ComposeResponse,
  FieldQuality,
  Framework,
  Prompt,
  PromptVersion,
  SlotDiagnostic,
  Technique,
} from "@/lib/types";

function parseCsv(input: string): string[] {
  return input
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
}

const emptyAssets: Assets = {
  persona: "",
  context: "",
  tone: "",
  constraints: "",
  examples: [],
  goal: "",
};

export default function PromptDetailPage() {
  const params = useParams<{ id: string }>();
  const promptId = params.id;

  const [prompt, setPrompt] = useState<Prompt | null>(null);
  const [versions, setVersions] = useState<PromptVersion[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const [title, setTitle] = useState("");
  const [status, setStatus] = useState("draft");
  const [category, setCategory] = useState("");
  const [tagsInput, setTagsInput] = useState("");
  const [updateLoading, setUpdateLoading] = useState(false);

  const [assets, setAssets] = useState<Assets>(emptyAssets);
  const [fieldReport, setFieldReport] = useState<Record<string, FieldQuality> | null>(null);
  const [frameworkId, setFrameworkId] = useState("");
  const [techniqueIds, setTechniqueIds] = useState<string[]>([]);
  const [diagnostics, setDiagnostics] = useState<SlotDiagnostic[]>([]);
  const [composeResult, setComposeResult] = useState<ComposeResponse | null>(null);
  const [composeLoading, setComposeLoading] = useState(false);
  const [saveVersionLoading, setSaveVersionLoading] = useState(false);

  const [frameworks, setFrameworks] = useState<Framework[]>([]);
  const [techniques, setTechniques] = useState<Technique[]>([]);

  const loadAll = useCallback(async () => {
    const token = getToken();
    if (!token) return;

    setLoading(true);
    setError("");
    try {
      const [promptRes, versionsRes, fwRes, techRes] = await Promise.all([
        getPrompt(token, promptId),
        listPromptVersions(token, promptId),
        listFrameworks(token),
        listTechniques(token),
      ]);

      setPrompt(promptRes);
      setVersions(versionsRes.items);
      setTitle(promptRes.title);
      setStatus(promptRes.status);
      setCategory(promptRes.category);
      setTagsInput(promptRes.tags.join(", "));
      setFrameworks(fwRes.items);
      setTechniques(techRes.items);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to load prompt details";
      setError(message);
    } finally {
      setLoading(false);
    }
  }, [promptId]);

  useEffect(() => {
    void loadAll();
  }, [loadAll]);

  async function onNormalize() {
    const token = getToken();
    if (!token) return;
    try {
      const res = await normalizeAssets(token, assets);
      setAssets(res.assets);
      setFieldReport(res.fieldReport);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Normalize failed");
    }
  }

  async function onCompose(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const token = getToken();
    if (!token) return;
    if (!frameworkId) {
      setError("Select a framework before composing.");
      return;
    }

    setComposeLoading(true);
    setError("");
    setComposeResult(null);
    setDiagnostics([]);
    try {
      const res = await compose(token, { assets, frameworkId, techniqueIds });
      setComposeResult(res);
      setDiagnostics(res.diagnostics ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Composition failed");
    } finally {
      setComposeLoading(false);
    }
  }

  async function onSaveVersion() {
    if (!composeResult) return;
    const token = getToken();
    if (!token) return;

    setSaveVersionLoading(true);
    setError("");
    try {
      await createPromptVersion(token, promptId, {
        assets,
        frameworkId: composeResult.frameworkId,
        techniqueIds: composeResult.techniqueIds,
        composedOutput: composeResult.composedOutput,
      });
      setComposeResult(null);
      setDiagnostics([]);
      setAssets(emptyAssets);
      setFieldReport(null);
      setFrameworkId("");
      setTechniqueIds([]);
      await loadAll();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to save version");
    } finally {
      setSaveVersionLoading(false);
    }
  }

  async function onUpdatePrompt(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const token = getToken();
    if (!token) return;
    setUpdateLoading(true);
    setError("");
    try {
      const updated = await updatePrompt(token, promptId, {
        title,
        status,
        category,
        tags: parseCsv(tagsInput),
      });
      setPrompt(updated);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to update prompt");
    } finally {
      setUpdateLoading(false);
    }
  }

  return (
    <ProtectedPage>
      <section className="stack">
        <div>
          <h1 className="page-title">Prompt Detail</h1>
          <p className="subtitle mono">{promptId}</p>
        </div>

        {error ? <p className="error">{error}</p> : null}
        {loading ? (
          <article className="card">Loading prompt details...</article>
        ) : (
          <>
            <article className="card">
              <h2>Prompt Metadata</h2>
              {prompt ? (
                <p className="muted">
                  Created {new Date(prompt.createdAt).toLocaleString()} | Updated{" "}
                  {new Date(prompt.updatedAt).toLocaleString()}
                </p>
              ) : null}
              <form onSubmit={onUpdatePrompt}>
                <label className="field">
                  Title
                  <input className="input" required value={title} onChange={(e) => setTitle(e.target.value)} />
                </label>
                <label className="field">
                  Status
                  <input className="input" value={status} onChange={(e) => setStatus(e.target.value)} />
                </label>
                <label className="field">
                  Category
                  <input className="input" value={category} onChange={(e) => setCategory(e.target.value)} />
                </label>
                <label className="field">
                  Tags (comma separated)
                  <input className="input" value={tagsInput} onChange={(e) => setTagsInput(e.target.value)} />
                </label>
                <button className="btn btn-primary" type="submit" disabled={updateLoading}>
                  {updateLoading ? "Saving..." : "Update prompt"}
                </button>
              </form>
            </article>

            <article className="card">
              <h2>Compose New Version</h2>
              <p className="muted">Fill assets, pick a framework and techniques, then compose.</p>

              <form onSubmit={onCompose}>
                <AssetEditor assets={assets} fieldReport={fieldReport} onChange={setAssets} />

                <div style={{ margin: "0.6rem 0" }}>
                  <button type="button" className="btn btn-secondary" onClick={() => void onNormalize()}>
                    Normalize &amp; check fields
                  </button>
                </div>

                <FrameworkSelector frameworks={frameworks} selectedId={frameworkId} onSelect={setFrameworkId} />
                <TechniqueToggles techniques={techniques} selectedIds={techniqueIds} onToggle={setTechniqueIds} />

                <div style={{ marginTop: "0.8rem" }}>
                  <button className="btn btn-primary" type="submit" disabled={composeLoading || !frameworkId}>
                    {composeLoading ? "Composing..." : "Compose prompt"}
                  </button>
                </div>
              </form>

              {diagnostics.length > 0 ? (
                <div style={{ marginTop: "0.8rem" }}>
                  <DiagnosticPanel diagnostics={diagnostics} />
                </div>
              ) : null}

              {composeResult ? (
                <div style={{ marginTop: "0.8rem" }}>
                  <h3>Composed Output</h3>
                  <pre
                    className="card mono"
                    style={{ whiteSpace: "pre-wrap", fontSize: "0.85rem", maxHeight: "400px", overflow: "auto" }}
                  >
                    {composeResult.composedOutput}
                  </pre>
                  <button
                    type="button"
                    className="btn btn-primary"
                    style={{ marginTop: "0.5rem" }}
                    disabled={saveVersionLoading}
                    onClick={() => void onSaveVersion()}
                  >
                    {saveVersionLoading ? "Saving version..." : "Save as new version"}
                  </button>
                </div>
              ) : null}
            </article>

            <article className="card">
              <div className="row">
                <h2>Versions</h2>
                <button className="btn btn-secondary" type="button" onClick={() => void loadAll()}>
                  Refresh
                </button>
              </div>
              {versions.length === 0 ? (
                <p>No versions yet.</p>
              ) : (
                <ul className="list">
                  {versions.map((version) => (
                    <li key={version.id} className="card">
                      <p>
                        <strong>Version #{version.versionNumber}</strong>
                      </p>
                      <p className="mono" style={{ fontSize: "0.8rem" }}>
                        ID: {version.id}
                      </p>
                      <p className="muted">Framework: {version.frameworkId || "-"}</p>
                      <p className="muted">
                        Techniques: {version.techniqueIds?.length > 0 ? version.techniqueIds.join(", ") : "none"}
                      </p>
                      {version.composedOutput ? (
                        <details>
                          <summary className="muted" style={{ cursor: "pointer" }}>
                            Show composed output
                          </summary>
                          <pre style={{ whiteSpace: "pre-wrap", fontSize: "0.85rem", marginTop: "0.3rem" }}>
                            {version.composedOutput}
                          </pre>
                        </details>
                      ) : (
                        <p className="muted">No composed output</p>
                      )}
                    </li>
                  ))}
                </ul>
              )}
            </article>
          </>
        )}
      </section>
    </ProtectedPage>
  );
}
