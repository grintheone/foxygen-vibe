import { useNavigate, useParams } from "react-router";
import ticketAssignedIcon from "../../../assets/icons/ticket-assigned.svg";
import ticketCanceledIcon from "../../../assets/icons/ticket-canceled.svg";
import ticketClosedIcon from "../../../assets/icons/ticket-closed.svg";
import ticketCreatedIcon from "../../../assets/icons/ticket-created.svg";
import ticketDoneIcon from "../../../assets/icons/ticket-done.svg";
import ticketInWorkIcon from "../../../assets/icons/ticket-inwork.svg";
import { routePaths } from "../../../shared/config/routes";
import { useGetTicketByIdQuery } from "../../../shared/api/tickets-api";
import { PageShell } from "../../../shared/ui/page-shell";

const statusIconByType = {
    assigned: ticketAssignedIcon,
    canceled: ticketCanceledIcon,
    cancelled: ticketCanceledIcon,
    closed: ticketClosedIcon,
    created: ticketCreatedIcon,
    inWork: ticketInWorkIcon,
    worksDone: ticketDoneIcon,
};

function formatMonthDay(value) {
    if (!value) {
        return null;
    }

    const date = new Date(value);
    if (Number.isNaN(date.getTime())) {
        return null;
    }

    const month = String(date.getMonth() + 1).padStart(2, "0");
    const day = String(date.getDate()).padStart(2, "0");
    return `${month}.${day}`;
}

export function TicketPage() {
    const navigate = useNavigate();
    const { ticketId } = useParams();
    const {
        data: ticket,
        isError,
        isFetching,
        isLoading,
    } = useGetTicketByIdQuery(ticketId, {
        skip: !ticketId,
    });
    const ticketNumber = ticket?.number ?? "—";
    const statusIcon = statusIconByType[ticket?.status] || ticketAssignedIcon;
    const finishedDate = formatMonthDay(ticket?.workfinished_at);
    const isInWork = ticket?.status === "inWork";

    return (
        <PageShell>
            <section className="w-full space-y-6">
                <header className="grid grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)] items-center rounded-3xl border border-white/10 bg-slate-950/35 p-6 shadow-xl shadow-black/20 backdrop-blur">
                    <div className="justify-self-start">
                        <button
                            type="button"
                            onClick={() => navigate(routePaths.dashboard)}
                            aria-label="Назад в дэшборд"
                            className="inline-flex h-11 w-11 items-center justify-center rounded-2xl bg-[#6A3BF2] text-white transition hover:bg-[#7C52F5]"
                        >
                            <svg
                                xmlns="http://www.w3.org/2000/svg"
                                viewBox="0 0 24 24"
                                fill="none"
                                stroke="currentColor"
                                strokeWidth="2.5"
                                strokeLinecap="round"
                                strokeLinejoin="round"
                                className="h-5 w-5"
                                aria-hidden="true"
                            >
                                <path d="M15 18l-6-6 6-6" />
                            </svg>
                        </button>
                    </div>

                    <div className="justify-self-center text-center">
                        <p className="text-xs font-semibold uppercase tracking-[0.3em] text-slate-400">Тикет</p>
                        <h1 className="mt-3 text-3xl font-bold tracking-tight sm:text-4xl">Заявка #{ticketNumber}</h1>
                    </div>

                    <div className="flex items-center justify-self-end gap-2">
                        {finishedDate ? <p className="text-sm font-semibold text-slate-100">{finishedDate}</p> : null}
                        <span className="relative inline-flex h-8 w-8 items-center justify-center">
                            {isInWork ? <span className="ticket-inwork-ripple" aria-hidden="true" /> : null}
                            <img src={statusIcon} alt={ticket?.status || "status"} className="relative z-[1] h-8 w-8" />
                        </span>
                    </div>
                </header>

                {isLoading || isFetching ? (
                    <div className="rounded-3xl border border-white/10 bg-white/5 p-6">
                        <p className="text-sm text-slate-300">Загрузка тикета...</p>
                    </div>
                ) : null}

                {isError ? (
                    <div className="rounded-3xl border border-rose-300/30 bg-rose-500/10 p-6">
                        <p className="text-sm text-rose-100">Не удалось загрузить тикет.</p>
                    </div>
                ) : null}

                {!isLoading && !isFetching && !isError && ticket ? (
                    <div className="rounded-3xl border border-white/10 bg-white/5 p-6 text-sm text-slate-200">
                        <p>Детальная страница тикета #{ticketNumber} (временный экран).</p>
                        <p className="mt-3">Статус: {ticket.statusTitle || ticket.status || "Не указано"}</p>
                        <p className="mt-1">Причина: {ticket.resolvedReason || "Не указано"}</p>
                        <p className="mt-1">Клиент: {ticket.clientName || "Не указано"}</p>
                        <p className="mt-1">Устройство: {ticket.deviceName || "Не указано"}</p>
                    </div>
                ) : null}
            </section>
        </PageShell>
    );
}
