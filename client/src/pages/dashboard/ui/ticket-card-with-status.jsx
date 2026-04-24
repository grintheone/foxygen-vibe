import fireIcon from "../../../assets/icons/fire-icon.svg";
import ticketAssignedIcon from "../../../assets/icons/ticket-assigned.svg";
import ticketCanceledIcon from "../../../assets/icons/ticket-canceled.svg";
import ticketClosedIcon from "../../../assets/icons/ticket-closed.svg";
import ticketCreatedIcon from "../../../assets/icons/ticket-created.svg";
import ticketDoneIcon from "../../../assets/icons/ticket-done.svg";
import ticketInWorkIcon from "../../../assets/icons/ticket-inwork.svg";
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

export function TicketCardWithStatus({ ticket, onOpenTicket }) {
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
  const shouldShowBadge = ticket.urgent;
  const shouldShowGradient = !deadlineDisplay.isFinishedDate && (deadlineDisplay.isOverdue || ticket.urgent);
  const badgeClassName = deadlineDisplay.isOverdue
    ? "border-rose-200/40 bg-rose-500/25 text-rose-50"
    : "border-cyan-200/40 bg-cyan-500/25 text-cyan-50";
  const gradientClassName = deadlineDisplay.isOverdue
    ? "from-rose-500/0 via-rose-400/80 to-rose-300/0"
    : "from-cyan-500/0 via-cyan-400/80 to-cyan-300/0";
  const statusIcon = statusIconByType[ticket.status] || ticketAssignedIcon;

  return (
    <button
      type="button"
      onClick={() => onOpenTicket(ticket.id)}
      className="relative w-full overflow-hidden rounded-lg border border-slate-400/20 bg-[#2f3748] px-4 py-3.5 text-left shadow-lg shadow-black/20 transition hover:border-slate-300/35 hover:bg-[#333c4f]"
    >
      <div className="grid grid-cols-[1fr_auto] gap-3">
        <div className="min-w-0 space-y-1.5">
          <div className="flex min-w-0 items-center gap-2">
            <img src={statusIcon} alt="" className="h-5 w-5 shrink-0" />
            <p className="truncate text-sm font-semibold text-slate-100">{reasonValue}</p>
          </div>
          <p className="text-base font-semibold text-white">{ticket.deviceName}</p>
          <p className="text-sm text-slate-300">{ticket.clientName}</p>
        </div>
        <div className="flex flex-col items-end justify-between">
          <div className="flex flex-col items-end">
            <p className="text-sm font-semibold text-slate-200">{deadlineValue}</p>
            <p className="text-sm font-semibold text-slate-200/80">#{ticket.number}</p>
          </div>
          {shouldShowBadge ? (
            <span
              className={`rounded-md border px-2 py-0.5 text-[10px] font-bold uppercase tracking-[0.08em] ${badgeClassName}`}
            >
              СРОЧНО
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
    </button>
  );
}
