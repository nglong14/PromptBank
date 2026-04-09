"use client";

import Link from "next/link";
import { FormEvent, useEffect, useState } from "react";
import ProtectedPage from "@/components/ProtectedPage";
import { createPrompt, derivePrompt, listPrompts } from "@/lib/api";
import { getToken } from "@/lib/auth";
import type { Prompt } from "@/lib/types";

function parseCsv(input: string): string[] {
  return input
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
}

export default function PromptsPage() {
  const [prompts, setPrompts] = useState<Prompt[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const [title, setTitle] = useState("");
  const [status, setStatus] = useState("draft");
  const [category, setCategory] = useState("");
  const [tagsInput, setTagsInput] = useState("");
  const [createLoading, setCreateLoading] = useState(false);

  const [sourcePromptId, setSourcePromptId] = useState("");
  const [sourceVersionId, setSourceVersionId] = useState("");
  const [newTitle, setNewTitle] = useState("");
  const [deriveLoading, setDeriveLoading] = useState(false);
  const [deriveResult, setDeriveResult] = useState("");

  async function loadPrompts() {
    const token = getToken();
    if (!token) {
      return;
    }

    setLoading(true);
    setError("");
    try {
      const response = await listPrompts(token);
      setPrompts(response.items);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to load prompts";
      setError(message);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void loadPrompts();
  }, []);

  async function onCreatePrompt(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const token = getToken();
    if (!token) {
      return;
    }

    setCreateLoading(true);
    setError("");
    try {
      await createPrompt(token, {
        title,
        status,
        category,
        tags: parseCsv(tagsInput),
      });
      setTitle("");
      setStatus("draft");
      setCategory("");
      setTagsInput("");
      await loadPrompts();
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to create prompt";
      setError(message);
    } finally {
      setCreateLoading(false);
    }
  }

  async function onDerivePrompt(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const token = getToken();
    if (!token) {
      return;
    }

    setDeriveLoading(true);
    setError("");
    setDeriveResult("");
    try {
      const response = await derivePrompt(token, {
        sourcePromptId,
        sourceVersionId: sourceVersionId || undefined,
        newTitle,
      });
      setDeriveResult(`Created prompt ${response.prompt.id} with version ${response.version.id}`);
      setSourcePromptId("");
      setSourceVersionId("");
      setNewTitle("");
      await loadPrompts();
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to derive prompt";
      setError(message);
    } finally {
      setDeriveLoading(false);
    }
  }

  return (
    <ProtectedPage>
      <section className="stack">
        <div>
          <h1 className="page-title">Prompts</h1>
          <p className="subtitle">Covers list/create endpoints and links to detail/version flows.</p>
        </div>

        <article className="card">
          <h2>Create Prompt</h2>
          <p className="muted">Endpoint: POST /api/v1/prompts</p>
          <form onSubmit={onCreatePrompt}>
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
              <input
                className="input"
                value={status}
                onChange={(event) => setStatus(event.target.value)}
                placeholder="draft"
              />
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
                placeholder="writing, creative"
              />
            </label>

            <button className="btn btn-primary" type="submit" disabled={createLoading}>
              {createLoading ? "Creating..." : "Create prompt"}
            </button>
          </form>
        </article>

        <article className="card">
          <h2>Derive Prompt</h2>
          <p className="muted">Endpoint: POST /api/v1/prompts/derive</p>
          <form onSubmit={onDerivePrompt}>
            <label className="field">
              Source Prompt ID
              <input
                className="input mono"
                required
                value={sourcePromptId}
                onChange={(event) => setSourcePromptId(event.target.value)}
              />
            </label>

            <label className="field">
              Source Version ID (optional)
              <input
                className="input mono"
                value={sourceVersionId}
                onChange={(event) => setSourceVersionId(event.target.value)}
              />
            </label>

            <label className="field">
              New Title
              <input
                className="input"
                required
                value={newTitle}
                onChange={(event) => setNewTitle(event.target.value)}
              />
            </label>

            <button className="btn btn-primary" type="submit" disabled={deriveLoading}>
              {deriveLoading ? "Deriving..." : "Derive prompt"}
            </button>
          </form>
          {deriveResult ? <p className="success">{deriveResult}</p> : null}
        </article>

        <article className="card">
          <div className="row">
            <h2>Your Prompt List</h2>
            <button className="btn btn-secondary" type="button" onClick={() => void loadPrompts()}>
              Refresh
            </button>
          </div>
          <p className="muted">Endpoint: GET /api/v1/prompts</p>
          {error ? <p className="error">{error}</p> : null}
          {loading ? (
            <p>Loading prompts...</p>
          ) : prompts.length === 0 ? (
            <p>No prompts yet.</p>
          ) : (
            <ul className="list">
              {prompts.map((prompt) => (
                <li key={prompt.id} className="card">
                  <div className="row">
                    <div>
                      <strong>{prompt.title}</strong>
                      <p className="muted">
                        {prompt.status} | {prompt.category || "uncategorized"}
                      </p>
                      <p className="mono">{prompt.id}</p>
                    </div>
                    <Link href={`/prompts/${prompt.id}`}>Open</Link>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </article>
      </section>
    </ProtectedPage>
  );
}
