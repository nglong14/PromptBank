"use client";

import { FormEvent, useCallback, useEffect, useState } from "react";
import { useParams } from "next/navigation";
import ProtectedPage from "@/components/ProtectedPage";
import { createPromptVersion, getPrompt, listPromptVersions, updatePrompt } from "@/lib/api";
import { getToken } from "@/lib/auth";
import type { Prompt, PromptVersion } from "@/lib/types";

function parseCsv(input: string): string[] {
  return input
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
}

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

  const [assetsInput, setAssetsInput] = useState("{}");
  const [frameworkId, setFrameworkId] = useState("");
  const [techniqueIdsInput, setTechniqueIdsInput] = useState("");
  const [composedOutput, setComposedOutput] = useState("");
  const [createVersionLoading, setCreateVersionLoading] = useState(false);

  const loadAll = useCallback(async () => {
    const token = getToken();
    if (!token) {
      return;
    }

    setLoading(true);
    setError("");
    try {
      const [promptResponse, versionsResponse] = await Promise.all([
        getPrompt(token, promptId),
        listPromptVersions(token, promptId),
      ]);

      setPrompt(promptResponse);
      setVersions(versionsResponse.items);
      setTitle(promptResponse.title);
      setStatus(promptResponse.status);
      setCategory(promptResponse.category);
      setTagsInput(promptResponse.tags.join(", "));
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

  async function onUpdatePrompt(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const token = getToken();
    if (!token) {
      return;
    }
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
      const message = err instanceof Error ? err.message : "Failed to update prompt";
      setError(message);
    } finally {
      setUpdateLoading(false);
    }
  }

  async function onCreateVersion(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const token = getToken();
    if (!token) {
      return;
    }

    setCreateVersionLoading(true);
    setError("");
    try {
      let assets: unknown = {};
      if (assetsInput.trim() !== "") {
        assets = JSON.parse(assetsInput);
      }

      await createPromptVersion(token, promptId, {
        assets,
        frameworkId,
        techniqueIds: parseCsv(techniqueIdsInput),
        composedOutput,
      });

      setAssetsInput("{}");
      setFrameworkId("");
      setTechniqueIdsInput("");
      setComposedOutput("");
      await loadAll();
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to create prompt version";
      setError(message);
    } finally {
      setCreateVersionLoading(false);
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
              <h2>Prompt Data</h2>
              <p className="muted">Endpoints: GET/PATCH /api/v1/prompts/{`{promptID}`}</p>
              {prompt ? (
                <p className="muted">
                  Created {new Date(prompt.createdAt).toLocaleString()} | Updated{" "}
                  {new Date(prompt.updatedAt).toLocaleString()}
                </p>
              ) : null}
              <form onSubmit={onUpdatePrompt}>
                <label className="field">
                  Title
                  <input
                    className="input"
                    required
                    value={title}
                    onChange={(event) => setTitle(event.target.value)}
                  />
                </label>

                <label className="field">
                  Status
                  <input className="input" value={status} onChange={(event) => setStatus(event.target.value)} />
                </label>

                <label className="field">
                  Category
                  <input
                    className="input"
                    value={category}
                    onChange={(event) => setCategory(event.target.value)}
                  />
                </label>

                <label className="field">
                  Tags (comma separated)
                  <input
                    className="input"
                    value={tagsInput}
                    onChange={(event) => setTagsInput(event.target.value)}
                  />
                </label>

                <button className="btn btn-primary" type="submit" disabled={updateLoading}>
                  {updateLoading ? "Saving..." : "Update prompt"}
                </button>
              </form>
            </article>

            <article className="card">
              <h2>Create Prompt Version</h2>
              <p className="muted">Endpoint: POST /api/v1/prompts/{`{promptID}`}/versions</p>
              <form onSubmit={onCreateVersion}>
                <label className="field">
                  Assets (JSON)
                  <textarea
                    className="textarea mono"
                    rows={4}
                    value={assetsInput}
                    onChange={(event) => setAssetsInput(event.target.value)}
                  />
                </label>

                <label className="field">
                  Framework ID
                  <input
                    className="input"
                    value={frameworkId}
                    onChange={(event) => setFrameworkId(event.target.value)}
                  />
                </label>

                <label className="field">
                  Technique IDs (comma separated)
                  <input
                    className="input"
                    value={techniqueIdsInput}
                    onChange={(event) => setTechniqueIdsInput(event.target.value)}
                  />
                </label>

                <label className="field">
                  Composed Output
                  <textarea
                    className="textarea"
                    rows={4}
                    value={composedOutput}
                    onChange={(event) => setComposedOutput(event.target.value)}
                  />
                </label>

                <button className="btn btn-primary" type="submit" disabled={createVersionLoading}>
                  {createVersionLoading ? "Creating version..." : "Create version"}
                </button>
              </form>
            </article>

            <article className="card">
              <h2>Versions</h2>
              <p className="muted">Endpoint: GET /api/v1/prompts/{`{promptID}`}/versions</p>
              {versions.length === 0 ? (
                <p>No versions yet.</p>
              ) : (
                <ul className="list">
                  {versions.map((version) => (
                    <li key={version.id} className="card">
                      <p>
                        <strong>Version #{version.versionNumber}</strong>
                      </p>
                      <p className="mono">ID: {version.id}</p>
                      <p className="muted">Framework: {version.frameworkId || "-"}</p>
                      <p className="muted">
                        Techniques:{" "}
                        {version.techniqueIds.length > 0 ? version.techniqueIds.join(", ") : "none"}
                      </p>
                      <p>{version.composedOutput || "No composed output"}</p>
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
