import { useNavigate, useParams } from "react-router";
import ticketAssignedIcon from "../../../assets/icons/ticket-assigned.svg";
import ticketCanceledIcon from "../../../assets/icons/ticket-canceled.svg";
import ticketClosedIcon from "../../../assets/icons/ticket-closed.svg";
import ticketCreatedIcon from "../../../assets/icons/ticket-created.svg";
import ticketDoneIcon from "../../../assets/icons/ticket-done.svg";
import fireIcon from "../../../assets/icons/fire-icon.svg";
import ticketInWorkIcon from "../../../assets/icons/ticket-inwork.svg";
import { formatWorkDuration, resolveTicketDeadlineDisplay } from "../../dashboard/lib/dashboard-formatters";
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

const MOCK_WORK_RESULT =
    "Проведен профилактический визит, выполнена базовая очистка рабочих поверхностей и узлов. Проверены ключевые элементы, дефектов не выявлено, устройство работает стабильно.";
const MOCK_TICKET_HISTORY = [
    {
        id: "closed",
        title: "Закрыл тикет",
        date: "14.02.25",
        time: "14:40",
        icon: ticketClosedIcon,
    },
    {
        id: "worksDone",
        title: "Завершил работы",
        date: "14.02.25",
        time: "14:05",
        icon: ticketDoneIcon,
    },
    {
        id: "inWork",
        title: "Начал работы",
        date: "14.02.25",
        time: "12:20",
        icon: ticketInWorkIcon,
    },
    {
        id: "assigned",
        title: "Назначил",
        date: "14.02.25",
        time: "10:30",
        icon: ticketAssignedIcon,
    },
];

function formatMonthDay(value) {
    if (!value) {
        return null;
    }

    const date = new Date(value);
    if (Number.isNaN(date.getTime())) {
        return null;
    }

    const day = String(date.getDate()).padStart(2, "0");
    const month = String(date.getMonth() + 1).padStart(2, "0");
    return `${day}.${month}`;
}

