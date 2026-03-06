import fireIcon from "../../../assets/icons/fire-icon.svg";
import { resolveTicketDeadlineDisplay, resolveTicketReason } from "../lib/dashboard-formatters";

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

  return (
    <button
      type="button"
      onClick={() => onOpenTicket(ticket.id)}
      className="relative w-full overflow-hidden rounded-2xl border border-cyan-200/25 bg-cyan-500/15 p-4 text-left shadow-lg transition hover:border-cyan-100/60"
    >
      <div className="grid gap-3 grid-cols-[1fr_auto]">
        <div className="space-y-1.5">
          <p className="text-sm font-semibold text-white">{reasonValue}</p>
          <p className="font-semibold text-slate-100">{ticket.deviceName}</p>
          <p className="text-sm text-slate-200/90">{ticket.clientName}</p>
        </div>
        <div className="flex flex-col justify-between">
          <div className="flex flex-col items-end justify-start">
            <p className="font-semibold text-white">{deadlineValue}</p>
            <p className="text-sm text-white">#{ticket.number}</p>
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
