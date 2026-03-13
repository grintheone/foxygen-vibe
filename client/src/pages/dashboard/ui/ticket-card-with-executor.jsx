import fireIcon from "../../../assets/icons/fire-icon.svg";
import ticketAssignedIcon from "../../../assets/icons/ticket-assigned.svg";
import ticketCanceledIcon from "../../../assets/icons/ticket-canceled.svg";
import ticketClosedIcon from "../../../assets/icons/ticket-closed.svg";
import ticketCreatedIcon from "../../../assets/icons/ticket-created.svg";
import ticketDoneIcon from "../../../assets/icons/ticket-done.svg";
import ticketInWorkIcon from "../../../assets/icons/ticket-inwork.svg";
import { resolveTicketDeadlineDisplay, resolveTicketReason } from "../lib/dashboard-formatters";

function PersonIcon({ className }) {
    return (
        <svg
            xmlns="http://www.w3.org/2000/svg"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="1.8"
            className={className}
            aria-hidden="true"
        >
            <circle cx="12" cy="8" r="3.6" />
            <path d="M4.5 19.2C5.9 15.9 8.6 14.4 12 14.4s6.1 1.5 7.5 4.8" />
        </svg>
    );
}

const statusIconByType = {
    assigned: ticketAssignedIcon,
    canceled: ticketCanceledIcon,
    cancelled: ticketCanceledIcon,
    closed: ticketClosedIcon,
    created: ticketCreatedIcon,
    inWork: ticketInWorkIcon,
    worksDone: ticketDoneIcon,
};

export function TicketCardWithExecutor({ ticket, executor, onOpenTicket }) {
    const reasonValue = resolveTicketReason(ticket);
    const deadlineDisplay = resolveTicketDeadlineDisplay(ticket);
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
    const statusIcon = statusIconByType[ticket.status] || ticketAssignedIcon;
    const detailsValue = ticket.status === "closed" ? ticket.result : ticket.description;
    const shouldShowGradient = !deadlineDisplay.isFinishedDate && (deadlineDisplay.isOverdue || ticket.urgent);
    const shouldShowUrgencyBadge = !deadlineDisplay.isFinishedDate && ticket.urgent;
    const gradientClassName = deadlineDisplay.isOverdue
        ? "from-rose-500/0 via-rose-400/80 to-rose-300/0"
        : "from-cyan-500/0 via-cyan-400/80 to-cyan-300/0";
    const urgencyBadgeClassName = deadlineDisplay.isOverdue
        ? "border-rose-200/30 bg-rose-500/20 text-rose-50"
        : "border-cyan-200/30 bg-cyan-400/20 text-cyan-50";

    return (
        <button
            type="button"
            onClick={() => onOpenTicket(ticket.id)}
            className="relative w-full overflow-hidden rounded-3xl border border-white/10 bg-slate-950/35 text-left shadow-xl shadow-black/20 transition hover:border-white/20 hover:bg-slate-950/45"
        >
            <div className="grid grid-cols-[1fr_auto] gap-3 p-5">
                <div className="space-y-2">
                    <p className="text-sm font-semibold text-cyan-100">{reasonValue}</p>
                    <p className="text-xl font-semibold tracking-tight text-white">{ticket.deviceName}</p>
                    <p className="text-sm text-slate-300">{detailsValue || "Не указано"}</p>
                </div>
                <div className="flex flex-col items-end justify-between gap-2">
                    <p className="text-sm font-semibold text-slate-200">{deadlineValue}</p>
                    <p className="text-sm font-semibold text-white">#{ticket.number}</p>
                    <img src={statusIcon} alt={ticket.status || "status"} className="h-6 w-6" />
                </div>
            </div>

            <div className="border-t border-white/10 bg-white/5 px-5 py-4">
                <div className="flex items-center gap-3">
                    {executor?.avatarUrl ? (
                        <img
                            src={executor.avatarUrl}
                            alt={executor.name}
                            className="h-10 w-10 rounded-full object-cover"
                        />
                    ) : (
                        <div className="flex h-10 w-10 items-center justify-center rounded-full bg-slate-100 text-slate-500">
                            <PersonIcon className="h-5 w-5" />
                        </div>
                    )}

                    <div className="min-w-0">
                        <p className="truncate text-base font-semibold text-slate-100">
                            {executor?.name || "Исполнитель не назначен"}
                        </p>
                        <p className="truncate text-sm text-slate-200/80">
                            {executor?.department || "Отдел не указан"}
                        </p>
                    </div>
                </div>
            </div>

            {shouldShowUrgencyBadge ? (
                <span
                    className={`absolute right-4 bottom-4 rounded-full border px-3 py-1 text-[10px] font-bold uppercase tracking-[0.12em] ${urgencyBadgeClassName}`}
                >
                    Срочно
                </span>
            ) : null}

            {shouldShowGradient ? (
                <span
                    aria-hidden="true"
                    className={`pointer-events-none absolute inset-x-0 bottom-0 h-[3px] rounded-full bg-gradient-to-r ${gradientClassName}`}
                />
            ) : null}
        </button>
    );
}
