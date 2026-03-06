export function TicketNavigationCard({ value, subtitle, onClick, disabled }) {
    return (
        <button
            type="button"
            onClick={onClick}
            disabled={disabled}
            className="flex w-full items-center gap-4 rounded-3xl border border-white/15 bg-white/10 p-5 text-left shadow-lg shadow-black/20 transition hover:border-white/30 disabled:cursor-not-allowed disabled:opacity-70"
        >
            <div className="min-w-0 flex-1">
                <p className="text-2xl font-semibold leading-tight text-slate-100">{value || "Не указано"}</p>
                <p className="mt-2 text-2xl text-slate-400">{subtitle || "Не указано"}</p>
            </div>
            <svg
                xmlns="http://www.w3.org/2000/svg"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="3"
                strokeLinecap="round"
                strokeLinejoin="round"
                className="h-7 w-7 shrink-0 text-white"
                aria-hidden="true"
            >
                <path d="M9 18l6-6-6-6" />
            </svg>
        </button>
    );
}
