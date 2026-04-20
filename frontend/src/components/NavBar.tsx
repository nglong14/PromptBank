"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { clearToken, getToken, subscribeAuthChanges } from "@/lib/auth";

export default function NavBar() {
  const router = useRouter();
  const [isAuthenticated, setIsAuthenticated] = useState(false);

  useEffect(() => {
    const syncAuth = () => setIsAuthenticated(Boolean(getToken()));
    syncAuth();
    const unsubscribe = subscribeAuthChanges(syncAuth);
    return unsubscribe;
  }, []);

  function onLogout() {
    clearToken();
    setIsAuthenticated(false);
    router.push("/login");
  }

  return (
    <header className="nav">
      <div className="container nav-inner">
        <Link href="/" className="brand">
          PromptBank UI
        </Link>
        <nav className="links">
          <Link href="/">Home</Link>
          {isAuthenticated ? (
            <>
              <Link href="/prompts">Create Prompt</Link>
              <Link href="/bank">Bank</Link>
              <button type="button" className="btn btn-secondary" onClick={onLogout}>
                Logout
              </button>
            </>
          ) : (
            <>
              <Link href="/signup">Sign up</Link>
              <Link href="/login">Login</Link>
            </>
          )}
        </nav>
      </div>
    </header>
  );
}
