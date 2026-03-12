import { useState } from "react";
import { useNavigate, useParams } from "react-router";
import {
    useGetClientByIdQuery,
    useGetClientTicketsQuery,
    useGetDeviceByIdQuery,
    useGetDeviceTicketsQuery,
} from "../../../shared/api/tickets-api";
import { routePaths } from "../../../shared/config/routes";
import { PageShell } from "../../../shared/ui/page-shell";
import { SlideOverSheet } from "../../../shared/ui/slide-over-sheet";
import { TicketCardWithExecutor } from "../../dashboard/ui/ticket-card-with-executor";

const archiveConfigByEntity = {
    client: {
        emptyAllMessage: "У этого клиента пока нет выездов.",
        emptyClosedMessage: "У этого клиента пока нет закрытых выездов.",
        emptyInWorkMessage: "У этого клиента сейчас нет выездов в работе.",
        entityFallbackTitle: "Клиент",
        loadingEntityMessage: "Загрузка клиента...",
        loadingTicketsMessage: "Загрузка архива выездов...",
        ticketsHeading: "Все выезды клиента",
        ticketsErrorMessage: "Не удалось загрузить архив выездов клиента.",
    },
    device: {
        emptyAllMessage: "У этого устройства пока нет выездов.",
        emptyClosedMessage: "У этого устройства пока нет закрытых выездов.",
        emptyInWorkMessage: "У этого устройства сейчас нет выездов в работе.",
        entityFallbackTitle: "Устройство",
        loadingEntityMessage: "Загрузка устройства...",
        loadingTicketsMessage: "Загрузка архива выездов...",
        ticketsHeading: "Все выезды устройства",
        ticketsErrorMessage: "Не удалось загрузить архив выездов устройства.",
    },
};

const archiveTabs = [
    { id: "closed", label: "Завершенные" },
    { id: "inWork", label: "В работе" },
    { id: "all", label: "Все" },
];

const archiveGroupingOptions = [
    { id: "months", description: "Собрать тикеты по месяцам.", label: "По месяцам" },
    { id: "reason", description: "Группировать по причине выезда.", label: "По причине" },
    {
        id: "departments",
        description: "Разделить архив по отделам.",
        label: "По отделам",
    },
];

const archiveSortOptions = [
    { id: "newest", description: "Сначала последние изменения и закрытия.", label: "Сначала новые" },
    { id: "oldest", description: "Начать с самых ранних выездов.", label: "Сначала старые" },
];

function BackButton({ onClick }) {
    return (
        <button
            type="button"
            onClick={onClick}
            aria-label="Назад"
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
    );
}

function ArchiveHeader({ entityMeta, entityTitle, onBack }) {
    return (
        <header className="rounded-3xl border border-white/10 bg-slate-950/35 p-6 shadow-xl shadow-black/20 backdrop-blur">
            <BackButton onClick={onBack} />

            <div className="mt-5">
                <p className="text-xs font-semibold uppercase tracking-[0.3em] text-slate-400">Архив</p>
                <p className="mt-4 text-2xl font-semibold tracking-tight text-slate-100 sm:text-3xl">{entityTitle}</p>
                {entityMeta ? <p className="mt-3 text-base text-slate-300 sm:text-lg">{entityMeta}</p> : null}
            </div>
        </header>
    );
}

function ArchiveTabButton({ isActive, label, onClick }) {
    return (
        <button
            type="button"
            onClick={onClick}
            className={`rounded-2xl border px-4 py-2 text-sm font-semibold transition sm:text-base ${
                isActive
                    ? "border-[#9B7BFF]/70 bg-[#9B7BFF]/20 text-[#E7DEFF]"
                    : "border-white/10 bg-white/5 text-slate-300 hover:border-white/20 hover:bg-white/10 hover:text-slate-100"
            }`}
        >
            {label}
        </button>
    );
}

