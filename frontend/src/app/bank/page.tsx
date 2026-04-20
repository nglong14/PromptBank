"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import ProtectedPage from "@/components/ProtectedPage";
import { listPromptVersions, listPrompts } from "@/lib/api";
import { getToken } from "@/lib/auth";
import type { Prompt, PromptVersion } from "@/lib/types";

type PromptWithVersions = {
  prompt: Prompt;
  versions: PromptVersion[];
};

export default function BankPage() {
  const [items, setItems] = useState<PromptWithVersions[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  async function loadBank() {
    const token = getToken();
    if (!token) {
      return;
    }

    setLoading(true);
    setError("");

    try {
      const promptsRes = await listPrompts(token);
      const prompts = [...promptsRes.items].sort(
        (a, b) => new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime(),
      );

      const versionLists = await Promise.all(
        prompts.map(async (prompt) => {
          try {
            const versionsRes = await listPromptVersions(token, prompt.id);
            return versionsRes.items;
          } catch {
            return [];
          }
        }),
      );

      const mapped = prompts.map((prompt, index) => ({
        prompt,
        versions: [...versionLists[index]].sort((a, b) => b.versionNumber - a.versionNumber),
      }));

      setItems(mapped);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to load prompt bank";
      setError(message);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void loadBank();
  }, []);

  const totalVersions = useMemo(
    () => items.reduce((sum, item) => sum + item.versions.length, 0),
    [items],
  );

  return (
    <ProtectedPage>
      <section className="stack">
        <div className="row">
          <div>
            <h1 className="page-title">Prompt Bank</h1>
            <p className="subtitle">
              All prompts in your account, each with its version history.
            </p>
          </div>
          <button className="btn btn-secondary" type="button" onClick={() => void loadBank()}>
            Refresh
          </button>
        </div>

        <article className="card">
          <p className="muted">
            Prompts: {items.length} | Versions: {totalVersions}
          </p>
          {error ? <p className="error">{error}</p> : null}
          {loading ? (
            <p>Loading prompt bank...</p>
          ) : items.length === 0 ? (
            <p>No prompts yet. Create your first prompt from the Create Prompt page.</p>
          ) : (
            <ul className="list">
              {items.map(({ prompt, versions }) => (
                <li key={prompt.id} className="card">
                  <div className="row">
                    <div>
                      <strong>{prompt.title}</strong>
                      <p className="muted">
                        {prompt.status} | {prompt.category || "uncategorized"} | Updated{" "}
                        {new Date(prompt.updatedAt).toLocaleString()}
                      </p>
                      <p className="mono">{prompt.id}</p>
                    </div>
                    <Link href={`/prompts/${prompt.id}`}>Open</Link>
                  </div>

                  {prompt.tags.length > 0 ? (
                    <p className="muted">Tags: {prompt.tags.join(", ")}</p>
                  ) : null}

                  <div className="stack">
                    <p className="muted">Versions ({versions.length})</p>
                    {versions.length === 0 ? (
                      <p className="muted">No versions yet.</p>
                    ) : (
                      <ul className="list">
                        {versions.map((version) => (
                          <li key={version.id} className="card version-card">
                            <div className="row">
                              <strong>v{version.versionNumber}</strong>
                              <span className="muted">
                                {new Date(version.createdAt).toLocaleString()}
                              </span>
                            </div>
                            <p className="muted">
                              Framework: {version.frameworkId || "none"} | Techniques:{" "}
                              {version.techniqueIds.length > 0
                                ? version.techniqueIds.join(", ")
                                : "none"}
                            </p>
                            <p className="mono">{version.id}</p>
                          </li>
                        ))}
                      </ul>
                    )}
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
