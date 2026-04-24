"use client";

import { useRouter } from "next/navigation";
import { FormEvent, useState } from "react";
import ProtectedPage from "@/components/ProtectedPage";
import { createPrompt, derivePrompt } from "@/lib/api";
import { getToken } from "@/lib/auth";

function parseCsv(input: string): string[] {
  return input
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
}

export default function PromptsPage() {
  const router = useRouter();
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

  async function onCreatePrompt(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const token = getToken();
    if (!token) {
      return;
    }

    setCreateLoading(true);
    setError("");
    try {
      const createdPrompt = await createPrompt(token, {
        title,
        status,
        category,
        tags: parseCsv(tagsInput),
      });
      setTitle("");
      setStatus("draft");
      setCategory("");
      setTagsInput("");
      router.push(`/prompts/${createdPrompt.id}`);
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
          <h1 className="page-title">New Prompt</h1>
          <p className="subtitle">
            Set up a prompt&apos;s metadata first. You&apos;ll build and save its versions on the next screen.
          </p>
        </div>
        {error ? <p className="error">{error}</p> : null}

        <article className="card">
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
          <h2>Fork from existing prompt</h2>
          <p className="muted">
            Create a new prompt by copying another prompt&apos;s latest version or a specific saved version.
          </p>
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
      </section>
    </ProtectedPage>
  );
}
