export function SessionPanel({
  session,
  isRefreshing,
  onRotate,
  onSignOut,
}) {
  return (
    <section className="rounded-lg border border-emerald-300/20 bg-emerald-400/10 p-6 shadow-2xl shadow-emerald-950/30 backdrop-blur-xl">
      <p className="text-xs font-semibold uppercase tracking-[0.35em] text-emerald-300">
        Сессия активна
      </p>
      <p className="mt-4 text-xl font-semibold text-slate-50">
        {session?.username || "Авторизованный пользователь"}
      </p>
      <p className="mt-2 text-sm text-slate-300">
        Эта сессия использует учетную запись из PostgreSQL.
      </p>

      <div className="mt-5 grid gap-3">
        <button
          type="button"
          onClick={onRotate}
          disabled={isRefreshing}
          className="rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-xs font-semibold uppercase tracking-[0.2em] text-slate-100 transition hover:bg-white/10 disabled:cursor-not-allowed disabled:opacity-60"
        >
          {isRefreshing ? "Выполняется ротация..." : "Обновить токен"}
        </button>
        <button
          type="button"
          onClick={onSignOut}
          className="rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3 text-xs font-semibold uppercase tracking-[0.2em] text-slate-100 transition hover:bg-slate-950/60"
        >
          Выйти
        </button>
      </div>
    </section>
  );
}
