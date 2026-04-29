import ticketAssignedIcon from "../../../assets/icons/ticket-assigned.svg";
import ticketCanceledIcon from "../../../assets/icons/ticket-canceled.svg";
import ticketClosedIcon from "../../../assets/icons/ticket-closed.svg";
import ticketCreatedIcon from "../../../assets/icons/ticket-created.svg";
import ticketDoneIcon from "../../../assets/icons/ticket-done.svg";
import ticketInWorkIcon from "../../../assets/icons/ticket-inwork.svg";
import { UserAvatar } from "../../../shared/ui/user-avatar";
import { resolveTicketDeadlineDisplay, resolveTicketReason } from "../lib/dashboard-formatters";

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
            <span aria-hidden="true">🔥</span>
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

    function handleCardKeyDown(event) {
        if (event.key !== "Enter" && event.key !== " ") {
            return;
        }

        event.preventDefault();
        onOpenTicket(ticket.id);
    }

    return (
        <article
            role="button"
            tabIndex={0}
            onClick={() => onOpenTicket(ticket.id)}
            onKeyDown={handleCardKeyDown}
            className="relative w-full cursor-pointer overflow-hidden rounded-lg border border-slate-400/20 bg-[#2f3748] text-left shadow-xl shadow-black/20 transition hover:border-slate-300/35 hover:bg-[#333c4f] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-white/60"
        >
            <div className="grid grid-cols-[1fr_auto] gap-3 px-4 py-3.5">
                <div className="space-y-1.5">
                    <div className="flex min-w-0 items-center gap-2">
                        <img src={statusIcon} alt={ticket.status || "status"} className="h-5 w-5 shrink-0" />
                        <p className="truncate text-sm font-semibold text-slate-100">{reasonValue}</p>
                    </div>
                    <p className="text-base font-semibold text-white">{ticket.deviceName}</p>
                    <p className="text-sm text-slate-300">{detailsValue || "Не указано"}</p>
                </div>
                <div className="flex flex-col items-end">
                    <p className="text-sm font-semibold text-slate-200">{deadlineValue}</p>
                    <p className="text-sm font-semibold text-slate-200/80">#{ticket.number}</p>
                </div>
            </div>

            <div className="border-t border-slate-400/10 bg-[#3f485a] px-4 py-3">
                <div className="flex items-end justify-between gap-3">
                    <div className="flex min-w-0 items-center gap-3">
                        <UserAvatar
                            avatarUrl={executor?.avatarUrl}
                            userId={executor?.id}
                            name={executor?.name}
                            className="h-10 w-10"
                            stopPropagation
                        />

                        <div className="min-w-0">
                            <p className="truncate text-base font-semibold text-slate-100">
                                {executor?.name || "Исполнитель не назначен"}
                            </p>
                            <p className="truncate text-sm text-slate-200/80">
                                {executor?.department || "Отдел не указан"}
                            </p>
                        </div>
                    </div>

                    {shouldShowUrgencyBadge ? (
                        <span
                            className={`shrink-0 rounded-full border px-3 py-1 text-[10px] font-bold uppercase tracking-[0.12em] ${urgencyBadgeClassName}`}
                        >
                            Срочно
                        </span>
                    ) : null}
                </div>
            </div>

            {shouldShowGradient ? (
                <span
                    aria-hidden="true"
                    className={`pointer-events-none absolute inset-x-0 bottom-0 h-[3px] rounded-full bg-gradient-to-r ${gradientClassName}`}
                />
            ) : null}
        </article>
    );
}
