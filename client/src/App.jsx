import { useEffect, useState } from "react";

const initialStatus = {
  status: "loading",
  message: "Checking API...",
};

export default function App() {
  const [health, setHealth] = useState(initialStatus);

  useEffect(() => {
    let cancelled = false;

    async function loadHealth() {
      try {
        const response = await fetch("/api/health");

        if (!response.ok) {
          throw new Error(`Unexpected status: ${response.status}`);
        }

        const data = await response.json();

        if (!cancelled) {
          let message = "API is running. PostgreSQL wiring is ready for the next step.";

          if (data.database.connected) {
            message = "API and database are connected.";
          } else if (data.database.configured) {
            message =
              "API is running and DATABASE_URL is set. Add a Postgres driver next.";
          }

          setHealth({
            status: "ready",
            message,
          });
        }
      } catch (error) {
        if (!cancelled) {
          setHealth({
            status: "error",
            message: error.message,
          });
        }
      }
    }

    loadHealth();

    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <main className="min-h-screen bg-gradient-to-br from-slate-950 via-slate-900 to-cyan-950 px-6 py-12 text-slate-100">
      <div className="mx-auto flex max-w-4xl flex-col gap-8">
        <header className="space-y-4">
          <p className="text-sm font-semibold uppercase tracking-[0.35em] text-cyan-300">
            Foxygen Vibe
          </p>
          <h1 className="max-w-2xl text-4xl font-bold tracking-tight sm:text-6xl">
            React, Tailwind, Go, and PostgreSQL in one starter.
          </h1>
          <p className="max-w-2xl text-base text-slate-300 sm:text-lg">
            This is the minimal working shell for the fullstack app. The client
            calls the Go API, and the API already accepts
            <code className="ml-1 rounded bg-slate-800 px-2 py-1 text-sm">
              DATABASE_URL
            </code>
            for the PostgreSQL layer.
          </p>
        </header>

        <section className="grid gap-4 md:grid-cols-2">
          <div className="rounded-3xl border border-white/10 bg-white/5 p-6 shadow-2xl shadow-cyan-950/50 backdrop-blur">
            <p className="text-sm font-medium uppercase tracking-[0.3em] text-slate-400">
              Client status
            </p>
            <p className="mt-4 text-2xl font-semibold">Vite is serving React.</p>
            <p className="mt-2 text-slate-300">
              Tailwind utilities are active and the frontend is ready for app
              routes and components.
            </p>
          </div>

          <div className="rounded-3xl border border-white/10 bg-white/5 p-6 shadow-2xl shadow-cyan-950/50 backdrop-blur">
            <p className="text-sm font-medium uppercase tracking-[0.3em] text-slate-400">
              API status
            </p>
            <p
              className={`mt-4 text-2xl font-semibold ${
                health.status === "error" ? "text-rose-300" : "text-emerald-300"
              }`}
            >
              {health.status === "loading" ? "Loading..." : health.status}
            </p>
            <p className="mt-2 text-slate-300">{health.message}</p>
          </div>
        </section>
      </div>
    </main>
  );
}
