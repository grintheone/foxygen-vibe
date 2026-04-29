import { Link } from "react-router";
import { routePaths } from "../../../../shared/config/routes";

export function TicketSummaryCard({ reasonValue, deadlineDisplay, description, referenceTicket }) {
    const deadlineValue = deadlineDisplay.shouldUseFireIcon ? (
        <span className="inline-flex items-center gap-1">
            <span aria-hidden="true">🔥</span>
            <span>{deadlineDisplay.dateValue}</span>
        </span>
    ) : deadlineDisplay.isFinishedDate || deadlineDisplay.isPlaceholder ? (
        deadlineDisplay.dateValue
    ) : (
        `до ${deadlineDisplay.dateValue}`
    );

    return (
        <section className="px-1">
            <p className="text-[16px] font-semibold tracking-tight text-[#BCC2CA] sm:text-[18px] lg:text-[20px]">Задача</p>
            <div className="mt-2 flex flex-wrap items-baseline gap-x-3 gap-y-1 text-[18px] font-semibold text-white">
                <h2 className="text-[18px] font-semibold leading-snug text-white">{reasonValue}</h2>
                <p className="text-[18px] font-semibold leading-snug text-white">{deadlineValue}</p>
            </div>
            <p className="mt-3 text-[16px] font-normal leading-relaxed text-slate-200">
                {description || "Не указано"}
            </p>
            {referenceTicket ? (
                <p className="mt-3 text-sm text-slate-300">
                    Создано из:{" "}
                    <Link
                        className="break-all text-sky-300 underline decoration-sky-300/60 underline-offset-2 transition hover:text-sky-200"
                        to={routePaths.ticketById(referenceTicket)}
                    >
                        {referenceTicket}
                    </Link>
                </p>
            ) : null}
        </section>
    );
}
