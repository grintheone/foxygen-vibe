import { StatusMessage } from "../../../shared/ui/status-message";

export function AuthForm({
  feedback,
  form,
  isSubmitting,
  onChange,
  onSubmit,
}) {
  return (
    <div className="rounded-[2rem] border border-white/10 bg-white/10 p-8 shadow-2xl shadow-cyan-950/50 backdrop-blur-xl">
      <p className="text-xs font-semibold uppercase tracking-[0.45em] text-cyan-300">
        Mobile Engineer V3
      </p>
      <h1 className="mt-4 text-4xl font-bold tracking-tight sm:text-5xl">
        Sign In
      </h1>
      <p className="mt-3 max-w-md text-sm text-slate-300 sm:text-base">
        Enter a valid username and password from the PostgreSQL-backed account
        store. Successful login redirects to the dashboard.
      </p>

      <form className="mt-8 space-y-5" onSubmit={onSubmit}>
        <label className="block">
          <span className="mb-2 block text-sm font-medium text-slate-200">
            Username
          </span>
          <input
            type="text"
            name="username"
            value={form.username}
            onChange={onChange}
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
            onChange={onChange}
            autoComplete="current-password"
            placeholder="Enter your password"
            className="w-full rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3 text-base text-slate-100 outline-none transition focus:border-cyan-400 focus:ring-2 focus:ring-cyan-400/30"
          />
        </label>

        {feedback.message ? <StatusMessage feedback={feedback} /> : null}

        <button
          type="submit"
          disabled={isSubmitting}
          className="w-full rounded-2xl bg-cyan-400 px-4 py-3 text-sm font-semibold uppercase tracking-[0.25em] text-slate-950 transition hover:bg-cyan-300 disabled:cursor-not-allowed disabled:bg-cyan-200"
        >
          {isSubmitting ? "Working..." : "Authenticate"}
        </button>
      </form>
    </div>
  );
}
