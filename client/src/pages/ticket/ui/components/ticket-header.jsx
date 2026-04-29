function BackButton({ onClick }) {
    return (
        <button
            type="button"
            onClick={onClick}
            aria-label="Назад"
            className="inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl bg-[#2F3545] text-[#94A3B8] transition hover:bg-[#394055] sm:h-12 sm:w-12 lg:h-14 lg:w-14"
        >
            <svg
                xmlns="http://www.w3.org/2000/svg"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="1.8"
                strokeLinecap="round"
                strokeLinejoin="round"
                className="h-5 w-5 sm:h-6 sm:w-6 lg:h-7 lg:w-7"
                aria-hidden="true"
            >
                <path d="M15 18l-6-6 6-6" />
            </svg>
        </button>
    );
}

export function TicketHeader({ title, ticketNumber, onBack }) {
    const headerTitle = title || `Тикет #${ticketNumber}`;

    return (
        <header className="bg-transparent px-1 pt-2">
            <div className="grid grid-cols-[auto_1fr_auto] items-center gap-4 sm:gap-6 lg:gap-8">
                <BackButton onClick={onBack} />
                <h1 className="justify-self-center text-center text-sm font-semibold tracking-[0.18em] text-[#94A3B8] sm:text-base lg:text-lg xl:text-xl">
                    {headerTitle}
                </h1>
                <div className="h-11 w-11 shrink-0 sm:h-12 sm:w-12 lg:h-14 lg:w-14" aria-hidden="true" />
            </div>
        </header>
    );
}
