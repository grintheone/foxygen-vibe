export function TicketHeader({ ticketNumber, isInWork, statusIcon, statusAlt, finishedDate, onBack }) {
    return (
        <header className="grid grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)] items-center rounded-3xl border border-white/10 bg-slate-950/35 p-6 shadow-xl shadow-black/20 backdrop-blur">
            <div className="justify-self-start">
                <button
                    type="button"
                    onClick={onBack}
                    aria-label="Назад в дэшборд"
                    className="inline-flex h-11 w-11 items-center justify-center rounded-2xl bg-[#6A3BF2] text-white transition hover:bg-[#7C52F5]"
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
            </div>

            <div className="justify-self-center text-center">
                <p className="text-xs font-semibold uppercase tracking-[0.3em] text-slate-400">Тикет</p>
                <h1 className="mt-3 text-3xl font-bold tracking-tight sm:text-4xl">Заявка #{ticketNumber}</h1>
            </div>

            <div className="flex items-center justify-self-end gap-2">
                <span className="relative inline-flex h-8 w-8 items-center justify-center">
                    {isInWork ? <span className="ticket-inwork-ripple" aria-hidden="true" /> : null}
                    <img src={statusIcon} alt={statusAlt} className="relative z-[1] h-8 w-8" />
                </span>
                {finishedDate ? <p className="text-sm font-semibold text-slate-100">{finishedDate}</p> : null}
            </div>
        </header>
    );
}
