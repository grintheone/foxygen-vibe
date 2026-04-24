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
        <span className="absolute -bottom-1 -right-1 inline-flex h-6 w-6 items-center justify-center rounded-full bg-emerald-400 text-emerald-950 shadow-md shadow-emerald-900/30">
            <svg
                xmlns="http://www.w3.org/2000/svg"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2.4"
                className="h-3.5 w-3.5"
                aria-hidden="true"
            >
                <path d="M5 12.5l4.2 4.2L19 7.8" />
            </svg>
        </span>
    );
}

function DisabledBadgeIcon() {
    return (
        <span className="absolute -bottom-1 -right-1 inline-flex h-6 w-6 items-center justify-center rounded-full bg-rose-100 text-rose-700 shadow-md shadow-rose-950/30">
            <svg
                xmlns="http://www.w3.org/2000/svg"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2.2"
                className="h-3.5 w-3.5"
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
    const pulseClass = isInWork ? "bg-[#6A3BF2]/55" : "bg-emerald-400/35";
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
            className={`w-32 shrink-0 overflow-hidden rounded-lg border border-slate-400/20 bg-[#2f3748] p-2.5 text-slate-100 shadow-xl shadow-black/20 transition hover:border-slate-300/35 hover:bg-[#333c4f] ${isDisabled ? "opacity-80" : ""}`}
        >
            <div className="flex justify-center">
                <div className="relative h-24 w-24">
                    {isInWork || isDone ? (
                        <span
                            aria-hidden="true"
                            className={`absolute -inset-1.5 animate-ping rounded-full ${pulseClass}`}
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
                            <PersonIcon className="h-7 w-7" />
                        </div>
                    )}

                    {isDisabled ? <DisabledBadgeIcon /> : null}
                    {!isDisabled && isDone ? <DoneBadgeIcon /> : null}
                </div>
            </div>

            <p className="mt-3 text-base font-semibold leading-5 text-white">{member.name}</p>

            <div className="mt-1.5 text-xs font-semibold leading-4">
                {isDisabled ? (
                    <p className={statusTextClass}>Временно недоступен</p>
                ) : isInWork ? (
                    <p className={`${statusTextClass} line-clamp-2`}>
                        {latestClientAddress?.trim() || "Адрес клиента не указан"}
                    </p>
                ) : isDone ? (
                    <p className={`${statusTextClass} line-clamp-2`}>
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
                {isDepartmentMembersLoading || isDepartmentMembersFetching ? (
                    <div className="app-subtle-notice">
                        Загружаем сотрудников...
                    </div>
                ) : isDepartmentMembersError ? (
                    <div className="rounded-2xl border border-rose-300/30 bg-rose-500/10 p-4 text-sm text-rose-100">
                        Не удалось загрузить сотрудников отдела.
                    </div>
                ) : departmentMembers.length > 0 ? (
                    <div className="-mx-1 flex gap-2 overflow-x-auto px-1 pt-1 scroll-px-1">
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
                    <div className="app-subtle-notice">
                        В этом отделе пока нет сотрудников.
                    </div>
                )}
            </section>

            <section className="space-y-3">
                <h2 className="text-base font-semibold tracking-[0.02em] text-slate-200">{`Ждут распределения (${unassignedTickets.length})`}</h2>
                {isLoading || isFetching ? (
                    <div className="app-subtle-notice">
                        Загружаем тикеты...
                    </div>
                ) : isError ? (
                    <div className="rounded-2xl border border-rose-300/30 bg-rose-500/10 p-4 text-sm text-rose-100">
                        Не удалось загрузить тикеты.
                    </div>
                ) : unassignedTickets.length > 0 ? (
                    <div className="grid gap-2">
                        {unassignedTickets.map((ticket) => (
                            <ProfileTicketCard key={ticket.id} ticket={ticket} onOpenTicket={handleOpenTicket} />
                        ))}
                    </div>
                ) : (
                    <div className="app-subtle-notice">
                        Нет тикетов к распределению
                    </div>
                )}
            </section>

            <section className="space-y-3">
                <h2 className="text-base font-semibold tracking-[0.02em] text-slate-200">{`Назначены и в работе (${activeDepartmentTickets.length})`}</h2>
                {isLoading || isFetching ? (
                    <div className="app-subtle-notice">
                        Загружаем тикеты...
                    </div>
                ) : isError ? (
                    <div className="rounded-2xl border border-rose-300/30 bg-rose-500/10 p-4 text-sm text-rose-100">
                        Не удалось загрузить тикеты.
                    </div>
                ) : activeDepartmentTickets.length > 0 ? (
                    <div className="grid gap-2">
                        {activeDepartmentTickets.map((ticket) => {
                            const fallbackExecutor =
                                ticket.executorName || ticket.executorDepartment
                                    ? {
                                          id: ticket.executor,
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
                    <div className="app-subtle-notice">
                        Нет активных тикетов для сотрудников вашего отдела.
                    </div>
                )}
            </section>
        </section>
    );
}
