import { useEffect, useState } from "react";

const accessTokenKey = "foxygen.access_token";
const refreshTokenKey = "foxygen.refresh_token";

const demoAccounts = [
  {
    username: "mobile.lead",
    password: "Alpha123!",
    title: "Mobile Lead",
  },
  {
    username: "qa.runner",
    password: "Beta123!",
    title: "QA Runner",
  },
  {
    username: "ops.viewer",
    password: "Gamma123!",
    title: "Ops Viewer",
  },
];

const initialForm = {
  username: "",
  password: "",
};

const initialFeedback = {
  tone: "idle",
  message: "",
};

function currentRoute() {
  return window.location.pathname === "/dashboard" ? "/dashboard" : "/";
}

export default function App() {
  const [form, setForm] = useState(initialForm);
  const [feedback, setFeedback] = useState(initialFeedback);
  const [session, setSession] = useState(null);
  const [route, setRoute] = useState(currentRoute);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isRefreshing, setIsRefreshing] = useState(false);

  useEffect(() => {
    function handlePopState() {
      setRoute(currentRoute());
    }

    window.addEventListener("popstate", handlePopState);

    return () => {
      window.removeEventListener("popstate", handlePopState);
    };
  }, []);

  useEffect(() => {
    const accessToken = window.localStorage.getItem(accessTokenKey);
    const refreshToken = window.localStorage.getItem(refreshTokenKey);

    if (!accessToken) {
      if (route === "/dashboard") {
        navigate("/");
      }
      return;
    }

    loadSession(accessToken).catch(async () => {
      if (!refreshToken) {
        clearTokens();
        if (route === "/dashboard") {
          navigate("/");
        }
        return;
      }

      try {
        await rotateSession(refreshToken, true);
      } catch {
        clearTokens();
        if (route === "/dashboard") {
          navigate("/");
        }
      }
    });
  }, []);

  function navigate(nextRoute) {
    if (window.location.pathname !== nextRoute) {
      window.history.pushState({}, "", nextRoute);
    }
    setRoute(nextRoute);
  }

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
      navigate("/dashboard");
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
    const response = await fetch("/api/profile", {
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

    if (currentRoute() === "/dashboard") {
      setRoute("/dashboard");
    }

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
    navigate("/");
  }

  function autofillDemoAccount(account) {
    setForm({
      username: account.username,
      password: account.password,
    });
    setFeedback({
      tone: "success",
      message: `Loaded ${account.username}.`,
    });
  }

  const activeDemo = demoAccounts.find((account) => account.username === session?.username);

  return (
    <main className="min-h-screen bg-gradient-to-br from-slate-950 via-slate-900 to-cyan-950 px-6 py-12 text-slate-100">
      <div className="mx-auto flex min-h-[calc(100vh-6rem)] max-w-6xl items-center justify-center">
        {route === "/dashboard" && session ? (
          <section className="grid w-full gap-6 lg:grid-cols-[1.35fr_0.85fr]">
            <div className="rounded-[2rem] border border-white/10 bg-white/10 p-8 shadow-2xl shadow-cyan-950/50 backdrop-blur-xl">
              <p className="text-xs font-semibold uppercase tracking-[0.45em] text-cyan-300">
                Mobile Engineer V3
              </p>
              <h1 className="mt-4 text-4xl font-bold tracking-tight sm:text-5xl">
                Dashboard
              </h1>
              <p className="mt-3 max-w-xl text-sm text-slate-300 sm:text-base">
                Your session is active. This dashboard now pulls a protected
                profile payload with the access token, and refresh rotation is
                still available without leaving the page.
              </p>

              <div className="mt-8 grid gap-4 sm:grid-cols-2">
                <article className="rounded-3xl border border-white/10 bg-slate-950/30 p-5">
                  <p className="text-xs font-semibold uppercase tracking-[0.3em] text-slate-400">
                    Name
                  </p>
                  <p className="mt-3 text-2xl font-semibold text-slate-50">
                    {session.name || "Not set"}
                  </p>
                </article>

                <article className="rounded-3xl border border-white/10 bg-slate-950/30 p-5">
                  <p className="text-xs font-semibold uppercase tracking-[0.3em] text-slate-400">
                    Username
                  </p>
                  <p className="mt-3 text-2xl font-semibold text-slate-50">
                    {session.username}
                  </p>
                </article>

                <article className="rounded-3xl border border-white/10 bg-slate-950/30 p-5">
                  <p className="text-xs font-semibold uppercase tracking-[0.3em] text-slate-400">
                    Email
                  </p>
                  <p className="mt-3 break-all text-sm text-slate-200">
                    {session.email || "Not set"}
                  </p>
                </article>

                <article className="rounded-3xl border border-white/10 bg-slate-950/30 p-5">
                  <p className="text-xs font-semibold uppercase tracking-[0.3em] text-slate-400">
                    User ID
                  </p>
                  <p className="mt-3 break-all text-sm text-slate-200">
                    {session.user_id}
                  </p>
                </article>

                <article className="rounded-3xl border border-white/10 bg-slate-950/30 p-5 sm:col-span-2">
                  <p className="text-xs font-semibold uppercase tracking-[0.3em] text-slate-400">
                    Department
                  </p>
                  <p className="mt-3 text-lg font-semibold text-slate-100">
                    {session.department || "Unassigned"}
                  </p>
                </article>

                <article className="rounded-3xl border border-white/10 bg-slate-950/30 p-5 sm:col-span-2">
                  <p className="text-xs font-semibold uppercase tracking-[0.3em] text-slate-400">
                    Session Notes
                  </p>
                  <p className="mt-3 text-sm leading-7 text-slate-300">
                    Access tokens are verified with the JWT signature.
                    Refresh tokens are rotated server-side and replaced after
                    each refresh request.
                  </p>
                </article>
              </div>
            </div>

            <aside className="space-y-6">
              <section className="rounded-[2rem] border border-emerald-300/20 bg-emerald-400/10 p-6 shadow-2xl shadow-emerald-950/30 backdrop-blur-xl">
                <p className="text-xs font-semibold uppercase tracking-[0.35em] text-emerald-300">
                  Signed In
                </p>
                <p className="mt-4 text-xl font-semibold text-slate-50">
                  {activeDemo ? activeDemo.title : "Authenticated User"}
                </p>
                <p className="mt-2 text-sm text-slate-300">
                  {activeDemo
                    ? `This session is using the seeded ${activeDemo.username} demo account.`
                    : "This session is using a non-demo account."}
                </p>

                <div className="mt-5 grid gap-3">
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
                    Sign Out
                  </button>
                </div>
              </section>

              {feedback.message ? (
                <section className="rounded-3xl border border-white/10 bg-white/10 p-5 backdrop-blur-xl">
                  <p
                    className={`text-sm ${
                      feedback.tone === "error"
                        ? "text-rose-300"
                        : "text-emerald-300"
                    }`}
                  >
                    {feedback.message}
                  </p>
                </section>
              ) : null}
            </aside>
          </section>
        ) : (
          <section className="grid w-full max-w-5xl gap-6 lg:grid-cols-[1.05fr_0.95fr]">
            <div className="rounded-[2rem] border border-white/10 bg-white/10 p-8 shadow-2xl shadow-cyan-950/50 backdrop-blur-xl">
              <p className="text-xs font-semibold uppercase tracking-[0.45em] text-cyan-300">
                Mobile Engineer V3
              </p>
              <h1 className="mt-4 text-4xl font-bold tracking-tight sm:text-5xl">
                Sign In
              </h1>
              <p className="mt-3 max-w-md text-sm text-slate-300 sm:text-base">
                Use one of the seeded demo accounts or enter another valid
                username and password. Successful login redirects to the
                dashboard.
              </p>

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
                      feedback.tone === "error"
                        ? "text-rose-300"
                        : "text-emerald-300"
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
            </div>

            <aside className="rounded-[2rem] border border-white/10 bg-white/10 p-8 shadow-2xl shadow-cyan-950/50 backdrop-blur-xl">
              <p className="text-xs font-semibold uppercase tracking-[0.35em] text-slate-400">
                Demo Accounts
              </p>
              <div className="mt-6 space-y-4">
                {demoAccounts.map((account) => (
                  <button
                    key={account.username}
                    type="button"
                    onClick={() => autofillDemoAccount(account)}
                    className="block w-full rounded-3xl border border-white/10 bg-slate-950/30 p-5 text-left transition hover:border-cyan-300/40 hover:bg-slate-950/40"
                  >
                    <p className="text-xs font-semibold uppercase tracking-[0.3em] text-cyan-300">
                      {account.title}
                    </p>
                    <p className="mt-3 text-lg font-semibold text-slate-50">
                      {account.username}
                    </p>
                    <p className="mt-1 text-sm text-slate-300">
                      Password: {account.password}
                    </p>
                  </button>
                ))}
              </div>
            </aside>
          </section>
        )}
      </div>
    </main>
  );
}
