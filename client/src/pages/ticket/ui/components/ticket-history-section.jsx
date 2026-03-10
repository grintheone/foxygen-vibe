import ticketAssignedIcon from "../../../../assets/icons/ticket-assigned.svg";
import ticketClosedIcon from "../../../../assets/icons/ticket-closed.svg";
import ticketDoneIcon from "../../../../assets/icons/ticket-done.svg";
import ticketInWorkIcon from "../../../../assets/icons/ticket-inwork.svg";

function formatHistoryDate(value) {
    if (!value) {
        return null;
    }

    const date = new Date(value);
    if (Number.isNaN(date.getTime())) {
        return null;
    }

    const day = String(date.getDate()).padStart(2, "0");
    const month = String(date.getMonth() + 1).padStart(2, "0");
    const year = String(date.getFullYear()).slice(-2);

    return `${day}.${month}.${year}`;
}

function formatHistoryTime(value) {
    if (!value) {
        return null;
    }

    const date = new Date(value);
    if (Number.isNaN(date.getTime())) {
        return null;
    }

    const hours = String(date.getHours()).padStart(2, "0");
    const minutes = String(date.getMinutes()).padStart(2, "0");

    return `${hours}:${minutes}`;
}

function buildHistoryEntries(ticket) {
    const rawEntries = [
        {
            id: "assigned",
            title: "Назначил",
            actorName: ticket?.assignedByName,
            happenedAt: ticket?.assigned_at,
            icon: ticketAssignedIcon,
        },
        {
            id: "inWork",
            title: "Начал работы",
            actorName: ticket?.executorName,
            happenedAt: ticket?.workstarted_at,
            icon: ticketInWorkIcon,
        },
        {
            id: "worksDone",
            title: "Завершил работы",
            actorName: ticket?.executorName,
            happenedAt: ticket?.workfinished_at,
            icon: ticketDoneIcon,
        },
        {
            id: "closed",
            title: "Закрыл тикет",
            actorName: ticket?.executorName,
            happenedAt: ticket?.closed_at,
            icon: ticketClosedIcon,
        },
    ];

    return rawEntries
        .filter((entry) => entry.actorName?.trim() && entry.happenedAt)
        .map((entry) => ({
            ...entry,
            date: formatHistoryDate(entry.happenedAt),
            time: formatHistoryTime(entry.happenedAt),
        }))
        .filter((entry) => entry.date && entry.time)
        .reverse();
}

export function TicketHistorySection({ ticket }) {
    const historyEntries = buildHistoryEntries(ticket);

    if (historyEntries.length === 0) {
        return null;
    }

    return (
        <section className="space-y-5 rounded-3xl border border-white/15 bg-white/10 p-5 sm:p-6">
            <h2 className="text-4xl font-semibold tracking-tight text-slate-300">История тикета</h2>
            <div className="space-y-6">
                {historyEntries.map((entry, index) => {
                    const isLast = index === historyEntries.length - 1;

                    return (
                        <article key={entry.id} className="grid grid-cols-[1fr_auto] gap-4">
                            <div className="flex min-w-0 items-start gap-4">
                                <span className="mt-1 inline-flex h-12 w-12 shrink-0 rounded-full bg-slate-950" />
                                <div className="min-w-0">
                                    <p className="text-2xl text-slate-400">{entry.title}</p>
                                    <p className="text-3xl font-semibold leading-tight text-slate-100">{entry.actorName}</p>
                                </div>
                            </div>

                            <div className="flex items-start gap-3">
                                <div className="text-right">
                                    <p className="text-2xl font-semibold text-slate-50">{entry.date}</p>
                                    <p className="text-2xl text-slate-400">{entry.time}</p>
                                </div>
                                <div className="relative flex min-h-[4.25rem] w-6 justify-center">
                                    {!isLast ? (
                                        <span className="absolute left-1/2 top-10 h-[calc(100%-1.25rem)] w-0.5 -translate-x-1/2 bg-slate-400/60" />
                                    ) : null}
                                    <img src={entry.icon} alt="" className="relative z-[1] mt-1 h-6 w-6" />
                                </div>
                            </div>
                        </article>
                    );
                })}
            </div>
        </section>
    );
}
