import { useMemo } from "react";
import { Link, useNavigate } from "react-router";
import { routePaths } from "../../../shared/config/routes";
import { useGetDepartmentMembersQuery, useGetDepartmentTicketsQuery } from "../../../shared/api/tickets-api";
import { ProfileTicketCard } from "../../../shared/ui/profile-ticket-card";
import { TicketCardWithExecutor } from "./ticket-card-with-executor";

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

function MemberCard({ latestClientAddress, member, to, totalTickets }) {
    const status = member.latestTicketStatus;
    const isDisabled = member.isDisabled || status === "disabled";
    const isInWork = status === "inWork";
    const isDone = status === "worksDone";
    const toneClass = isDisabled
        ? "border-rose-300/25 bg-rose-500/10"
        : isInWork
        ? "border-[#9f85ff]/45 bg-[#6A3BF2]/24"
        : isDone
          ? "border-emerald-300/35 bg-emerald-500/18"
          : "border-cyan-300/25 bg-cyan-500/10";
    const pulseClass = isInWork ? "bg-[#6A3BF2]/55" : "bg-emerald-400/35";
    const cardClass = isDisabled
        ? "border-rose-300/20 bg-rose-950/25"
        : isInWork
        ? "border-[#8d73ff]/45 bg-[#4b24c7]/32 shadow-[#6A3BF2]/30"
        : isDone
          ? "border-emerald-300/35 bg-emerald-950/30 shadow-emerald-500/20"
          : "border-white/10 bg-slate-950/35";
    const statusTextClass = isDisabled
        ? "text-rose-200"
        : isInWork
        ? "text-[#ede7ff]"
        : isDone
          ? "text-emerald-100"
          : "text-cyan-200";

    return (
        <Link
            to={to}
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

            <div className="mt-2 min-h-[3.2rem] text-sm font-semibold">
                {isDisabled ? (
                    <p className={statusTextClass}>Временно недоступен</p>
                ) : isInWork ? (
                    <p className={`${statusTextClass} text-[0.95rem] leading-5`}>
                        {latestClientAddress?.trim() || "Адрес клиента не указан"}
                    </p>
                ) : isDone ? (
                    <p className={`${statusTextClass} text-[0.95rem] leading-5`}>
                        {latestClientAddress?.trim() || "Адрес клиента не указан"}
                    </p>
                ) : (
                    <p className={statusTextClass}>{`${totalTickets} тикетов`}</p>
                )}
            </div>
        </Link>
    );
}

export function CoordinatorDashboard({ department }) {
    const navigate = useNavigate();
    const normalizedDepartment = (department || "").trim();
    const { data: departmentTickets = [], isError, isFetching, isLoading } = useGetDepartmentTicketsQuery(
        normalizedDepartment || "__no_department__",
        { skip: !normalizedDepartment },
    );
    const {
        data: departmentMembers = [],
        isError: isDepartmentMembersError,
        isFetching: isDepartmentMembersFetching,
        isLoading: isDepartmentMembersLoading,
    } = useGetDepartmentMembersQuery(undefined, {
        skip: !normalizedDepartment,
    });

    const unassignedTickets = useMemo(() => {
        return sortByAssignedEndDesc(departmentTickets.filter((ticket) => ticket.status === "created"));
    }, [departmentTickets]);

    const activeDepartmentTickets = useMemo(() => {
        const excludedStatuses = new Set(["canceled", "cancelled", "closed"]);
        return sortByAssignedEndDesc(
            departmentTickets.filter((ticket) => !excludedStatuses.has(ticket.status) && ticket.status !== "created"),
        );
    }, [departmentTickets]);

    const departmentMemberById = useMemo(() => {
        return departmentMembers.reduce((accumulator, member) => {
            accumulator[member.id] = member;
            return accumulator;
        }, {});
    }, [departmentMembers]);

    const latestDepartmentTicketById = useMemo(() => {
        return departmentTickets.reduce((accumulator, ticket) => {
            accumulator[ticket.id] = ticket;
            return accumulator;
        }, {});
    }, [departmentTickets]);

    const inWorkTicketByExecutor = useMemo(() => {
        return departmentTickets.reduce((accumulator, ticket) => {
            if (ticket.status !== "inWork" || !ticket.executor) {
                return accumulator;
            }

            const currentTicket = accumulator[ticket.executor];
            if (!currentTicket || toTimestampOrMin(ticket.workstarted_at) > toTimestampOrMin(currentTicket.workstarted_at)) {
                accumulator[ticket.executor] = ticket;
            }

            return accumulator;
        }, {});
    }, [departmentTickets]);

    const worksDoneTicketByExecutor = useMemo(() => {
        return departmentTickets.reduce((accumulator, ticket) => {
            if (ticket.status !== "worksDone" || !ticket.executor) {
                return accumulator;
            }

            const currentTicket = accumulator[ticket.executor];
            if (
                !currentTicket ||
                toTimestampOrMin(ticket.workfinished_at) > toTimestampOrMin(currentTicket.workfinished_at)
            ) {
                accumulator[ticket.executor] = ticket;
            }

            return accumulator;
        }, {});
    }, [departmentTickets]);

    const latestClientAddressByMemberId = useMemo(() => {
        return departmentMembers.reduce((accumulator, member) => {
            const latestTicket = member.latestTicket ? latestDepartmentTicketById[member.latestTicket] : null;
            const latestAddressTicket =
                latestTicket?.status === "inWork" || latestTicket?.status === "worksDone"
                    ? latestTicket
                    : member.latestTicketStatus === "inWork"
                      ? inWorkTicketByExecutor[member.id]
                      : member.latestTicketStatus === "worksDone"
                        ? worksDoneTicketByExecutor[member.id]
                        : null;

            accumulator[member.id] = latestAddressTicket?.clientAddress || "";
            return accumulator;
        }, {});
    }, [departmentMembers, inWorkTicketByExecutor, latestDepartmentTicketById, worksDoneTicketByExecutor]);

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
                            <ProfileTicketCard key={ticket.id} ticket={ticket} onOpenTicket={handleOpenTicket} />
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
                {isDepartmentMembersLoading || isDepartmentMembersFetching ? (
                    <div className="rounded-2xl border border-white/10 bg-white/5 p-4 text-sm text-slate-300">
                        Загружаем сотрудников...
                    </div>
                ) : isDepartmentMembersError ? (
                    <div className="rounded-2xl border border-rose-300/30 bg-rose-500/10 p-4 text-sm text-rose-100">
                        Не удалось загрузить сотрудников отдела.
                    </div>
                ) : departmentMembers.length > 0 ? (
                    <div className="flex gap-4 overflow-x-auto px-1 pt-1 pb-5 pr-3">
                        {departmentMembers.map((member) => (
                            <MemberCard
                                key={member.id}
                                latestClientAddress={latestClientAddressByMemberId[member.id]}
                                member={member}
                                to={routePaths.profileById(member.id)}
                                totalTickets={ticketsByExecutor[member.id] || 0}
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
