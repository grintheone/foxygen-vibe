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
            <div className="flex items-start gap-2 text-xl font-semibold text-white">
                {reasonValue} {deadlineValue}
            </div>
            {referenceTicket ? (
                <p className="mt-4 text-sm text-slate-300">
                    Создано из:{" "}
                    <Link
                        className="break-all text-sky-300 underline decoration-sky-300/60 underline-offset-2 transition hover:text-sky-200"
                        to={routePaths.ticketById(referenceTicket)}
                    >
                        {referenceTicket}
                    </Link>
                </p>
            ) : null}
            <p className="mt-4 text-sm text-slate-200">{description || "Не указано"}</p>
        </div>
    );
}
