import fireIcon from "../../../../assets/icons/fire-icon.svg";

export function TicketSummaryCard({ reasonValue, deadlineDisplay, description }) {
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
        <div className="rounded-3xl border border-white/10 bg-white/5 p-6 text-sm text-slate-200">
            <div className="flex items-start justify-between gap-4">
                <h2 className="text-base font-semibold text-white">{reasonValue}</h2>
                <p className="font-semibold text-white">{deadlineValue}</p>
            </div>
            <p className="mt-4 text-sm text-slate-200">{description || "Не указано"}</p>
        </div>
    );
}
