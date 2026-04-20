export function DashboardHeader({ onOpenProfile }) {
  const today = new Intl.DateTimeFormat("ru-RU", {
    day: "numeric",
    month: "long",
  }).format(new Date());

  return (
    <header className="rounded-3xl border border-white/10 bg-slate-950/35 p-6 shadow-xl shadow-black/20 backdrop-blur">
      <div className="flex justify-between items-center gap-4">
        <div>
          <h1 className="text-base font-semibold tracking-[0.2em] text-slate-300 uppercase sm:text-lg">Сегодня {today}</h1>
        </div>
        <button
          type="button"
          onClick={onOpenProfile}
          aria-label="Открыть профиль"
          className="inline-flex h-11 w-11 items-center justify-center rounded-2xl border border-white/15 bg-[#6A3BF2] text-white shadow-lg shadow-[#6A3BF2]/35 transition hover:bg-[#7C52F5]"
        >
          <svg
            xmlns="http://www.w3.org/2000/svg"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="1.8"
            className="h-5 w-5"
          >
            <circle cx="12" cy="8" r="3.6" />
            <path d="M4.5 19.2C5.9 15.9 8.6 14.4 12 14.4s6.1 1.5 7.5 4.8" />
          </svg>
        </button>
      </div>
    </header>
  );
}
