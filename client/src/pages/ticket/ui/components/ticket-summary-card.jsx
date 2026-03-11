import { Link } from "react-router";
import fireIcon from "../../../../assets/icons/fire-icon.svg";
import { routePaths } from "../../../../shared/config/routes";
import { ticketSurfaceClassName } from "./ticket-surface";

export function TicketSummaryCard({ reasonValue, deadlineDisplay, description, referenceTicket }) {
    const deadlineValue = deadlineDisplay.shouldUseFireIcon ? (
        <span className="inline-flex items-center gap-1">
            <img src={fireIcon} alt="" className="h-4 w-4" />
            <span>{deadlineDisplay.dateValue}</span>
        </span>
    ) : deadlineDisplay.isFinishedDate || deadlineDisplay.isPlaceholder ? (
        deadlineDisplay.dateValue
    ) : (
        `до ${deadlineDisplay.dateValue}`
    );

    return (
        <div className={`${ticketSurfaceClassName} p-6 text-sm text-slate-200`}>
            <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between sm:gap-6">
                <div className="min-w-0">
                    <p className="text-[11px] font-semibold uppercase tracking-[0.24em] text-slate-400">Сводка</p>
                    <h2 className="mt-3 text-2xl font-semibold leading-tight text-white sm:text-3xl">{reasonValue}</h2>
                </div>
                <p className="text-sm font-semibold text-slate-100 sm:pt-1 sm:text-base">{deadlineValue}</p>
            </div>
            {referenceTicket ? (
                <p className="mt-4 text-sm text-slate-300 sm:text-base">
                    Создано из:{" "}
                    <Link
                        className="break-all text-sky-300 underline decoration-sky-300/60 underline-offset-2 transition hover:text-sky-200"
                        to={routePaths.ticketById(referenceTicket)}
                    >
                        {referenceTicket}
                    </Link>
                </p>
            ) : null}
            <p className="mt-4 text-sm leading-relaxed text-slate-200 sm:text-base">{description || "Не указано"}</p>
        </div>
    );
}
