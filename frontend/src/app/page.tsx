"use client";

import Link from "next/link";

export default function Home() {
  return (
    <section className="stack">
      <article className="card">
        <h2>Create Prompt</h2>
        <p className="muted">Start a new prompt and continue in the guided wizard.</p>
        <Link href="/prompts" className="btn btn-primary">
          Create Prompt
        </Link>
      </article>

      <article className="card">
        <h2>Browse Prompts</h2>
        <p className="muted">Open your prompt library and jump into existing drafts or versions.</p>
        <Link href="/bank" className="btn btn-secondary">
          Browse Prompts
        </Link>
      </article>

      <article className="card">
        <h2>Account</h2>
        <p className="muted">Sign in to manage prompts or create a new account.</p>
        <div className="row">
          <Link href="/login" className="btn btn-secondary">
            Login
          </Link>
          <Link href="/signup" className="btn btn-secondary">
            Sign up
          </Link>
        </div>
      </article>
    </section>
  );
}
