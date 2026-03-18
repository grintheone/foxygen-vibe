import { PageShell } from "../../../shared/ui/page-shell";

export function BackButton({ onClick }) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-label="Назад"
      className="inline-flex h-11 w-11 items-center justify-center rounded-2xl bg-[#6A3BF2] text-white shadow-lg shadow-[#6A3BF2]/35 transition hover:bg-[#7C52F5]"
    >
      <svg
        xmlns="http://www.w3.org/2000/svg"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="2.5"
        strokeLinecap="round"
        strokeLinejoin="round"
        className="h-5 w-5"
        aria-hidden="true"
      >
        <path d="M15 18l-6-6 6-6" />
      </svg>
    </button>
  );
}

export function SummaryCard({ label, value }) {
  return (
    <div className="rounded-3xl border border-white/10 bg-slate-950/35 p-4 shadow-lg shadow-black/15">
      <p className="text-xs font-semibold uppercase tracking-[0.24em] text-slate-500">{label}</p>
      <p className="mt-3 text-xl font-semibold text-slate-100">{value}</p>
    </div>
  );
}

export function EditorFormField({ label, children, hint }) {
  return (
    <label className="block">
      <span className="text-xs font-semibold uppercase tracking-[0.22em] text-slate-400">{label}</span>
      {children}
      {hint ? <span className="mt-2 block text-sm text-slate-500">{hint}</span> : null}
    </label>
  );
}

export function EditorNoAccess({ onBack }) {
  return (
    <PageShell>
      <section className="w-full space-y-6">
        <header className="rounded-3xl border border-white/10 bg-slate-950/35 p-6 shadow-xl shadow-black/20 backdrop-blur">
          <div className="flex items-center justify-between gap-4">
            <div>
              <p className="text-xs font-semibold uppercase tracking-[0.3em] text-slate-400">Редактор</p>
              <h1 className="mt-3 text-3xl font-bold tracking-tight text-white sm:text-4xl">Нет доступа</h1>
            </div>
            <BackButton onClick={onBack} />
          </div>
        </header>

        <section className="rounded-3xl border border-rose-300/20 bg-rose-500/10 p-6 shadow-xl shadow-black/20 backdrop-blur">
          <p className="text-base text-rose-50">
            Редактор пока доступен только координаторам и администраторам.
          </p>
        </section>
      </section>
    </PageShell>
  );
}

export function EditorEntityCard({ badge, description, disabled = false, onClick, title }) {
  const classes = disabled
    ? "cursor-not-allowed border-white/10 bg-white/5 text-slate-500 opacity-80"
    : "border-cyan-200/20 bg-cyan-400/10 text-slate-100 hover:border-cyan-100/40 hover:bg-cyan-400/15";

  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      className={`rounded-[2rem] border p-6 text-left shadow-xl shadow-black/15 transition ${classes}`}
    >
      <div className="flex items-start justify-between gap-4">
        <div>
          <p className="text-2xl font-bold tracking-tight">{title}</p>
          <p className="mt-3 max-w-md text-sm text-slate-300">{description}</p>
        </div>
        {badge ? (
          <span className="rounded-full border border-white/10 bg-black/15 px-3 py-1 text-xs font-semibold uppercase tracking-[0.2em] text-slate-200">
            {badge}
          </span>
        ) : null}
      </div>
    </button>
  );
}
