export function CoordinatorDashboard() {
  return (
    <section className="grid gap-4 sm:grid-cols-2">
      <article className="rounded-3xl border border-white/10 bg-white/5 p-5">
        <p className="text-xs font-semibold uppercase tracking-[0.25em] text-slate-400">Команда</p>
        <p className="mt-3 text-2xl font-semibold text-slate-100">24 инженера</p>
      </article>
      <article className="rounded-3xl border border-white/10 bg-white/5 p-5">
        <p className="text-xs font-semibold uppercase tracking-[0.25em] text-slate-400">Требуют внимания</p>
        <p className="mt-3 text-2xl font-semibold text-slate-100">5 задач</p>
      </article>
    </section>
  );
}
