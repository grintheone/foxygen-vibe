import { useMemo } from "react";
import { useNavigate } from "react-router";
import { routePaths } from "../../../shared/config/routes";
import { useGetDepartmentTicketsQuery } from "../../../shared/api/tickets-api";
import { MOCK_DEPARTMENT_MEMBERS } from "../model/mock-dashboard-data";
import { TicketCardWithExecutor } from "./ticket-card-with-executor";
import { TicketCardWithStatus } from "./ticket-card-with-status";

function toTimestampOrMin(value) {
    if (!value) {
        return Number.NEGATIVE_INFINITY;
    }

    const timestamp = new Date(value).getTime();
    if (Number.isNaN(timestamp)) {
        return Number.NEGATIVE_INFINITY;
    }

    return timestamp;
}

function sortByAssignedEndDesc(tickets) {
    return [...tickets].sort((a, b) => {
        return toTimestampOrMin(b.assigned_end) - toTimestampOrMin(a.assigned_end);
    });
}

function PersonIcon({ className }) {
    return (
        <svg
            xmlns="http://www.w3.org/2000/svg"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="1.8"
            className={className}
            aria-hidden="true"
        >
            <circle cx="12" cy="8" r="3.6" />
            <path d="M4.5 19.2C5.9 15.9 8.6 14.4 12 14.4s6.1 1.5 7.5 4.8" />
        </svg>
    );
}

function DoneBadgeIcon() {
    return (
        <span className="absolute -bottom-1 -right-1 inline-flex h-8 w-8 items-center justify-center rounded-full bg-emerald-400 text-emerald-950 shadow-md shadow-emerald-900/30">
            <svg
                xmlns="http://www.w3.org/2000/svg"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2.4"
                className="h-4.5 w-4.5"
                aria-hidden="true"
            >
                <path d="M5 12.5l4.2 4.2L19 7.8" />
            </svg>
        </span>
    );
}

function DisabledBadgeIcon() {
    return (
        <span className="absolute -bottom-1 -right-1 inline-flex h-8 w-8 items-center justify-center rounded-full bg-rose-100 text-rose-700 shadow-md shadow-rose-950/30">
            <svg
                xmlns="http://www.w3.org/2000/svg"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2.2"
                className="h-4.5 w-4.5"
                aria-hidden="true"
            >
                <circle cx="12" cy="12" r="8" />
                <path d="M6.3 6.3l11.4 11.4" />
            </svg>
        </span>
    );
}

function MemberCard({ member, totalTickets }) {
    const status = member.latestTicketStatus;
    const isDisabled = member.isDisabled || status === "disabled";
    const isInWork = status === "inWork";
    const isDone = status === "worksDone";
    const toneClass = isDisabled
        ? "border-rose-300/25 bg-rose-500/10"
        : isInWork
        ? "border-emerald-300/25 bg-emerald-500/10"
        : isDone
          ? "border-fuchsia-300/25 bg-fuchsia-500/10"
          : "border-cyan-300/25 bg-cyan-500/10";
    const pulseClass = isInWork ? "bg-[#6A3BF2]/45" : "bg-emerald-400/35";
    const cardClass = isDisabled
        ? "border-rose-300/20 bg-rose-950/25"
        : isInWork
        ? "border-emerald-300/20 bg-slate-950/35"
        : isDone
          ? "border-fuchsia-300/20 bg-slate-950/35"
          : "border-white/10 bg-slate-950/35";
    const statusTextClass = isDisabled
        ? "text-rose-200"
        : isInWork
        ? "text-emerald-200"
        : isDone
          ? "text-fuchsia-200"
          : "text-cyan-200";

    return (
        <article
            className={`h-[19rem] w-[14rem] shrink-0 overflow-hidden rounded-[1.65rem] border p-3 text-slate-100 shadow-xl shadow-black/20 backdrop-blur ${cardClass}`}
        >
            <div className={`flex justify-center rounded-[1.3rem] border p-3 ${toneClass}`}>
                <div className="relative h-36 w-36">
                    {isInWork || isDone ? (
                        <span
                            aria-hidden="true"
                            className={`absolute -inset-2 animate-ping rounded-full ${pulseClass}`}
                        />
                    ) : null}
                    <span className="absolute inset-0 rounded-full bg-white/5" aria-hidden="true" />

                    {member.avatarUrl ? (
                        <img
                            src={member.avatarUrl}
                            alt={member.name}
                            className={`relative h-full w-full rounded-full border border-white/10 object-cover ${isDisabled ? "grayscale opacity-80" : ""}`}
                        />
                    ) : (
                        <div
                            className={`relative flex h-full w-full items-center justify-center rounded-full border border-white/10 bg-white/5 text-slate-400 ${isDisabled ? "text-rose-200/70" : ""}`}
                        >
                            <PersonIcon className="h-10 w-10" />
                        </div>
                    )}

                    {isDisabled ? <DisabledBadgeIcon /> : null}
                    {!isDisabled && isDone ? <DoneBadgeIcon /> : null}
                </div>
            </div>

            <p className="mt-3 text-[1.75rem] leading-7 font-semibold tracking-tight text-white">{member.name}</p>

            <div className="mt-2 min-h-[1.8rem] text-sm font-semibold">
                {isDisabled ? (
                    <p className={statusTextClass}>Временно недоступен</p>
                ) : isInWork ? (
                    <p className={statusTextClass}>В работе</p>
                ) : isDone ? (
                    <p className={statusTextClass}>Работы завершены</p>
                ) : (
                    <p className={statusTextClass}>{`${totalTickets} тикетов`}</p>
                )}
            </div>
        </article>
    );
}

