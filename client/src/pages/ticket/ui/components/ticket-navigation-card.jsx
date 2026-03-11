import { ticketSurfaceInteractiveClassName } from "./ticket-surface";

export function TicketNavigationCard({ value, subtitle, onClick, disabled }) {
    return (
        <button
            type="button"
            onClick={onClick}
            disabled={disabled}
            className={`${ticketSurfaceInteractiveClassName} flex w-full items-center gap-4 p-5 text-left disabled:cursor-not-allowed disabled:opacity-70`}
        >
            <div className="min-w-0 flex-1">
                <p className="text-lg font-semibold leading-tight text-slate-100 sm:text-2xl">{value || "Не указано"}</p>
                <p className="mt-2 text-sm text-slate-400 sm:text-xl">{subtitle || "Не указано"}</p>
            </div>
            <svg
                xmlns="http://www.w3.org/2000/svg"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="3"
                strokeLinecap="round"
                strokeLinejoin="round"
                className="h-6 w-6 shrink-0 text-white sm:h-7 sm:w-7"
                aria-hidden="true"
            >
                <path d="M9 18l6-6-6-6" />
            </svg>
        </button>
    );
}
