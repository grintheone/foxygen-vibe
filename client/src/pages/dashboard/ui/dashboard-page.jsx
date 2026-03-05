import { useMemo, useRef, useState } from "react";
import { useNavigate } from "react-router";
import { useAuth } from "../../../features/auth";
import { routePaths } from "../../../shared/config/routes";
import { PageShell } from "../../../shared/ui/page-shell";

const MOCK_EXECUTOR_ID = "01eaedd4-1bf2-11ef-811c-40b0765b1e01";

const MOCK_TICKET_REASONS = [
    {
        id: "repair",
        title: "Ремонт",
        past: "Проведен ремонт",
        present: "Ремонт",
        future: "Провести ремонт",
    },
    {
        id: "diagnostic",
        title: "Диагностика",
        past: "Проведена диагностика",
        present: "Диагностика",
        future: "Провести диагностику",
    },
];

const MOCK_TICKETS = [
    {
        id: "ticket-12411",
        number: "12411",
        status: "inWork",
        executor: "01eaedd4-1bf2-11ef-811c-40b0765b1e01",
        reason: "repair",
        workstarted_at: "2026-03-05T08:10:00Z",
        workfinished_at: null,
        deviceName: "HP Color LaserJet Pro MFP M479fdw",
        deviceSerialNumber: "CNBXA59R4Q",
        clientName: "ООО ТехноСнаб",
        clientAddress: "Москва, ул. Большая Почтовая, 22",
    },
    {
        id: "ticket-12427",
        number: "12427",
        status: "worksDone",
        executor: "01eaedd4-1bf2-11ef-811c-40b0765b1e01",
        reason: "repair",
        workstarted_at: "2026-03-05T06:00:00Z",
        workfinished_at: "2026-03-05T07:20:00Z",
        deviceName: "Xerox AltaLink C8130",
        deviceSerialNumber: "XRX-C8130-9931",
        clientName: "АО СеверСтрой",
        clientAddress: "Санкт-Петербург, Лиговский проспект, 145",
    },
    {
        id: "ticket-12435",
        number: "12435",
        status: "inWork",
        executor: "01eaedd4-1bf2-11ef-811c-40b0765b1e01",
        reason: "diagnostic",
        workstarted_at: "2026-03-05T09:00:00Z",
        workfinished_at: null,
        deviceName: "Kyocera TASKalfa 4053ci",
        deviceSerialNumber: "KY4053CI-7F31",
        clientName: "ИП Вектор Логистика",
        clientAddress: "Казань, ул. Декабристов, 83",
    },
    {
        id: "ticket-12440",
        number: "12440",
        status: "worksDone",
        executor: "1c61434f-c3d6-431e-9a2d-4f2b4fa7f72a",
        reason: "diagnostic",
        workstarted_at: "2026-03-05T03:00:00Z",
        workfinished_at: "2026-03-05T04:15:00Z",
        deviceName: "Ricoh IM C3000",
        deviceSerialNumber: "RICOH-C3-12004",
        clientName: "ООО Прайм Консалт",
        clientAddress: "Екатеринбург, ул. Малышева, 51",
    },
    {
        id: "ticket-12453",
        number: "12453",
        status: "assigned",
        executor: "01eaedd4-1bf2-11ef-811c-40b0765b1e01",
        reason: "repair",
        assigned_end: "2026-03-12T18:00:00Z",
        urgent: false,
        deviceName: "Konica Minolta bizhub C300i",
        deviceSerialNumber: "KM-C300I-2201",
        clientName: "ООО ФинТраст",
        clientAddress: "Москва, ул. Бутырская, 18",
    },
    {
        id: "ticket-12461",
        number: "12461",
        status: "assigned",
        executor: "01eaedd4-1bf2-11ef-811c-40b0765b1e01",
        reason: "diagnostic",
        assigned_end: "2026-03-05T10:00:00Z",
        urgent: true,
        deviceName: "Canon imageRUNNER ADVANCE DX C3835i",
        deviceSerialNumber: "CANON-DX3835-31A",
        clientName: "АО Городской Центр Услуг",
        clientAddress: "Санкт-Петербург, ул. Савушкина, 41",
    },
    {
        id: "ticket-12472",
        number: "12472",
        status: "assigned",
        executor: "01eaedd4-1bf2-11ef-811c-40b0765b1e01",
        reason: "repair",
        assigned_end: "2026-03-17T12:00:00Z",
        urgent: true,
        deviceName: "Sharp BP-50C26",
        deviceSerialNumber: "SHARP-BP50-16M",
        clientName: "ООО Ресурс Поставка",
        clientAddress: "Нижний Новгород, ул. Белинского, 63",
    },
    {
        id: "ticket-12479",
        number: "12479",
        status: "assigned",
        executor: "01eaedd4-1bf2-11ef-811c-40b0765b1e01",
        reason: "diagnostic",
        assigned_end: "2026-03-02T09:00:00Z",
        urgent: false,
        deviceName: "Brother MFC-L6900DW",
        deviceSerialNumber: "BRO-L6900-771",
        clientName: "ООО Балтик Лайн",
        clientAddress: "Калининград, Ленинский проспект, 31",
    },
    {
        id: "ticket-12468",
        number: "12468",
        status: "assigned",
        executor: "f2b1d1ac-a088-4f13-8b43-72a4de0051fd",
        reason: "repair",
        assigned_end: "2026-03-07T15:00:00Z",
        urgent: true,
        deviceName: "Epson WorkForce Enterprise AM-C6000",
        deviceSerialNumber: "EPS-AMC6000-09P",
        clientName: "ООО Сигма Плюс",
        clientAddress: "Казань, ул. Петербургская, 36",
    },
];

