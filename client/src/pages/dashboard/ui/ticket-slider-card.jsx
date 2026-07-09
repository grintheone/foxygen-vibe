import ticketDoneIcon from "../../../assets/icons/ticket-done.svg";
import ticketInWorkIcon from "../../../assets/icons/ticket-inwork.svg";
import { formatWorkDuration } from "../lib/dashboard-formatters";

const sliderStatusConfigByType = {
  inWork: {
    cardClassName: "border-emerald-300/35 bg-emerald-500/30 hover:border-emerald-100/65 hover:bg-emerald-500/35",
    footerClassName: "border-emerald-950/30 bg-emerald-950/50",
    icon: ticketInWorkIcon,
    markerClassName: "bg-emerald-50",
    pingClassName: "bg-emerald-100",
    statusLabel: "На выезде",
    toneBlockClassName: "bg-emerald-700/35",
  },
  worksDone: {
    cardClassName: "border-fuchsia-300/35 bg-fuchsia-500/30 hover:border-fuchsia-100/65 hover:bg-fuchsia-500/35",
    footerClassName: "border-fuchsia-950/30 bg-fuchsia-950/50",
    icon: ticketDoneIcon,
    markerClassName: "bg-fuchsia-50",
    pingClassName: "bg-fuchsia-100",
    statusLabel: "Работы завершены",
    toneBlockClassName: "bg-fuchsia-700/35",
  },
};

function resolveSliderStatusConfig(status) {
  return (
    sliderStatusConfigByType[status] || {
      cardClassName: "border-slate-300/25 bg-slate-500/20 hover:border-white/50 hover:bg-slate-500/25",
      footerClassName: "border-slate-950/25 bg-slate-950/40",
      icon: ticketInWorkIcon,
      markerClassName: "bg-slate-50",
      pingClassName: "bg-slate-100",
      statusLabel: "Статус",
      toneBlockClassName: "bg-slate-700/25",
    }
  );
}

export function TicketSliderCard({ ticket, onOpenTicket }) {
  const isInWork = ticket.status === "inWork";
  const statusConfig = resolveSliderStatusConfig(ticket.status);
  const workValue = isInWork ? "В процессе" : formatWorkDuration(ticket.workstarted_at, ticket.workfinished_at);

  return (
    <button
      type="button"
      onClick={() => onOpenTicket(ticket.id)}
      className={`w-full overflow-hidden rounded-lg border text-left text-white shadow-xl shadow-black/20 backdrop-blur transition focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-white/60 ${statusConfig.cardClassName}`}
    >
      <div className="px-4 py-3.5">
        <div className="flex min-w-0 items-center gap-3">
          <p
            className={`-ml-4 inline-flex min-w-0 items-center gap-2 rounded-r-full py-1 pl-4 pr-3 text-sm font-semibold leading-5 ${statusConfig.toneBlockClassName}`}
          >
            <img src={statusConfig.icon} alt="" className="h-4 w-4 shrink-0" />
            <span className="truncate">{statusConfig.statusLabel}</span>
          </p>

          <p className="inline-flex shrink-0 items-center gap-2 text-sm font-semibold leading-5">
            {isInWork ? (
              <span className="relative flex h-2.5 w-2.5" aria-hidden="true">
                <span className={`absolute inline-flex h-full w-full animate-ping rounded-full opacity-80 ${statusConfig.pingClassName}`} />
                <span className={`relative inline-flex h-2.5 w-2.5 rounded-full ${statusConfig.markerClassName}`} />
              </span>
            ) : null}
            <span>{workValue}</span>
          </p>

          <p className="ml-auto shrink-0 text-sm font-semibold leading-5">#{ticket.number}</p>
        </div>

        <div className="mt-5 flex flex-col gap-2">
          <p className="text-lg font-semibold leading-6">{ticket.deviceName || "Устройство не указано"}</p>
          <p className="text-lg font-semibold leading-6">С/Н:&nbsp;&nbsp;{ticket.deviceSerialNumber || "Не указан"}</p>
        </div>
      </div>

      <div className={`border-t px-4 py-3 ${statusConfig.footerClassName}`}>
        <p className="text-lg font-semibold leading-6">{ticket.clientName || "Клиент не указан"}</p>
        {ticket.clientAddress ? <p className="mt-1 text-lg leading-6 text-slate-100/95">{ticket.clientAddress}</p> : null}
      </div>
    </button>
  );
}
