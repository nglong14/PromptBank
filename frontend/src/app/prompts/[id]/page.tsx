"use client";

import { FormEvent, useCallback, useEffect, useState } from "react";
import { useParams } from "next/navigation";
import ProtectedPage from "@/components/ProtectedPage";
import AssetEditor from "@/components/AssetEditor";
import FrameworkSelector from "@/components/FrameworkSelector";
import TechniqueToggles from "@/components/TechniqueToggles";
import DiagnosticPanel from "@/components/DiagnosticPanel";
import QuestionWizard from "@/components/QuestionWizard";
import {
  compose,
  createPromptVersion,
  getPrompt,
  listFrameworks,
  listPromptVersions,
  listTechniques,
  llmNormalize,
  llmRefine,
  llmScore,
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
  QualityScore,
  RefineMessage,
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

function qualityColor(score: number): string {
  if (score >= 8) return "#16a34a";
  if (score >= 5) return "#d97706";
  return "#dc2626";
}

type BuildMode = "wizard" | "normalizing" | "editor";

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

  // Compose state
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

  // Wizard state
  const [buildMode, setBuildMode] = useState<BuildMode>("wizard");
  const [wizardError, setWizardError] = useState("");

  // Quality scoring state
  const [qualityScore, setQualityScore] = useState<QualityScore | null>(null);
  const [qualityLoading, setQualityLoading] = useState(false);

  // Refinement state
  const [refineHistory, setRefineHistory] = useState<RefineMessage[]>([]);
  const [refineFeedback, setRefineFeedback] = useState("");
  const [refineLoading, setRefineLoading] = useState(false);
  const [refineError, setRefineError] = useState("");
  const [showSavePrompt, setShowSavePrompt] = useState(false);
  const [copied, setCopied] = useState(false);

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

  async function onWizardComplete(answers: Record<string, string>) {
    const token = getToken();
    if (!token) return;

    setBuildMode("normalizing");
    setWizardError("");

    try {
      const res = await llmNormalize(token, answers);
      setAssets(res.assets);
      if (res.suggestedFrameworkId) {
        setFrameworkId(res.suggestedFrameworkId);
      }
      setBuildMode("editor");
    } catch (err) {
      setWizardError(err instanceof Error ? err.message : "Failed to interpret your answers. Please try again.");
      setBuildMode("wizard");
    }
  }

  function resetCompose() {
    setComposeResult(null);
    setDiagnostics([]);
    setQualityScore(null);
    setRefineHistory([]);
    setRefineError("");
    setRefineFeedback("");
    setShowSavePrompt(false);
    setCopied(false);
  }

  function switchToWizard() {
    setAssets(emptyAssets);
    setFieldReport(null);
    setFrameworkId("");
    setTechniqueIds([]);
    setWizardError("");
    resetCompose();
    setBuildMode("wizard");
  }

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

  function silentlyScore(output: string, token: string) {
    setQualityScore(null);
    setQualityLoading(true);
    llmScore(token, output)
      .then(setQualityScore)
      .catch(() => {})
      .finally(() => setQualityLoading(false));
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
    resetCompose();
    try {
      const res = await compose(token, { assets, frameworkId, techniqueIds });
      setComposeResult(res);
      setDiagnostics(res.diagnostics ?? []);
      silentlyScore(res.composedOutput, token);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Composition failed");
    } finally {
      setComposeLoading(false);
    }
  }

  async function onRefine() {
    const token = getToken();
    if (!token || !composeResult || !refineFeedback.trim()) return;

    setRefineLoading(true);
    setRefineError("");

    const currentFeedback = refineFeedback;

    try {
      const res = await llmRefine(token, {
        assets,
        composedOutput: composeResult.composedOutput,
        userFeedback: currentFeedback,
        history: refineHistory,
      });

      const userMsg: RefineMessage = { role: "user", content: currentFeedback };
      const agentMsg: RefineMessage = { role: "agent", content: res.explanation };
      setRefineHistory((h) => [...h, userMsg, agentMsg]);
      setRefineFeedback("");
      setAssets(res.updatedAssets);

      // Auto-recompose with updated assets
      const recomposeRes = await compose(token, {
        assets: res.updatedAssets,
        frameworkId,
        techniqueIds,
      });
      setComposeResult(recomposeRes);
      setDiagnostics(recomposeRes.diagnostics ?? []);
      silentlyScore(recomposeRes.composedOutput, token);
      setShowSavePrompt(true);
    } catch (err) {
      setRefineError(err instanceof Error ? err.message : "Refinement failed");
    } finally {
      setRefineLoading(false);
    }
  }

  async function onCopy() {
    if (!composeResult) return;
    try {
      await navigator.clipboard.writeText(composeResult.composedOutput);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // Clipboard API may not be available
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
      setAssets(emptyAssets);
      setFieldReport(null);
      setFrameworkId("");
      setTechniqueIds([]);
      resetCompose();
      setBuildMode("wizard");
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

            <div className="two-col">
              <article className="card">
                <div className="row" style={{ marginBottom: "0.8rem" }}>
                  <div>
                    <h2 style={{ margin: 0 }}>Compose New Version</h2>
                    <p className="muted" style={{ margin: "0.2rem 0 0", fontSize: "0.9rem" }}>
                      {buildMode === "editor"
                        ? "Review and edit your prompt assets, then compose."
                        : "Answer a few questions and we'll build the prompt for you."}
                    </p>
                  </div>
                  {buildMode === "editor" ? (
                    <button
                      type="button"
                      className="btn btn-secondary"
                      style={{ fontSize: "0.85rem", whiteSpace: "nowrap" }}
                      onClick={switchToWizard}
                    >
                      Start over with questions
                    </button>
                  ) : null}
                </div>

                {buildMode === "normalizing" ? (
                  <div
                    style={{
                      display: "flex",
                      flexDirection: "column",
                      alignItems: "center",
                      gap: "0.6rem",
                      padding: "2.5rem 1rem",
                      color: "var(--muted)",
                    }}
                  >
                    <div
                      style={{
                        width: "28px",
                        height: "28px",
                        border: "3px solid var(--border)",
                        borderTopColor: "var(--primary)",
                        borderRadius: "50%",
                        animation: "spin 0.8s linear infinite",
                      }}
                    />
                    <p style={{ margin: 0 }}>Interpreting your answers with AI...</p>
                    <style>{`@keyframes spin { to { transform: rotate(360deg); } }`}</style>
                  </div>
                ) : buildMode === "wizard" ? (
                  <QuestionWizard
                    onComplete={(answers) => void onWizardComplete(answers)}
                    onCancel={() => setBuildMode("editor")}
                    error={wizardError}
                  />
                ) : (
                  <form onSubmit={onCompose}>
                    {wizardError && (
                      <p className="error" style={{ marginBottom: "0.6rem" }}>
                        {wizardError}
                      </p>
                    )}

                    <AssetEditor assets={assets} fieldReport={fieldReport} onChange={setAssets} />

                    <div style={{ margin: "0.6rem 0" }}>
                      <button type="button" className="btn btn-secondary" onClick={() => void onNormalize()}>
                        Normalize &amp; check fields
                      </button>
                    </div>

                    <FrameworkSelector frameworks={frameworks} selectedId={frameworkId} onSelect={setFrameworkId} />
                    <TechniqueToggles techniques={techniques} selectedIds={techniqueIds} onToggle={setTechniqueIds} assets={assets} onAssetsChange={setAssets} />

                    <div style={{ marginTop: "0.8rem" }}>
                      <button className="btn btn-primary" type="submit" disabled={composeLoading || !frameworkId}>
                        {composeLoading ? "Composing..." : "Compose prompt"}
                      </button>
                    </div>

                    {diagnostics.length > 0 ? (
                      <div style={{ marginTop: "0.8rem" }}>
                        <DiagnosticPanel diagnostics={diagnostics} />
                      </div>
                    ) : null}

                    {composeResult ? (
                      <div style={{ marginTop: "0.8rem" }}>
                        <div className="row" style={{ marginBottom: "0.4rem" }}>
                          <h3 style={{ margin: 0 }}>Composed Output</h3>
                          <div style={{ display: "flex", alignItems: "center", gap: "0.5rem" }}>
                            {qualityLoading ? (
                              <span className="muted" style={{ fontSize: "0.85rem" }}>
                                Scoring...
                              </span>
                            ) : qualityScore ? (
                              <span
                                className="quality-badge"
                                style={{
                                  background: qualityColor(qualityScore.score),
                                }}
                              >
                                {qualityScore.score}/10
                              </span>
                            ) : null}
                            <button
                              type="button"
                              className="btn btn-secondary"
                              style={{ fontSize: "0.78rem", padding: "0.2rem 0.55rem" }}
                              onClick={() => void onCopy()}
                            >
                              {copied ? "Copied!" : "Copy"}
                            </button>
                          </div>
                        </div>

                        {qualityScore?.feedback && (
                          <p className="muted" style={{ fontSize: "0.85rem", margin: "0 0 0.6rem" }}>
                            {qualityScore.feedback}
                          </p>
                        )}

                        <pre className="composed-output mono">
                          {composeResult.composedOutput}
                        </pre>

                        {/* Refine with AI panel */}
                        <div className="card" style={{ marginTop: "0.8rem" }}>
                          <h3 style={{ margin: "0 0 0.3rem" }}>Refine with AI</h3>
                          <p className="muted" style={{ margin: "0 0 0.8rem", fontSize: "0.9rem" }}>
                            Describe what you&#39;d like to change. The AI will update the prompt and re-compose it for you.
                          </p>

                          {refineHistory.length > 0 && (
                            <div
                              style={{
                                borderTop: "1px solid var(--border)",
                                paddingTop: "0.6rem",
                                marginBottom: "0.8rem",
                                display: "grid",
                                gap: "0.5rem",
                              }}
                            >
                              {refineHistory.map((msg, i) => (
                                <div
                                  key={i}
                                  className="refine-msg"
                                >
                                  <span
                                    className={msg.role === "user" ? "refine-role refine-role-user" : "refine-role refine-role-agent"}
                                  >
                                    {msg.role === "user" ? "You" : "AI"}
                                  </span>
                                  <span style={{ fontSize: "0.9rem" }}>{msg.content}</span>
                                </div>
                              ))}
                            </div>
                          )}

                          {refineError && (
                            <p className="error" style={{ marginBottom: "0.5rem" }}>
                              {refineError}
                            </p>
                          )}

                          <textarea
                            className="textarea"
                            rows={2}
                            placeholder='e.g. "Make it sound less formal" or "Add more constraints about response length"'
                            value={refineFeedback}
                            onChange={(e) => setRefineFeedback(e.target.value)}
                            disabled={refineLoading}
                          />
                          <button
                            type="button"
                            className="btn btn-primary"
                            style={{ marginTop: "0.5rem" }}
                            onClick={() => void onRefine()}
                            disabled={refineLoading || !refineFeedback.trim()}
                          >
                            {refineLoading ? "Refining..." : "Refine prompt"}
                          </button>
                        </div>

                        {showSavePrompt ? (
                          <div className="save-prompt-bar">
                            <span style={{ fontWeight: 500, fontSize: "0.9rem" }}>
                              Refinement applied. Save as new version?
                            </span>
                            <div style={{ display: "flex", gap: "0.4rem" }}>
                              <button
                                type="button"
                                className="btn btn-primary"
                                style={{ fontSize: "0.85rem", padding: "0.3rem 0.7rem" }}
                                disabled={saveVersionLoading}
                                onClick={() => void onSaveVersion()}
                              >
                                {saveVersionLoading ? "Saving..." : "Save version"}
                              </button>
                              <button
                                type="button"
                                className="btn btn-secondary"
                                style={{ fontSize: "0.85rem", padding: "0.3rem 0.7rem" }}
                                onClick={() => setShowSavePrompt(false)}
                              >
                                Keep editing
                              </button>
                            </div>
                          </div>
                        ) : (
                          <button
                            type="button"
                            className="btn btn-primary"
                            style={{ marginTop: "0.8rem" }}
                            disabled={saveVersionLoading}
                            onClick={() => void onSaveVersion()}
                          >
                            {saveVersionLoading ? "Saving version..." : "Save as new version"}
                          </button>
                        )}
                      </div>
                    ) : null}
                  </form>
                )}
              </article>

              <article className="card versions-panel">
                <div className="row">
                  <h2>Versions</h2>
                  <button className="btn btn-secondary" type="button" onClick={() => void loadAll()}>
                    Refresh
                  </button>
                </div>
                {versions.length === 0 ? (
                  <p className="muted">No versions yet.</p>
                ) : (
                  <ul className="list">
                    {versions.map((version) => (
                      <li key={version.id} className="card version-card">
                        <div className="row" style={{ marginBottom: "0.3rem" }}>
                          <strong style={{ fontSize: "1rem" }}>v{version.versionNumber}</strong>
                          <span className="muted" style={{ fontSize: "0.78rem" }}>
                            {new Date(version.createdAt).toLocaleDateString()}
                          </span>
                        </div>
                        <div style={{ display: "flex", flexWrap: "wrap", gap: "0.3rem", marginBottom: "0.4rem" }}>
                          {version.frameworkId && (
                            <span className="tag-pill">{version.frameworkId}</span>
                          )}
                          {version.techniqueIds?.map((tid) => (
                            <span key={tid} className="tag-pill tag-pill-muted">{tid}</span>
                          ))}
                        </div>
                        {version.composedOutput ? (
                          <details>
                            <summary className="muted" style={{ cursor: "pointer", fontSize: "0.85rem" }}>
                              Show composed output
                            </summary>
                            <pre className="composed-output mono" style={{ marginTop: "0.3rem", maxHeight: "200px" }}>
                              {version.composedOutput}
                            </pre>
                          </details>
                        ) : (
                          <p className="muted" style={{ fontSize: "0.85rem", margin: 0 }}>No composed output</p>
                        )}
                        <details style={{ marginTop: "0.3rem" }}>
                          <summary className="muted" style={{ cursor: "pointer", fontSize: "0.78rem" }}>
                            ID
                          </summary>
                          <p className="mono" style={{ fontSize: "0.75rem", margin: "0.2rem 0 0", color: "var(--muted)" }}>
                            {version.id}
                          </p>
                        </details>
                      </li>
                    ))}
                  </ul>
                )}
              </article>
            </div>
          </>
        )}
      </section>
    </ProtectedPage>
  );
}