function ArchiveFilterButton({ onClick }) {
    return (
        <button
            type="button"
            onClick={onClick}
            aria-label="Открыть фильтры"
            className="inline-flex h-11 w-11 items-center justify-center rounded-2xl border border-white/10 bg-white/5 text-slate-200 transition hover:border-white/20 hover:bg-white/10 hover:text-white"
        >
            <svg
                xmlns="http://www.w3.org/2000/svg"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="1.9"
                strokeLinecap="round"
                strokeLinejoin="round"
                className="h-5 w-5"
                aria-hidden="true"
            >
                <path d="M4 5h16l-6 7v5l-4 2v-7L4 5Z" />
            </svg>
        </button>
    );
}

function filterTicketsByTab(tickets, activeTab) {
    if (activeTab === "closed") {
        return tickets.filter((ticket) => ticket.status === "closed");
    }

    if (activeTab === "inWork") {
        return tickets.filter((ticket) => ticket.status === "inWork");
    }

    return tickets;
}

function resolveEmptyMessage(config, activeTab) {
    if (activeTab === "closed") {
        return config.emptyClosedMessage;
    }

    if (activeTab === "inWork") {
        return config.emptyInWorkMessage;
    }

    return config.emptyAllMessage;
}

function capitalizeFirstLetter(value) {
    if (!value) {
        return value;
    }

    return value.charAt(0).toUpperCase() + value.slice(1);
}

function resolveTicketTimelineDate(ticket) {
    return ticket?.closed_at || ticket?.workfinished_at || ticket?.assigned_end || ticket?.workstarted_at || null;
}

function resolveTicketGroupLabel(ticket, activeGrouping) {
    if (activeGrouping === "reason") {
        return ticket?.reason?.trim() || "Причина не указана";
    }

    if (activeGrouping === "departments") {
        return ticket?.executorDepartment?.trim() || "Отдел не указан";
    }

    const timelineDate = resolveTicketTimelineDate(ticket);
    if (!timelineDate) {
        return "Без даты";
    }

    const date = new Date(timelineDate);
    if (Number.isNaN(date.getTime())) {
        return "Без даты";
    }

    return capitalizeFirstLetter(
        new Intl.DateTimeFormat("ru-RU", {
            month: "long",
            year: "numeric",
        }).format(date),
    );
}

function groupTickets(tickets, activeGrouping) {
    const groups = new Map();

    tickets.forEach((ticket) => {
        const label = resolveTicketGroupLabel(ticket, activeGrouping);

        if (!groups.has(label)) {
            groups.set(label, []);
        }

        groups.get(label).push(ticket);
    });

    return Array.from(groups.entries()).map(([label, items]) => ({
        items,
        label,
    }));
}

function resolveGroupingLabel(activeGrouping) {
    return archiveGroupingOptions.find((option) => option.id === activeGrouping)?.label || "По месяцам";
}

function FilterSection({ children, description, title }) {
    return (
        <section className="space-y-3 rounded-3xl border border-white/10 bg-white/5 p-5">
            <div>
                <h3 className="text-lg font-semibold text-slate-100">{title}</h3>
                {description ? <p className="mt-2 text-sm leading-6 text-slate-300">{description}</p> : null}
            </div>
            {children}
        </section>
    );
}

function FilterToggleChip({ isActive, label, onClick }) {
    return (
        <button
            type="button"
            onClick={onClick}
            className={`rounded-2xl border px-4 py-3 text-sm font-semibold transition ${
                isActive
                    ? "border-[#9B7BFF]/70 bg-[#9B7BFF]/20 text-[#E7DEFF]"
                    : "border-white/10 bg-slate-950/35 text-slate-300 hover:border-white/20 hover:bg-white/10 hover:text-slate-100"
            }`}
        >
            {label}
        </button>
    );
}

function FilterOptionCard({ description, isSelected, label, onClick }) {
    return (
        <button
            type="button"
            onClick={onClick}
            className={`w-full rounded-2xl border p-4 text-left transition ${
                isSelected
                    ? "border-[#9B7BFF]/70 bg-[#9B7BFF]/12"
                    : "border-white/10 bg-slate-950/35 hover:border-white/20 hover:bg-white/10"
            }`}
        >
            <div className="flex items-start justify-between gap-4">
                <div>
                    <p className="text-base font-semibold text-slate-100">{label}</p>
                    <p className="mt-2 text-sm leading-6 text-slate-300">{description}</p>
                </div>
                <span
                    className={`mt-1 inline-flex h-5 w-5 shrink-0 rounded-full border ${
                        isSelected ? "border-[#9B7BFF] bg-[#9B7BFF]" : "border-white/20 bg-transparent"
                    }`}
                    aria-hidden="true"
                >
                    {isSelected ? <span className="m-auto h-2 w-2 rounded-full bg-white" /> : null}
                </span>
            </div>
        </button>
    );
}

