export function DashboardHeader({ department, onOpenProfile }) {
  const today = new Intl.DateTimeFormat("ru-RU", {
    day: "numeric",
    month: "long",
  }).format(new Date());

  return (
    <header className="bg-transparent px-1 pt-2">
      <div className="flex items-start justify-between gap-4 sm:gap-6 lg:gap-8">
        <div className="min-w-0">
          <p className="text-sm font-semibold tracking-[0.18em] text-slate-300 uppercase sm:text-base lg:text-lg xl:text-xl">
            {today}
          </p>
          <h1 className="mt-2 truncate text-[32px] font-semibold leading-none text-white sm:text-[36px] lg:text-[44px] xl:text-[52px]">
            {department || "Отдел не указан"}
          </h1>
        </div>
        <button
          type="button"
          onClick={onOpenProfile}
          aria-label="Открыть профиль"
          className="inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl bg-[#2F3545] text-[#94A3B8] transition hover:bg-[#394055] sm:h-12 sm:w-12 lg:h-14 lg:w-14"
        >
          <svg
            xmlns="http://www.w3.org/2000/svg"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="1.8"
            className="h-5 w-5 sm:h-6 sm:w-6 lg:h-7 lg:w-7"
          >
            <circle cx="12" cy="8" r="3.6" />
            <path d="M4.5 19.2C5.9 15.9 8.6 14.4 12 14.4s6.1 1.5 7.5 4.8" />
          </svg>
        </button>
      </div>
    </header>
  );
}