function normalizePhoneValue(phone) {
    if (!phone) {
        return "";
    }

    return phone.replace(/[^\d+]/g, "");
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
    const deadlineDisplay = resolveTicketDeadlineDisplay(ticket);
    const reasonValue = ticket?.resolvedReason || "Не указано";
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
    const canOpenDevice = Boolean(ticket?.device);
    const canOpenClient = Boolean(ticket?.client);
    const phoneHrefValue = normalizePhoneValue(ticket?.contactPhone);
    const phoneHref = phoneHrefValue ? `tel:${phoneHrefValue}` : null;
    const emailHref = ticket?.contactEmail ? `mailto:${ticket.contactEmail}` : null;
    const workDuration = formatWorkDuration(ticket?.workstarted_at, ticket?.workfinished_at);
    const historyActorName = ticket?.executorName || "Имя Фамилия";

    function handleOpenDevice() {
        if (!ticket?.device) {
            return;
        }

        navigate(routePaths.deviceById(ticket.device));
    }

    function handleOpenClient() {
        if (!ticket?.client) {
            return;
        }

        navigate(routePaths.clientById(ticket.client));
    }

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
                        <span className="relative inline-flex h-8 w-8 items-center justify-center">
                            {isInWork ? <span className="ticket-inwork-ripple" aria-hidden="true" /> : null}
                            <img src={statusIcon} alt={ticket?.status || "status"} className="relative z-[1] h-8 w-8" />
                        </span>
                        {finishedDate ? <p className="text-sm font-semibold text-slate-100">{finishedDate}</p> : null}
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
                    <>
                        <div className="rounded-3xl border border-white/10 bg-white/5 p-6 text-sm text-slate-200">
                            <div className="flex items-start justify-between gap-4">
                                <h2 className="text-base font-semibold text-white">{reasonValue}</h2>
                                <p className="font-semibold text-white">{deadlineValue}</p>
                            </div>
                            <p className="mt-4 text-sm text-slate-200">{ticket.description || "Не указано"}</p>
                        </div>

                        <section className="space-y-3">
                            <h2 className="text-3xl font-semibold tracking-tight text-slate-300">Оборудование</h2>
                            <button
                                type="button"
                                onClick={handleOpenDevice}
                                disabled={!canOpenDevice}
                                className="flex w-full items-center gap-4 rounded-3xl border border-white/15 bg-white/10 p-5 text-left shadow-lg shadow-black/20 transition hover:border-white/30 disabled:cursor-not-allowed disabled:opacity-70"
                            >
                                <div className="min-w-0 flex-1">
                                    <p className="text-2xl font-semibold leading-tight text-slate-100">
                                        {ticket.deviceName || "Не указано"}
                                    </p>
                                    <p className="mt-4 text-2xl text-slate-400">
                                        С/Н: {ticket.deviceSerialNumber || "Не указано"}
                                    </p>
                                </div>
                                <svg
                                    xmlns="http://www.w3.org/2000/svg"
                                    viewBox="0 0 24 24"
                                    fill="none"
                                    stroke="currentColor"
                                    strokeWidth="3"
                                    strokeLinecap="round"
                                    strokeLinejoin="round"
                                    className="h-7 w-7 shrink-0 text-white"
                                    aria-hidden="true"
                                >
                                    <path d="M9 18l6-6-6-6" />
                                </svg>
                            </button>
                        </section>

                        <section className="space-y-3">
                            <h2 className="text-3xl font-semibold tracking-tight text-slate-300">Клиент</h2>

                            <button
                                type="button"
                                onClick={handleOpenClient}
                                disabled={!canOpenClient}
                                className="flex w-full items-center gap-4 rounded-3xl border border-white/15 bg-white/10 p-5 text-left shadow-lg shadow-black/20 transition hover:border-white/30 disabled:cursor-not-allowed disabled:opacity-70"
                            >
                                <div className="min-w-0 flex-1">
                                    <p className="text-2xl font-semibold leading-tight text-slate-100">
                                        {ticket.clientName || "Не указано"}
                                    </p>
                                    <p className="mt-2 text-2xl text-slate-400">{ticket.clientAddress || "Не указано"}</p>
                                </div>
                                <svg
                                    xmlns="http://www.w3.org/2000/svg"
                                    viewBox="0 0 24 24"
                                    fill="none"
                                    stroke="currentColor"
                                    strokeWidth="3"
                                    strokeLinecap="round"
                                    strokeLinejoin="round"
                                    className="h-7 w-7 shrink-0 text-white"
                                    aria-hidden="true"
                                >
                                    <path d="M9 18l6-6-6-6" />
                                </svg>
                            </button>

                            <div className="flex items-center gap-4 rounded-3xl border border-white/15 bg-white/10 p-5 text-left shadow-lg shadow-black/20">
                                <div className="min-w-0 flex-1">
                                    <p className="text-2xl font-semibold leading-tight text-slate-100">
                                        {ticket.contactName || "Не указано"}
                                    </p>
                                    <p className="mt-2 text-2xl text-slate-400">
                                        {ticket.contactPosition || "Не указано"}
                                    </p>
                                </div>
                                <div className="flex items-center gap-3">
                                    {phoneHref ? (
                                        <a
                                            href={phoneHref}
                                            aria-label={`Позвонить ${ticket.contactName || "контакту"}`}
                                            className="inline-flex h-16 w-16 items-center justify-center rounded-full border border-white/20 text-white transition hover:border-white/40 hover:bg-white/10"
                                        >
                                            <svg
                                                xmlns="http://www.w3.org/2000/svg"
                                                viewBox="0 0 24 24"
                                                fill="none"
                                                stroke="currentColor"
                                                strokeWidth="2"
                                                strokeLinecap="round"
                                                strokeLinejoin="round"
                                                className="h-8 w-8"
                                                aria-hidden="true"
                                            >
                                                <path d="M22 16.92v2a2 2 0 0 1-2.18 2 19.8 19.8 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.8 19.8 0 0 1-3.07-8.67A2 2 0 0 1 4.11 1h2a2 2 0 0 1 2 1.72c.12.9.32 1.79.59 2.64a2 2 0 0 1-.45 2.11L7.09 8.91a16 16 0 0 0 8 8l1.44-1.16a2 2 0 0 1 2.11-.45c.85.27 1.74.47 2.64.59A2 2 0 0 1 22 16.92z" />
                                                <path d="M15.5 5.5a5 5 0 0 1 3 3" />
                                                <path d="M15.5 1.5a9 9 0 0 1 7 7" />
                                            </svg>
                                        </a>
                                    ) : null}
                                    {emailHref ? (
                                        <a
                                            href={emailHref}
                                            aria-label={`Написать ${ticket.contactName || "контакту"}`}
                                            className="inline-flex h-16 w-16 items-center justify-center rounded-full border border-white/20 text-white transition hover:border-white/40 hover:bg-white/10"
                                        >
                                            <svg
                                                xmlns="http://www.w3.org/2000/svg"
                                                viewBox="0 0 24 24"
                                                fill="none"
                                                stroke="currentColor"
                                                strokeWidth="2"
                                                strokeLinecap="round"
                                                strokeLinejoin="round"
                                                className="h-8 w-8"
                                                aria-hidden="true"
                                            >
                                                <rect x="3" y="5" width="18" height="14" rx="2" />
                                                <path d="M3 7l9 6 9-6" />
                                            </svg>
                                        </a>
                                    ) : null}
                                </div>
                            </div>
                        </section>

                        <section className="space-y-3 rounded-3xl border border-emerald-300/25 bg-emerald-500/10 p-5 sm:p-6">
                            <div className="flex items-center justify-between gap-4">
                                <h2 className="text-3xl font-semibold tracking-tight text-emerald-100">Результат работы</h2>
                                <p className="inline-flex items-center gap-2 text-2xl font-semibold text-emerald-100">
                                    <svg
                                        xmlns="http://www.w3.org/2000/svg"
                                        viewBox="0 0 24 24"
                                        fill="none"
                                        stroke="currentColor"
                                        strokeWidth="2.2"
                                        strokeLinecap="round"
                                        strokeLinejoin="round"
                                        className="h-6 w-6"
                                        aria-hidden="true"
                                    >
                                        <circle cx="12" cy="12" r="8" />
                                        <path d="M12 8v5l3 2" />
                                    </svg>
                                    {workDuration}
                                </p>
                            </div>

                            <div className="rounded-2xl border border-white/15 bg-white/10 p-5 shadow-lg shadow-black/15">
                                <div className="flex items-start gap-4">
                                    <span className="inline-flex h-12 w-12 shrink-0 items-center justify-center rounded-full bg-slate-950 text-sm font-semibold text-slate-100">
                                        {ticket.executorName ? ticket.executorName.trim().charAt(0).toUpperCase() : "?"}
                                    </span>
                                    <div className="min-w-0">
                                        <p className="text-2xl font-semibold leading-tight text-slate-100">
                                            {ticket.executorName || "Исполнитель не назначен"}
                                        </p>
                                        <p className="text-2xl text-slate-400">
                                            {ticket.executorDepartment || "Отдел не указан"}
                                        </p>
                                    </div>
                                </div>

                                <p className="mt-4 text-2xl leading-relaxed text-slate-200">
                                    {ticket.result || MOCK_WORK_RESULT}
                                </p>
                            </div>

                            <div className="grid grid-cols-3 gap-2 sm:gap-3">
                                <div className="flex aspect-[4/3] items-center justify-center rounded-xl border border-white/15 bg-slate-900/25 text-sm font-semibold text-slate-200">
                                    Фото 1
                                </div>
                                <div className="flex aspect-[4/3] items-center justify-center rounded-xl border border-white/15 bg-slate-900/25 text-sm font-semibold text-slate-200">
                                    Фото 2
                                </div>
                                <div className="flex aspect-[4/3] items-center justify-center rounded-xl border border-white/15 bg-slate-900/25 text-sm font-semibold text-slate-200">
                                    Фото 3
                                </div>
                            </div>
                        </section>

                        <section className="space-y-5 rounded-3xl border border-white/15 bg-white/10 p-5 sm:p-6">
                            <h2 className="text-4xl font-semibold tracking-tight text-slate-300">История тикета</h2>
                            <div className="space-y-6">
                                {MOCK_TICKET_HISTORY.map((entry, index) => {
                                    const isLast = index === MOCK_TICKET_HISTORY.length - 1;

                                    return (
                                        <article key={entry.id} className="grid grid-cols-[1fr_auto] gap-4">
                                            <div className="flex min-w-0 items-start gap-4">
                                                <span className="mt-1 inline-flex h-12 w-12 shrink-0 rounded-full bg-slate-950" />
                                                <div className="min-w-0">
                                                    <p className="text-2xl text-slate-400">{entry.title}</p>
                                                    <p className="text-3xl font-semibold leading-tight text-slate-100">
                                                        {historyActorName}
                                                    </p>
                                                </div>
                                            </div>

                                            <div className="flex items-start gap-3">
                                                <div className="text-right">
                                                    <p className="text-2xl font-semibold text-slate-50">{entry.date}</p>
                                                    <p className="text-2xl text-slate-400">{entry.time}</p>
                                                </div>
                                                <div className="relative flex min-h-[4.25rem] w-6 justify-center">
                                                    {!isLast ? (
                                                        <span className="absolute left-1/2 top-7 h-[calc(100%-1.25rem)] w-0.5 -translate-x-1/2 bg-slate-400/60" />
                                                    ) : null}
                                                    <img src={entry.icon} alt="" className="relative z-[1] mt-1 h-6 w-6" />
                                                </div>
                                            </div>
                                        </article>
                                    );
                                })}
                            </div>
                        </section>
                    </>
                ) : null}
            </section>
        </PageShell>
    );
}