function ArchiveFilterSheet({ draftFilters, entityTitle, isOpen, onApply, onClose, onDraftChange, onReset }) {
    return (
        <SlideOverSheet
            isOpen={isOpen}
            onClose={onClose}
            closeLabel="Закрыть фильтры архива"
            eyebrow="Архив"
            title={`Фильтры и группировка${entityTitle ? `: ${entityTitle}` : ""}`}
            panelClassName="lg:w-[38rem] xl:w-[42rem]"
        >
            <div className="mt-8 space-y-6 pb-28">
                <FilterSection
                    title="Быстрые фильтры"
                    description="Подготовили базовые переключатели для архива. Логику их применения можно будет подключить следующим шагом."
                >
                    <div className="flex flex-wrap gap-3">
                        <FilterToggleChip
                            isActive={draftFilters.onlyUrgent}
                            label="Только срочные"
                            onClick={() => onDraftChange("onlyUrgent", !draftFilters.onlyUrgent)}
                        />
                        <FilterToggleChip
                            isActive={draftFilters.onlyWithExecutor}
                            label="С назначенным исполнителем"
                            onClick={() => onDraftChange("onlyWithExecutor", !draftFilters.onlyWithExecutor)}
                        />
                        <FilterToggleChip
                            isActive={draftFilters.onlyWithResult}
                            label="С заполненным результатом"
                            onClick={() => onDraftChange("onlyWithResult", !draftFilters.onlyWithResult)}
                        />
                    </div>
                </FilterSection>

                <FilterSection title="Группировка" description="Выберите, как раскладывать тикеты внутри архива.">
                    <div className="space-y-3">
                        {archiveGroupingOptions.map((option) => (
                            <FilterOptionCard
                                key={option.id}
                                description={option.description}
                                isSelected={draftFilters.groupBy === option.id}
                                label={option.label}
                                onClick={() => onDraftChange("groupBy", option.id)}
                            />
                        ))}
                    </div>
                </FilterSection>

                <FilterSection title="Сортировка" description="Определяет порядок карточек внутри выбранной вкладки.">
                    <div className="space-y-3">
                        {archiveSortOptions.map((option) => (
                            <FilterOptionCard
                                key={option.id}
                                description={option.description}
                                isSelected={draftFilters.sortBy === option.id}
                                label={option.label}
                                onClick={() => onDraftChange("sortBy", option.id)}
                            />
                        ))}
                    </div>
                </FilterSection>
            </div>

            <div className="sticky -bottom-6 -mx-6 flex gap-3 border-t border-white/15 bg-slate-950/95 px-6 py-5 backdrop-blur">
                <button
                    type="button"
                    onClick={onReset}
                    className="min-h-14 flex-1 rounded-2xl border border-white/10 bg-white/5 px-5 text-base font-semibold text-slate-100 transition hover:border-white/20 hover:bg-white/10"
                >
                    Сбросить
                </button>
                <button
                    type="button"
                    onClick={onApply}
                    className="min-h-14 flex-1 rounded-2xl bg-[#6A3BF2] px-5 text-base font-semibold text-white transition hover:bg-[#7C52F5]"
                >
                    Применить
                </button>
            </div>
        </SlideOverSheet>
    );
}