function resolveTicketReason(ticket) {
    const reason = MOCK_TICKET_REASONS.find((item) => item.id === ticket.reason);
    if (!reason) {
        return "Не указано";
    }

    if (ticket.status === "assigned") {
        return reason.future || reason.title || "Не указано";
    }

    if (ticket.status === "worksDone") {
        return reason.past || reason.title || "Не указано";
    }

    return reason.present || reason.title || "Не указано";
}

function formatWorkDuration(startedAt, finishedAt) {
    if (!startedAt || !finishedAt) {
        return "Не указано";
    }

    const start = new Date(startedAt).getTime();
    const finish = new Date(finishedAt).getTime();
    if (Number.isNaN(start) || Number.isNaN(finish) || finish < start) {
        return "Не указано";
    }

    const minutes = Math.floor((finish - start) / 60000);
    const hoursPart = Math.floor(minutes / 60);
    const minutesPart = minutes % 60;

    if (hoursPart === 0) {
        return `${minutesPart} мин`;
    }

    return `${hoursPart} ч. ${minutesPart} мин`;
}

function formatDateDayMonth(value) {
    if (!value) {
        return "--.--";
    }

    const date = new Date(value);
    if (Number.isNaN(date.getTime())) {
        return "--.--";
    }

    const day = String(date.getDate()).padStart(2, "0");
    const month = String(date.getMonth() + 1).padStart(2, "0");
    return `${day}.${month}`;
}

function isTodayOrPast(value) {
    if (!value) {
        return false;
    }

    const target = new Date(value);
    if (Number.isNaN(target.getTime())) {
        return false;
    }

    const now = new Date();
    const todayStart = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime();
    const targetStart = new Date(target.getFullYear(), target.getMonth(), target.getDate()).getTime();

    return todayStart >= targetStart;
}

function DashboardHeader({ onOpenProfile }) {
    const today = new Intl.DateTimeFormat("ru-RU", {
        day: "numeric",
        month: "long",
    }).format(new Date());

    return (
        <header className="rounded-3xl border border-white/10 bg-slate-950/35 p-6 shadow-xl shadow-black/20 backdrop-blur">
            <div className="flex justify-between items-center gap-4">
                <div>
                    <p className="text-xs font-semibold uppercase tracking-[0.3em] text-slate-400">Дэшборд</p>
                    <h1 className="mt-3 text-3xl font-bold tracking-tight sm:text-4xl">Сегодня {today}</h1>
                </div>
                <button
                    type="button"
                    onClick={onOpenProfile}
                    aria-label="Открыть профиль"
                    className="inline-flex h-11 w-11 items-center justify-center rounded-2xl border border-white/15 bg-[#6A3BF2] text-white shadow-lg shadow-[#6A3BF2]/35 transition hover:bg-[#7C52F5]"
                >
                    <svg
                        xmlns="http://www.w3.org/2000/svg"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        strokeWidth="1.8"
                        className="h-5 w-5"
                    >
                        <circle cx="12" cy="8" r="3.6" />
                        <path d="M4.5 19.2C5.9 15.9 8.6 14.4 12 14.4s6.1 1.5 7.5 4.8" />
                    </svg>
                </button>
            </div>
        </header>
    );
}

