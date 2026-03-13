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

  return (
    <button
      type="button"
      onClick={() => onOpenTicket(ticket.id)}
      className="relative w-full overflow-hidden rounded-3xl border border-white/10 bg-slate-950/35 p-5 text-left shadow-xl shadow-black/20 transition hover:border-white/20 hover:bg-slate-950/45"
    >
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0 flex-1 space-y-2">
          <p className="text-sm font-semibold text-cyan-100">{reasonValue}</p>
          <p className="text-xl font-semibold tracking-tight text-white">{ticket?.deviceName || "Устройство не указано"}</p>
          <p className="text-sm text-slate-300">{ticket?.clientName || "Клиент не указан"}</p>
        </div>

        <div className="flex flex-col items-end gap-2">
          <span
            className={`inline-flex items-center gap-2 rounded-full border px-3 py-1 text-xs font-semibold ${statusConfig.toneClassName}`}
          >
            <img src={statusConfig.icon} alt="" className="h-4 w-4" />
            <span>{statusConfig.label}</span>
          </span>
          <p className="text-sm font-semibold text-white">#{ticket?.number}</p>
          <p className="text-sm font-semibold text-slate-200">{deadlineValue}</p>
        </div>
      </div>

      <div className="mt-4 grid gap-2 border-t border-white/10 pt-4 text-sm text-slate-300">
        <p>{detailsValue || "Описание не указано"}</p>
        {ticket?.clientAddress ? <p className="text-slate-400">{ticket.clientAddress}</p> : null}
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