function ArchiveTicketsSection({
    activeTab,
    emptyMessage,
    groupedTickets,
    groupingLabel,
    isError,
    isLoading,
    loadingMessage,
    onChangeTab,
    onOpenFilters,
    onOpenTicket,
    tickets,
    ticketsErrorMessage,
    ticketsHeading,
}) {
    return (
        <section className="space-y-4">
            <h2 className="text-2xl font-bold tracking-tight text-slate-50 sm:text-3xl">{ticketsHeading}</h2>

            {!isLoading && !isError ? (
                <p className="text-sm text-slate-300">
                    Найдено: <span className="font-semibold text-slate-100">{tickets.length}</span>
                    <span className="mx-2 text-slate-500">•</span>
                    Группировка: <span className="font-semibold text-slate-100">{groupingLabel}</span>
                </p>
            ) : null}

            <div className="flex items-center justify-between gap-3 rounded-3xl border border-white/10 bg-slate-950/25 p-4 shadow-lg shadow-black/10 backdrop-blur">
                <div className="flex min-w-0 flex-1 gap-2 overflow-x-auto pr-2">
                    {archiveTabs.map((tab) => (
                        <ArchiveTabButton
                            key={tab.id}
                            isActive={tab.id === activeTab}
                            label={tab.label}
                            onClick={() => onChangeTab(tab.id)}
                        />
                    ))}
                </div>

                <div className="shrink-0">
                    <ArchiveFilterButton onClick={onOpenFilters} />
                </div>
            </div>

            {isLoading ? (
                <div className="rounded-3xl border border-white/10 bg-white/5 p-6">
                    <p className="text-sm text-slate-300">{loadingMessage}</p>
                </div>
            ) : null}

            {isError ? (
                <div className="rounded-3xl border border-rose-300/30 bg-rose-500/10 p-6">
                    <p className="text-sm text-rose-100">{ticketsErrorMessage}</p>
                </div>
            ) : null}

            {!isLoading && !isError && tickets.length > 0 ? (
                <div className="space-y-5">
                    {groupedTickets.map((group) => (
                        <section key={group.label} className="space-y-3">
                            <div className="sticky top-4 z-10 rounded-2xl border border-white/10 bg-slate-950/85 px-4 py-3 shadow-lg shadow-black/20 backdrop-blur">
                                <div className="flex items-center justify-between gap-4">
                                    <p className="text-sm font-semibold text-slate-200 sm:text-base">{group.label}</p>
                                    <span className="rounded-full border border-white/10 bg-white/5 px-3 py-1 text-xs font-semibold text-slate-300">
                                        {group.items.length}
                                    </span>
                                </div>
                            </div>

                            <div className="grid gap-3">
                                {group.items.map((ticket) => (
                                    <TicketCardWithExecutor
                                        key={ticket.id}
                                        ticket={ticket}
                                        executor={{
                                            department: ticket.executorDepartment,
                                            name: ticket.executorName,
                                        }}
                                        onOpenTicket={onOpenTicket}
                                    />
                                ))}
                            </div>
                        </section>
                    ))}
                </div>
            ) : null}

            {!isLoading && !isError && tickets.length === 0 ? (
                <div className="rounded-3xl border border-white/10 bg-white/5 p-6">
                    <p className="text-sm text-slate-300">{emptyMessage}</p>
                </div>
            ) : null}
        </section>
    );
}