export function CoordinatorDashboard({ department }) {
    const navigate = useNavigate();
    const normalizedDepartment = (department || "").trim();
    const { data: departmentTickets = [], isError, isFetching, isLoading } = useGetDepartmentTicketsQuery(
        normalizedDepartment || "__no_department__",
        { skip: !normalizedDepartment },
    );

    const unassignedTickets = useMemo(() => {
        return sortByAssignedEndDesc(departmentTickets.filter((ticket) => ticket.status === "created"));
    }, [departmentTickets]);

    const departmentMembers = useMemo(() => {
        if (!normalizedDepartment) {
            return MOCK_DEPARTMENT_MEMBERS;
        }

        return MOCK_DEPARTMENT_MEMBERS.filter((member) => (member.department || "").trim() === normalizedDepartment);
    }, [normalizedDepartment]);

    const activeDepartmentTickets = useMemo(() => {
        const excludedStatuses = new Set(["canceled", "cancelled", "closed"]);
        return sortByAssignedEndDesc(
            departmentTickets.filter((ticket) => !excludedStatuses.has(ticket.status) && ticket.status !== "created"),
        );
    }, [departmentTickets]);

    const departmentMemberById = useMemo(() => {
        return departmentMembers.reduce((accumulator, member) => {
            accumulator[member.userId] = member;
            return accumulator;
        }, {});
    }, [departmentMembers]);

    const ticketsByExecutor = useMemo(() => {
        return departmentTickets.reduce((accumulator, ticket) => {
            if (!ticket.executor) {
                return accumulator;
            }

            const currentCount = accumulator[ticket.executor] || 0;
            accumulator[ticket.executor] = currentCount + 1;
            return accumulator;
        }, {});
    }, [departmentTickets]);

    function handleOpenTicket(ticketId) {
        navigate(routePaths.ticketById(ticketId));
    }

    return (
        <section className="space-y-6">
            <section className="space-y-3">
                <h2 className="text-sm font-semibold tracking-[0.02em] text-slate-200">{`Ждут распределения - ${unassignedTickets.length}`}</h2>
                {isLoading || isFetching ? (
                    <div className="rounded-2xl border border-white/10 bg-white/5 p-4 text-sm text-slate-300">
                        Загружаем тикеты...
                    </div>
                ) : isError ? (
                    <div className="rounded-2xl border border-rose-300/30 bg-rose-500/10 p-4 text-sm text-rose-100">
                        Не удалось загрузить тикеты.
                    </div>
                ) : unassignedTickets.length > 0 ? (
                    <div className="grid gap-3">
                        {unassignedTickets.map((ticket) => (
                            <TicketCardWithStatus key={ticket.id} ticket={ticket} onOpenTicket={handleOpenTicket} />
                        ))}
                    </div>
                ) : (
                    <div className="rounded-2xl border border-white/10 bg-white/5 p-4 text-sm text-slate-300">
                        Нет тикетов в статусе created для вашего отдела.
                    </div>
                )}
            </section>

            <section className="space-y-3">
                <h2 className="text-3xl font-bold tracking-tight text-white sm:text-4xl">
                    {`Отдел ${departmentMembers.length}`}
                </h2>
                {departmentMembers.length > 0 ? (
                    <div className="flex gap-4 overflow-x-auto pb-2 pr-2">
                        {departmentMembers.map((member) => (
                            <MemberCard
                                key={member.userId}
                                member={member}
                                totalTickets={ticketsByExecutor[member.userId] || 0}
                            />
                        ))}
                    </div>
                ) : (
                    <div className="rounded-2xl border border-white/10 bg-white/5 p-4 text-sm text-slate-300">
                        В этом отделе пока нет сотрудников.
                    </div>
                )}
            </section>

            <section className="space-y-3">
                <h2 className="text-sm font-semibold tracking-[0.02em] text-slate-200">{`Назначены и в работе - ${activeDepartmentTickets.length}`}</h2>
                {isLoading || isFetching ? (
                    <div className="rounded-2xl border border-white/10 bg-white/5 p-4 text-sm text-slate-300">
                        Загружаем тикеты...
                    </div>
                ) : isError ? (
                    <div className="rounded-2xl border border-rose-300/30 bg-rose-500/10 p-4 text-sm text-rose-100">
                        Не удалось загрузить тикеты.
                    </div>
                ) : activeDepartmentTickets.length > 0 ? (
                    <div className="grid gap-3">
                        {activeDepartmentTickets.map((ticket) => {
                            const fallbackExecutor =
                                ticket.executorName || ticket.executorDepartment
                                    ? {
                                          name: ticket.executorName || "Исполнитель не назначен",
                                          department: ticket.executorDepartment || "Отдел не указан",
                                          avatarUrl: "",
                                      }
                                    : null;

                            return (
                            <TicketCardWithExecutor
                                key={ticket.id}
                                ticket={ticket}
                                executor={departmentMemberById[ticket.executor] || fallbackExecutor}
                                onOpenTicket={handleOpenTicket}
                            />
                            );
                        })}
                    </div>
                ) : (
                    <div className="rounded-2xl border border-white/10 bg-white/5 p-4 text-sm text-slate-300">
                        Нет активных тикетов для сотрудников вашего отдела.
                    </div>
                )}
            </section>
        </section>
    );
}