function AssignedTicketCard({ ticket, onOpenTicket }) {
    const reasonValue = resolveTicketReason(ticket);
    const dueValue = formatDateDayMonth(ticket.assigned_end);
    const isOverdue = isTodayOrPast(ticket.assigned_end);
    const deadlineText = isOverdue ? `🔥 ${dueValue}` : `до ${dueValue}`;
    const shouldShowBadge = ticket.urgent;
    const shouldShowGradient = isOverdue || ticket.urgent;
    const badgeClassName = isOverdue
        ? "border-rose-200/40 bg-rose-500/25 text-rose-50"
        : "border-cyan-200/40 bg-cyan-500/25 text-cyan-50";
    const gradientClassName = isOverdue
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
                        <p className="font-semibold text-white">{deadlineText}</p>
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

function EngineerDashboard() {
    const navigate = useNavigate();
    const [activeSlide, setActiveSlide] = useState(0);
    const pointerStartXRef = useRef(null);
    const suppressClickRef = useRef(false);
    const tickets = useMemo(
        () =>
            MOCK_TICKETS.filter(
                (ticket) =>
                    ticket.executor === MOCK_EXECUTOR_ID &&
                    (ticket.status === "inWork" || ticket.status === "worksDone"),
            ),
        [],
    );
    const assignedTickets = useMemo(
        () => MOCK_TICKETS.filter((ticket) => ticket.executor === MOCK_EXECUTOR_ID && ticket.status === "assigned"),
        [],
    );

    function goToPreviousSlide() {
        setActiveSlide((prev) => (prev - 1 + tickets.length) % tickets.length);
    }

    function goToNextSlide() {
        setActiveSlide((prev) => (prev + 1) % tickets.length);
    }

    function handlePointerDown(event) {
        pointerStartXRef.current = event.clientX;
        suppressClickRef.current = false;
    }

    function handlePointerUp(event) {
        if (pointerStartXRef.current === null) {
            return;
        }

        const deltaX = event.clientX - pointerStartXRef.current;
        const swipeThreshold = 45;

        if (Math.abs(deltaX) >= swipeThreshold) {
            suppressClickRef.current = true;
            if (deltaX > 0) {
                goToPreviousSlide();
            } else {
                goToNextSlide();
            }
        }

        pointerStartXRef.current = null;
    }

    function handleOpenTicket(ticketId) {
        if (suppressClickRef.current) {
            suppressClickRef.current = false;
            return;
        }

        navigate(routePaths.ticketById(ticketId));
    }

    if (tickets.length === 0) {
        return (
            <section className="rounded-3xl border border-white/10 bg-white/5 p-6">
                <p className="text-sm text-slate-300">Для текущего инженера нет тикетов в процессе.</p>
            </section>
        );
    }

    return (
        <section className="space-y-6">
            <div
                className="overflow-hidden rounded-3xl"
                onPointerDown={handlePointerDown}
                onPointerUp={handlePointerUp}
                onPointerCancel={() => {
                    pointerStartXRef.current = null;
                }}
            >
                <div
                    className="flex transition-transform duration-300 ease-out"
                    style={{ transform: `translateX(-${activeSlide * 100}%)` }}
                >
                    {tickets.map((ticket) => {
                        const isInWork = ticket.status === "inWork";
                        const reasonValue = resolveTicketReason(ticket);
                        const isInWorkValue = isInWork
                            ? "В процессе"
                            : formatWorkDuration(ticket.workstarted_at, ticket.workfinished_at);
                        const cardClassName = isInWork
                            ? "border-emerald-300/30 bg-emerald-500/20"
                            : "border-fuchsia-300/30 bg-fuchsia-500/20";
                        const toneBlockClass = isInWork
                            ? "border-emerald-200/30 bg-emerald-200/20"
                            : "border-fuchsia-200/30 bg-fuchsia-200/20";

                        return (
                            <article key={ticket.id} className="min-w-full px-1">
                                <button
                                    type="button"
                                    onClick={() => handleOpenTicket(ticket.id)}
                                    className={`w-full rounded-3xl border p-6 text-left shadow-xl backdrop-blur transition hover:border-white/50 ${cardClassName}`}
                                >
                                    <div className="flex items-center gap-2">
                                        <p
                                            className={`rounded-xl border px-3 py-1.5 text-sm font-semibold text-white ${toneBlockClass}`}
                                        >
                                            {reasonValue || "Не указано"}
                                        </p>
                                        {isInWork ? (
                                            <p className="inline-flex items-center gap-2 text-sm font-semibold text-white">
                                                <span className="relative flex h-2.5 w-2.5">
                                                    <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-100 opacity-80" />
                                                    <span className="relative inline-flex h-2.5 w-2.5 rounded-full bg-emerald-50" />
                                                </span>
                                                {isInWorkValue}
                                            </p>
                                        ) : (
                                            <p className="text-sm font-semibold text-white">{isInWorkValue}</p>
                                        )}
                                        <p className="ml-auto text-sm font-semibold text-white">#{ticket.number}</p>
                                    </div>

                                    <div className="mt-4 flex flex-col gap-1.5 text-white">
                                        <p className="text-base font-semibold">{ticket.deviceName}</p>
                                        <p className="text-sm text-slate-100/90">С/Н {ticket.deviceSerialNumber}</p>
                                    </div>

                                    <div className={`mt-4 rounded-2xl border p-4 ${toneBlockClass}`}>
                                        <div className="flex flex-col gap-1 text-white">
                                            <p className="text-sm font-semibold">{ticket.clientName}</p>
                                            <p className="text-sm text-slate-100/90">{ticket.clientAddress}</p>
                                        </div>
                                    </div>
                                </button>
                            </article>
                        );
                    })}
                </div>
            </div>

            <div className="flex items-center justify-center gap-2">
                {tickets.map((ticket, index) => {
                    const isActive = index === activeSlide;

                    return (
                        <button
                            key={ticket.id}
                            type="button"
                            onClick={() => setActiveSlide(index)}
                            aria-label={`Перейти к слайду ${index + 1}`}
                            className={`h-2.5 rounded-full transition ${
                                isActive ? "w-8 bg-white" : "w-2.5 bg-white/35 hover:bg-white/60"
                            }`}
                        />
                    );
                })}
            </div>

            <section className="space-y-3">
                <h2 className="text-sm font-semibold uppercase tracking-[0.18em] text-slate-300">Назначенные тикеты</h2>
                {assignedTickets.length > 0 ? (
                    <div className="grid gap-3">
                        {assignedTickets.map((ticket) => (
                            <AssignedTicketCard key={ticket.id} ticket={ticket} onOpenTicket={handleOpenTicket} />
                        ))}
                    </div>
                ) : (
                    <div className="rounded-2xl border border-white/10 bg-white/5 p-4 text-sm text-slate-300">
                        Нет назначенных тикетов.
                    </div>
                )}
            </section>
        </section>
    );
}

function CoordinatorDashboard() {
    return (
        <section className="grid gap-4 sm:grid-cols-2">
            <article className="rounded-3xl border border-white/10 bg-white/5 p-5">
                <p className="text-xs font-semibold uppercase tracking-[0.25em] text-slate-400">Команда</p>
                <p className="mt-3 text-2xl font-semibold text-slate-100">24 инженера</p>
            </article>
            <article className="rounded-3xl border border-white/10 bg-white/5 p-5">
                <p className="text-xs font-semibold uppercase tracking-[0.25em] text-slate-400">Требуют внимания</p>
                <p className="mt-3 text-2xl font-semibold text-slate-100">5 задач</p>
            </article>
        </section>
    );
}

export function DashboardPage() {
    const navigate = useNavigate();
    const { session } = useAuth();
    const role = session?.role || "user";
    const isCoordinatorOrAdmin = role === "coordinator" || role === "admin";

    return (
        <PageShell>
            <section className="w-full space-y-6">
                <DashboardHeader onOpenProfile={() => navigate(routePaths.profile)} />
                {isCoordinatorOrAdmin ? <CoordinatorDashboard /> : <EngineerDashboard />}
            </section>
        </PageShell>
    );
}
