"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { getHealth } from "@/lib/api";

export default function Home() {
  const [status, setStatus] = useState("checking...");
  const [error, setError] = useState("");

  useEffect(() => {
    async function loadHealth() {
      try {
        const response = await getHealth();
        setStatus(response.status);
      } catch (err) {
        const message = err instanceof Error ? err.message : "Unable to reach backend";
        setError(message);
      }
    }

    void loadHealth();
  }, []);

  return (
    <section className="stack">
      <div>
        <h1 className="page-title">PromptBank Frontend</h1>
        <p className="subtitle">
          Minimal light-theme Next.js UI for signup, login, and every PromptBank API endpoint.
        </p>
      </div>

      <article className="card">
        <h2>API Health</h2>
        <p className="muted">Endpoint: GET /health</p>
        {error ? <p className="error">{error}</p> : <p>Status: {status}</p>}
      </article>

      <article className="card">
        <h2>Quick Actions</h2>
        <div className="row">
          <Link href="/signup">Create account</Link>
          <Link href="/login">Login</Link>
          <Link href="/prompts">Go to prompts</Link>
        </div>
      </article>
    </section>
  );
}