export function TicketArchivePage({ entityType }) {
    const navigate = useNavigate();
    const { clientId, deviceId } = useParams();
    const [activeTab, setActiveTab] = useState("closed");
    const [activeGrouping, setActiveGrouping] = useState("months");
    const [isFilterSheetOpen, setIsFilterSheetOpen] = useState(false);
    const [filterDraft, setFilterDraft] = useState({
        groupBy: "months",
        onlyUrgent: false,
        onlyWithExecutor: false,
        onlyWithResult: false,
        sortBy: "newest",
    });
    const config = archiveConfigByEntity[entityType] || archiveConfigByEntity.client;
    const isClientArchive = entityType === "client";
    const entityId = isClientArchive ? clientId : deviceId;

    const {
        data: client,
        isError: isClientError,
        isFetching: isClientFetching,
        isLoading: isClientLoading,
    } = useGetClientByIdQuery(entityId, {
        skip: !entityId || !isClientArchive,
    });
    const {
        data: device,
        isError: isDeviceError,
        isFetching: isDeviceFetching,
        isLoading: isDeviceLoading,
    } = useGetDeviceByIdQuery(entityId, {
        skip: !entityId || isClientArchive,
    });
    const {
        data: clientTickets = [],
        isError: isClientTicketsError,
        isFetching: isClientTicketsFetching,
        isLoading: isClientTicketsLoading,
    } = useGetClientTicketsQuery(
        {
            clientId: entityId,
            limit: 100,
        },
        {
            skip: !entityId || !isClientArchive,
        },
    );
    const {
        data: deviceTickets = [],
        isError: isDeviceTicketsError,
        isFetching: isDeviceTicketsFetching,
        isLoading: isDeviceTicketsLoading,
    } = useGetDeviceTicketsQuery(
        {
            deviceId: entityId,
            limit: 100,
        },
        {
            skip: !entityId || isClientArchive,
        },
    );

    const entity = isClientArchive ? client : device;
    const tickets = isClientArchive ? clientTickets : deviceTickets;
    const isEntityError = isClientArchive ? isClientError : isDeviceError;
    const isEntityLoading = isClientArchive ? isClientLoading || isClientFetching : isDeviceLoading || isDeviceFetching;
    const isTicketsError = isClientArchive ? isClientTicketsError : isDeviceTicketsError;
    const isTicketsLoading = isClientArchive
        ? isClientTicketsLoading || isClientTicketsFetching
        : isDeviceTicketsLoading || isDeviceTicketsFetching;
    const entityTitle = isClientArchive
        ? entity?.title?.trim() || config.entityFallbackTitle
        : entity?.title?.trim() || config.entityFallbackTitle;
    const entityMeta = isClientArchive
        ? entity?.address?.trim() || "Адрес не указан"
        : entity?.serialNumber?.trim()
          ? `С/Н: ${entity.serialNumber.trim()}`
          : "Серийный номер не указан";
    const filteredTickets = filterTicketsByTab(tickets, activeTab);
    const groupedTickets = groupTickets(filteredTickets, activeGrouping);

    function handleBack() {
        if (!entityId) {
            navigate(-1);
            return;
        }

        navigate(isClientArchive ? routePaths.clientById(entityId) : routePaths.deviceById(entityId));
    }

    function handleDraftChange(key, value) {
        setFilterDraft((currentValue) => ({
            ...currentValue,
            [key]: value,
        }));
    }

    function handleResetDraft() {
        setFilterDraft({
            groupBy: "months",
            onlyUrgent: false,
            onlyWithExecutor: false,
            onlyWithResult: false,
            sortBy: "newest",
        });
    }

    function handleApplyDraft() {
        setActiveGrouping(filterDraft.groupBy);
        setIsFilterSheetOpen(false);
    }

    return (
        <PageShell>
            <section className="w-full space-y-6">
                <ArchiveHeader entityMeta={entityMeta} entityTitle={entityTitle} onBack={handleBack} />

                {isEntityLoading ? (
                    <div className="rounded-3xl border border-white/10 bg-white/5 p-6">
                        <p className="text-sm text-slate-300">{config.loadingEntityMessage}</p>
                    </div>
                ) : null}

                {isEntityError ? (
                    <div className="rounded-3xl border border-rose-300/30 bg-rose-500/10 p-6">
                        <p className="text-sm text-rose-100">Не удалось загрузить данные страницы.</p>
                    </div>
                ) : null}

                {!isEntityLoading && !isEntityError ? (
                    <ArchiveTicketsSection
                        activeTab={activeTab}
                        emptyMessage={resolveEmptyMessage(config, activeTab)}
                        groupedTickets={groupedTickets}
                        groupingLabel={resolveGroupingLabel(activeGrouping)}
                        isError={isTicketsError}
                        isLoading={isTicketsLoading}
                        loadingMessage={config.loadingTicketsMessage}
                        onChangeTab={setActiveTab}
                        onOpenFilters={() => setIsFilterSheetOpen(true)}
                        onOpenTicket={(ticketId) => navigate(routePaths.ticketById(ticketId))}
                        tickets={filteredTickets}
                        ticketsErrorMessage={config.ticketsErrorMessage}
                        ticketsHeading={config.ticketsHeading}
                    />
                ) : null}
            </section>

            <ArchiveFilterSheet
                draftFilters={filterDraft}
                entityTitle={entityTitle}
                isOpen={isFilterSheetOpen}
                onApply={handleApplyDraft}
                onClose={() => setIsFilterSheetOpen(false)}
                onDraftChange={handleDraftChange}
                onReset={handleResetDraft}
            />
        </PageShell>
    );
}
