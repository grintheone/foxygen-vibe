import fireIcon from "../../assets/icons/fire-icon.svg";
import ticketAssignedIcon from "../../assets/icons/ticket-assigned.svg";
import ticketCanceledIcon from "../../assets/icons/ticket-canceled.svg";
import ticketClosedIcon from "../../assets/icons/ticket-closed.svg";
import ticketCreatedIcon from "../../assets/icons/ticket-created.svg";
import ticketDoneIcon from "../../assets/icons/ticket-done.svg";
import ticketInWorkIcon from "../../assets/icons/ticket-inwork.svg";
import { resolveTicketDeadlineDisplay, resolveTicketReason } from "../../pages/dashboard/lib/dashboard-formatters";

const statusConfigByType = {
  assigned: {
    icon: ticketAssignedIcon,
    label: "Назначен",
    toneClassName: "border-cyan-200/30 bg-cyan-400/15 text-cyan-50",
  },
  canceled: {
    icon: ticketCanceledIcon,
    label: "Отменен",
    toneClassName: "border-rose-200/30 bg-rose-500/15 text-rose-50",
  },
  cancelled: {
    icon: ticketCanceledIcon,
    label: "Отменен",
    toneClassName: "border-rose-200/30 bg-rose-500/15 text-rose-50",
  },
  closed: {
    icon: ticketClosedIcon,
    label: "Закрыт",
    toneClassName: "border-emerald-200/30 bg-emerald-500/15 text-emerald-50",
  },
  created: {
    icon: ticketCreatedIcon,
    label: "Создан",
    toneClassName: "border-slate-200/20 bg-white/10 text-slate-100",
  },
  inWork: {
    icon: ticketInWorkIcon,
    label: "На выезде",
    toneClassName: "border-violet-200/30 bg-violet-500/20 text-violet-50",
  },
  worksDone: {
    icon: ticketDoneIcon,
    label: "Работы завершены",
    toneClassName: "border-emerald-200/30 bg-emerald-500/15 text-emerald-50",
  },
};

function resolveStatusConfig(status) {
  return (
    statusConfigByType[status] || {
      icon: ticketAssignedIcon,
      label: status || "Статус",
      toneClassName: "border-slate-200/20 bg-white/10 text-slate-100",
    }
  );
}

export function ProfileTicketCard({ ticket, onOpenTicket }) {
  const reasonValue = resolveTicketReason(ticket);
  const deadlineDisplay = resolveTicketDeadlineDisplay(ticket);
  const statusConfig = resolveStatusConfig(ticket?.status);
  const detailsValue = ticket?.status === "closed" ? ticket?.result : ticket?.description;
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
  const shouldShowGradient = !deadlineDisplay.isFinishedDate && (deadlineDisplay.isOverdue || ticket?.urgent);
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
      className="relative w-full overflow-hidden rounded-lg border border-slate-400/20 bg-[#2f3748] px-4 py-3.5 text-left shadow-xl shadow-black/20 transition hover:border-slate-300/35 hover:bg-[#333c4f]"
    >
      <div className="grid grid-cols-[1fr_auto] gap-3">
        <div className="min-w-0 space-y-1.5">
          <div className="flex min-w-0 items-center gap-2">
            <img src={statusConfig.icon} alt="" className="h-5 w-5 shrink-0" />
            <p className="truncate text-sm font-semibold text-slate-100">{reasonValue}</p>
          </div>
          <p className="text-base font-semibold text-white">{ticket?.deviceName || "Устройство не указано"}</p>
          <p className="text-sm text-slate-300">{detailsValue || "Описание не указано"}</p>
        </div>

        <div className="flex flex-col items-end">
          <p className="text-sm font-semibold text-slate-200">{deadlineValue}</p>
          <p className="text-sm font-semibold text-slate-200/80">#{ticket?.number}</p>
        </div>
      </div>

      <div className="mt-3 flex items-end justify-between gap-3 border-t border-slate-400/10 pt-3">
        <div className="min-w-0 space-y-1.5 text-sm text-slate-300">
          <p className="truncate">{ticket?.clientName || "Клиент не указан"}</p>
          {ticket?.clientAddress ? <p className="truncate text-slate-200/80">{ticket.clientAddress}</p> : null}
        </div>

        {ticket?.urgent ? (
          <span
            className={`shrink-0 rounded-full border px-3 py-1 text-[10px] font-bold uppercase tracking-[0.12em] ${urgencyBadgeClassName}`}
          >
            Срочно
          </span>
        ) : null}
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
