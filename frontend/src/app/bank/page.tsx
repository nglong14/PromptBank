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

type SortOption = "recent" | "oldest" | "alpha" | "versions";

function formatRelativeTime(value: string): string {
  const diffMs = new Date(value).getTime() - Date.now();
  const formatter = new Intl.RelativeTimeFormat(undefined, { numeric: "auto" });
  const ranges: Array<[Intl.RelativeTimeFormatUnit, number]> = [
    ["year", 1000 * 60 * 60 * 24 * 365],
    ["month", 1000 * 60 * 60 * 24 * 30],
    ["day", 1000 * 60 * 60 * 24],
    ["hour", 1000 * 60 * 60],
    ["minute", 1000 * 60],
  ];

  for (const [unit, ms] of ranges) {
    if (Math.abs(diffMs) >= ms || unit === "minute") {
      return formatter.format(Math.round(diffMs / ms), unit);
    }
  }

  return "just now";
}

function getOutputPreview(version?: PromptVersion): string {
  if (!version?.composedOutput) {
    return "No composed output yet. Open this prompt to create or save a version.";
  }

  const compact = version.composedOutput.replace(/\s+/g, " ").trim();
  if (compact.length <= 140) {
    return compact;
  }

  return `${compact.slice(0, 137)}...`;
}

export default function BankPage() {
  const [items, setItems] = useState<PromptWithVersions[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [query, setQuery] = useState("");
  const [sortBy, setSortBy] = useState<SortOption>("recent");
  const [statusFilter, setStatusFilter] = useState("all");

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

  const filteredItems = useMemo(() => {
    const normalizedQuery = query.trim().toLowerCase();

    const filtered = items.filter(({ prompt }) => {
      const matchesStatus = statusFilter === "all" || prompt.status === statusFilter;
      if (!matchesStatus) {
        return false;
      }

      if (!normalizedQuery) {
        return true;
      }

      const haystack = [prompt.title, prompt.category, ...prompt.tags].join(" ").toLowerCase();
      return haystack.includes(normalizedQuery);
    });

    const sorted = [...filtered];
    sorted.sort((a, b) => {
      switch (sortBy) {
        case "oldest":
          return new Date(a.prompt.updatedAt).getTime() - new Date(b.prompt.updatedAt).getTime();
        case "alpha":
          return a.prompt.title.localeCompare(b.prompt.title);
        case "versions":
          return b.versions.length - a.versions.length;
        case "recent":
        default:
          return new Date(b.prompt.updatedAt).getTime() - new Date(a.prompt.updatedAt).getTime();
      }
    });

    return sorted;
  }, [items, query, sortBy, statusFilter]);

  const statusOptions = useMemo(() => {
    const statuses = new Set(items.map(({ prompt }) => prompt.status).filter(Boolean));
    return ["all", ...Array.from(statuses)];
  }, [items]);

  return (
    <ProtectedPage>
      <section className="stack">
        <div className="row">
          <div>
            <h1 className="page-title">Prompt Bank</h1>
            <p className="subtitle">
              Browse your prompts as a library. Each card shows the latest saved version at a glance.
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
            <p>No prompts yet. Create your first prompt from the New Prompt page.</p>
          ) : (
            <>
              <div className="bank-toolbar">
                <label className="field" style={{ marginBottom: 0 }}>
                  Search
                  <input
                    className="input"
                    placeholder="Search by title, category, or tag"
                    value={query}
                    onChange={(event) => setQuery(event.target.value)}
                  />
                </label>

                <label className="field" style={{ marginBottom: 0 }}>
                  Sort
                  <select
                    className="input"
                    value={sortBy}
                    onChange={(event) => setSortBy(event.target.value as SortOption)}
                  >
                    <option value="recent">Recently updated</option>
                    <option value="oldest">Oldest updated</option>
                    <option value="alpha">A-Z</option>
                    <option value="versions">Most versions</option>
                  </select>
                </label>
              </div>

              <div className="status-chips" role="tablist" aria-label="Filter prompts by status">
                {statusOptions.map((status) => (
                  <button
                    key={status}
                    type="button"
                    className={`status-chip${statusFilter === status ? " status-chip-active" : ""}`}
                    onClick={() => setStatusFilter(status)}
                  >
                    {status}
                  </button>
                ))}
              </div>

              {filteredItems.length === 0 ? (
                <p className="muted">No prompts match the current search or status filter.</p>
              ) : (
                <div className="bank-grid">
                  {filteredItems.map(({ prompt, versions }) => {
                    const latest = versions[0];
                    const visibleTags = prompt.tags.slice(0, 3);
                    const extraTagCount = prompt.tags.length - visibleTags.length;

                    return (
                      <article key={prompt.id} className="card prompt-card" title={prompt.id}>
                        <div className="prompt-card-header">
                          <div className="stack" style={{ gap: "0.4rem" }}>
                            <h2 className="prompt-card-title line-clamp-2">{prompt.title}</h2>
                            <div className="prompt-card-meta">
                              <span className="status-badge">{prompt.status}</span>
                              <span className="tag-pill tag-pill-muted">
                                {prompt.category || "uncategorized"}
                              </span>
                              <span className="muted">{formatRelativeTime(prompt.updatedAt)}</span>
                            </div>
                          </div>
                          <Link href={`/prompts/${prompt.id}`}>Open</Link>
                        </div>

                        <div className="prompt-tag-row">
                          {visibleTags.map((tag) => (
                            <span key={tag} className="tag-pill tag-pill-muted">
                              {tag}
                            </span>
                          ))}
                          {extraTagCount > 0 ? (
                            <span className="tag-pill tag-pill-muted">+{extraTagCount} more</span>
                          ) : null}
                          {visibleTags.length === 0 ? (
                            <span className="muted">No tags</span>
                          ) : null}
                        </div>

                        <div className="prompt-card-summary">
                          <strong>{latest ? `v${latest.versionNumber}` : "No versions yet"}</strong>
                          <span className="muted">
                            {latest?.frameworkId || "no framework"} •{" "}
                            {latest ? `${latest.techniqueIds.length} techniques` : "0 techniques"}
                          </span>
                        </div>

                        <p className="prompt-preview mono line-clamp-4">{getOutputPreview(latest)}</p>

                        <div className="prompt-card-footer">
                          <span className="muted">
                            {versions.length === 1 ? "1 version" : `${versions.length} versions`}
                          </span>
                          {latest ? (
                            <span className="muted">
                              Latest saved {formatRelativeTime(latest.createdAt)}
                            </span>
                          ) : null}
                        </div>
                      </article>
                    );
                  })}
                </div>
              )}
            </>
          )}
        </article>
      </section>
    </ProtectedPage>
  );
}
