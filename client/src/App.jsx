import { useEffect, useState } from "react";

const accessTokenKey = "foxygen.access_token";
const refreshTokenKey = "foxygen.refresh_token";

const initialForm = {
  username: "",
  password: "",
};

const initialFeedback = {
  tone: "idle",
  message: "",
};

export default function App() {
  const [form, setForm] = useState(initialForm);
  const [feedback, setFeedback] = useState(initialFeedback);
  const [session, setSession] = useState(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isRefreshing, setIsRefreshing] = useState(false);

  useEffect(() => {
    const accessToken = window.localStorage.getItem(accessTokenKey);
    const refreshToken = window.localStorage.getItem(refreshTokenKey);

    if (!accessToken) {
      return;
    }

    loadSession(accessToken).catch(async () => {
      if (!refreshToken) {
        clearTokens();
        return;
      }

      try {
        await rotateSession(refreshToken, true);
      } catch {
        clearTokens();
      }
    });
  }, []);

  function handleChange(event) {
    const { name, value } = event.target;

    setForm((current) => ({
      ...current,
      [name]: value,
    }));
  }

  async function handleSubmit(event) {
    event.preventDefault();

    const username = form.username.trim();
    const password = form.password.trim();

    if (!username || !password) {
      setFeedback({
        tone: "error",
        message: "Username and password are required.",
      });
      return;
    }

    setIsSubmitting(true);
    setFeedback(initialFeedback);

    try {
      const response = await fetch("/api/auth/login", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          username,
          password,
        }),
      });

      if (!response.ok) {
        const errorMessage = await response.text();
        throw new Error(errorMessage || "Authentication failed.");
      }

      const data = await response.json();
      storeTokens(data);
      await loadSession(data.access_token);

      setFeedback({
        tone: "success",
        message: `Welcome back, ${data.username}.`,
      });
    } catch (error) {
      clearTokens();
      setSession(null);
      setFeedback({
        tone: "error",
        message: error.message,
      });
    } finally {
      setIsSubmitting(false);
    }
  }

  async function loadSession(accessToken) {
    const response = await fetch("/api/auth/session", {
      headers: {
        Authorization: `Bearer ${accessToken}`,
      },
    });

    if (!response.ok) {
      const errorMessage = await response.text();
      throw new Error(errorMessage || "Session validation failed.");
    }

    const data = await response.json();
    setSession(data);
    return data;
  }

  async function rotateSession(currentRefreshToken, silent = false) {
    setIsRefreshing(true);

    try {
      const response = await fetch("/api/auth/refresh", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          refresh_token: currentRefreshToken,
        }),
      });

      if (!response.ok) {
        const errorMessage = await response.text();
        throw new Error(errorMessage || "Token rotation failed.");
      }

      const data = await response.json();
      storeTokens(data);
      await loadSession(data.access_token);

      if (!silent) {
        setFeedback({
          tone: "success",
          message: "Session rotated successfully.",
        });
      }

      return data;
    } catch (error) {
      clearTokens();
      setSession(null);

      if (!silent) {
        setFeedback({
          tone: "error",
          message: error.message,
        });
      }

      throw error;
    } finally {
      setIsRefreshing(false);
    }
  }

  function storeTokens(payload) {
    window.localStorage.setItem(accessTokenKey, payload.access_token);
    window.localStorage.setItem(refreshTokenKey, payload.refresh_token);
  }

  function clearTokens() {
    window.localStorage.removeItem(accessTokenKey);
    window.localStorage.removeItem(refreshTokenKey);
  }

  function handleRotate() {
    const currentRefreshToken = window.localStorage.getItem(refreshTokenKey);

    if (!currentRefreshToken) {
      setFeedback({
        tone: "error",
        message: "No refresh token available.",
      });
      return;
    }

    setFeedback(initialFeedback);
    rotateSession(currentRefreshToken).catch(() => {});
  }

  function handleSignOut() {
    clearTokens();
    setSession(null);
    setFeedback({
      tone: "success",
      message: "Signed out.",
    });
  }

  return (
    <main className="min-h-screen bg-gradient-to-br from-slate-950 via-slate-900 to-cyan-950 px-6 py-12 text-slate-100">
      <div className="flex min-h-[calc(100vh-6rem)] items-center justify-center">
        <section className="w-full max-w-md rounded-3xl border border-white/10 bg-white/10 p-8 shadow-2xl shadow-cyan-950/50 backdrop-blur-xl">
          <div className="text-center">
            <p className="text-xs font-semibold uppercase tracking-[0.45em] text-cyan-300">
              Mobile Engineer V3
            </p>
            <h1 className="mt-4 text-3xl font-bold tracking-tight sm:text-4xl">
              Sign in
            </h1>
            <p className="mt-3 text-sm text-slate-300">
              Enter your username and password to continue.
            </p>
          </div>

          <form className="mt-8 space-y-5" onSubmit={handleSubmit}>
            <label className="block">
              <span className="mb-2 block text-sm font-medium text-slate-200">
                Username
              </span>
              <input
                type="text"
                name="username"
                value={form.username}
                onChange={handleChange}
                autoComplete="username"
                placeholder="mobile.engineer"
                className="w-full rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3 text-base text-slate-100 outline-none transition focus:border-cyan-400 focus:ring-2 focus:ring-cyan-400/30"
              />
            </label>

            <label className="block">
              <span className="mb-2 block text-sm font-medium text-slate-200">
                Password
              </span>
              <input
                type="password"
                name="password"
                value={form.password}
                onChange={handleChange}
                autoComplete="current-password"
                placeholder="Enter your password"
                className="w-full rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3 text-base text-slate-100 outline-none transition focus:border-cyan-400 focus:ring-2 focus:ring-cyan-400/30"
              />
            </label>

            {feedback.message ? (
              <p
                className={`text-sm ${
                  feedback.tone === "error" ? "text-rose-300" : "text-emerald-300"
                }`}
              >
                {feedback.message}
              </p>
            ) : null}

            <button
              type="submit"
              disabled={isSubmitting}
              className="w-full rounded-2xl bg-cyan-400 px-4 py-3 text-sm font-semibold uppercase tracking-[0.25em] text-slate-950 transition hover:bg-cyan-300 disabled:cursor-not-allowed disabled:bg-cyan-200"
            >
              {isSubmitting ? "Working..." : "Authenticate"}
            </button>
          </form>

          {session ? (
            <div className="mt-6 rounded-2xl border border-emerald-300/20 bg-emerald-400/10 p-4">
              <p className="text-xs font-semibold uppercase tracking-[0.3em] text-emerald-300">
                Active Session
              </p>
              <p className="mt-3 text-lg font-semibold text-slate-50">
                {session.username}
              </p>
              <p className="mt-1 text-xs text-slate-300">{session.user_id}</p>
              <div className="mt-4 grid grid-cols-2 gap-3">
                <button
                  type="button"
                  onClick={handleRotate}
                  disabled={isRefreshing}
                  className="rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-xs font-semibold uppercase tracking-[0.2em] text-slate-100 transition hover:bg-white/10 disabled:cursor-not-allowed disabled:opacity-60"
                >
                  {isRefreshing ? "Rotating..." : "Rotate Token"}
                </button>
                <button
                  type="button"
                  onClick={handleSignOut}
                  className="rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3 text-xs font-semibold uppercase tracking-[0.2em] text-slate-100 transition hover:bg-slate-950/60"
                >
                  Sign out
                </button>
              </div>
            </div>
          ) : null}
        </section>
      </div>
    </main>
  );
}
