export function DashboardOverview({ session }) {
  return (
    <div className="rounded-[2rem] border border-white/10 bg-white/10 p-8 shadow-2xl shadow-cyan-950/50 backdrop-blur-xl">
      <p className="text-xs font-semibold uppercase tracking-[0.45em] text-cyan-300">
        Mobile Engineer V3
      </p>
      <h1 className="mt-4 text-4xl font-bold tracking-tight sm:text-5xl">
        Dashboard
      </h1>
      <p className="mt-3 max-w-xl text-sm text-slate-300 sm:text-base">
        Your session is active. This dashboard now pulls a protected profile
        payload with the access token, and refresh rotation is still available
        without leaving the page.
      </p>

      <div className="mt-8 grid gap-4 sm:grid-cols-2">
        <article className="rounded-3xl border border-white/10 bg-slate-950/30 p-5">
          <p className="text-xs font-semibold uppercase tracking-[0.3em] text-slate-400">
            Name
          </p>
          <p className="mt-3 text-2xl font-semibold text-slate-50">
            {session?.name || "Not set"}
          </p>
        </article>

        <article className="rounded-3xl border border-white/10 bg-slate-950/30 p-5">
          <p className="text-xs font-semibold uppercase tracking-[0.3em] text-slate-400">
            Username
          </p>
          <p className="mt-3 text-2xl font-semibold text-slate-50">
            {session?.username}
          </p>
        </article>

        <article className="rounded-3xl border border-white/10 bg-slate-950/30 p-5">
          <p className="text-xs font-semibold uppercase tracking-[0.3em] text-slate-400">
            Email
          </p>
          <p className="mt-3 break-all text-sm text-slate-200">
            {session?.email || "Not set"}
          </p>
        </article>

        <article className="rounded-3xl border border-white/10 bg-slate-950/30 p-5">
          <p className="text-xs font-semibold uppercase tracking-[0.3em] text-slate-400">
            User ID
          </p>
          <p className="mt-3 break-all text-sm text-slate-200">
            {session?.user_id}
          </p>
        </article>

        <article className="rounded-3xl border border-white/10 bg-slate-950/30 p-5 sm:col-span-2">
          <p className="text-xs font-semibold uppercase tracking-[0.3em] text-slate-400">
            Department
          </p>
          <p className="mt-3 text-lg font-semibold text-slate-100">
            {session?.department || "Unassigned"}
          </p>
        </article>

        <article className="rounded-3xl border border-white/10 bg-slate-950/30 p-5 sm:col-span-2">
          <p className="text-xs font-semibold uppercase tracking-[0.3em] text-slate-400">
            Session Notes
          </p>
          <p className="mt-3 text-sm leading-7 text-slate-300">
            Access tokens are verified with the JWT signature. Refresh tokens
            are rotated server-side and replaced after each refresh request.
          </p>
        </article>
      </div>
    </div>
  );
}
