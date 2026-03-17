import { useEffect, useRef, useState } from "react";
import { useNavigate, useParams } from "react-router";
import {
    useGetClientByIdQuery,
    useGetClientTicketArchiveFacetsQuery,
    useGetClientTicketsPageQuery,
    useGetDeviceByIdQuery,
    useGetDeviceTicketArchiveFacetsQuery,
    useGetDeviceTicketsPageQuery,
    useGetProfileByIdQuery,
    useGetProfileTicketArchiveFacetsQuery,
    useGetProfileTicketsPageQuery,
} from "../../../shared/api/tickets-api";
import { routePaths } from "../../../shared/config/routes";
import { ProfileTicketCard } from "../../../shared/ui/profile-ticket-card";
import { PageShell } from "../../../shared/ui/page-shell";
import { SelectField } from "../../../shared/ui/select-field";
import { SlideOverSheet } from "../../../shared/ui/slide-over-sheet";
import { TicketCardWithExecutor } from "../../dashboard/ui/ticket-card-with-executor";

const ARCHIVE_PAGE_SIZE = 100;

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
    profile: {
        emptyAllMessage: "У этого сотрудника пока нет выездов.",
        emptyClosedMessage: "У этого сотрудника пока нет закрытых выездов.",
        emptyInWorkMessage: "У этого сотрудника сейчас нет выездов в работе.",
        entityFallbackTitle: "Сотрудник",
        loadingEntityMessage: "Загрузка профиля сотрудника...",
        loadingTicketsMessage: "Загрузка архива выездов...",
        ticketsHeading: "Все выезды сотрудника",
        ticketsErrorMessage: "Не удалось загрузить архив выездов сотрудника.",
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

function resolveEmptyMessage(config, activeTab) {
    if (activeTab === "closed") {
        return config.emptyClosedMessage;
    }

    if (activeTab === "inWork") {
        return config.emptyInWorkMessage;
    }

    return config.emptyAllMessage;
}

function formatDateFieldValue(value) {
    if (!value) {
        return "";
    }

    const date = new Date(`${value}T00:00:00`);
    if (Number.isNaN(date.getTime())) {
        return "";
    }

    const day = String(date.getDate()).padStart(2, "0");
    const month = String(date.getMonth() + 1).padStart(2, "0");
    const year = date.getFullYear();

    return `${day}.${month}.${year}`;
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
        return ticket?.reasonTitle?.trim() || ticket?.reason?.trim() || "Причина не указана";
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

function resolveSortLabel(sortBy) {
    return archiveSortOptions.find((option) => option.id === sortBy)?.label || "Сначала новые";
}

function resolveStatusFilter(activeTab) {
    if (activeTab === "all") {
        return "";
    }

    return activeTab;
}

function includeSelectedOption(options, selectedValue) {
    if (!selectedValue || options.includes(selectedValue)) {
        return options;
    }

    return [selectedValue, ...options];
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

function PeriodDateField({ max, min, name, onChange, value }) {
    const hasValue = Boolean(value);
    const inputRef = useRef(null);

    function openPicker() {
        const input = inputRef.current;
        if (!input) {
            return;
        }

        input.focus();
        if (typeof input.showPicker === "function") {
            input.showPicker();
        }
    }

    return (
        <label className="group relative block min-w-0 flex-1 cursor-pointer" onClick={openPicker}>
            <input
                ref={inputRef}
                type="date"
                name={name}
                value={value}
                min={min}
                max={max}
                onChange={onChange}
                className="peer absolute inset-0 z-10 cursor-pointer opacity-0"
            />

            <span className="flex min-h-16 items-center justify-between rounded-2xl border border-white/10 bg-slate-950/35 px-4 py-3 text-left transition group-hover:border-white/20 group-hover:bg-white/10 peer-focus:border-[#9B7BFF]/70 peer-focus:ring-2 peer-focus:ring-[#9B7BFF]/20">
                <span className={`text-lg ${hasValue ? "text-slate-100" : "text-slate-400"}`}>
                    {hasValue ? formatDateFieldValue(value) : "Выберите дату"}
                </span>
                <svg
                    xmlns="http://www.w3.org/2000/svg"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    strokeWidth="2"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    className="h-5 w-5 shrink-0 text-slate-400 transition group-hover:text-slate-200"
                    aria-hidden="true"
                >
                    <path d="m6 9 6 6 6-6" />
                </svg>
            </span>
        </label>
    );
}

function ArchiveFilterSheet({
    deviceOptions,
    draftFilters,
    entityTitle,
    isOpen,
    isClientArchive,
    onApply,
    onClose,
    onDraftChange,
    onReset,
    reasonOptions,
}) {
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
                    title="Период"
                    description="Ограничьте архив выбранным диапазоном дат или вернитесь к полному периоду."
                >
                    <div className="space-y-3">
                        <div className="flex flex-col gap-3 lg:flex-row lg:items-center">
                            <PeriodDateField
                                max={draftFilters.endDate || undefined}
                                name="period_start"
                                onChange={(event) => {
                                    onDraftChange("startDate", event.target.value);
                                    onDraftChange("useEntirePeriod", false);
                                }}
                                value={draftFilters.startDate}
                            />

                            <span className="hidden shrink-0 text-lg text-slate-400 lg:block">-</span>

                            <PeriodDateField
                                min={draftFilters.startDate || undefined}
                                name="period_end"
                                onChange={(event) => {
                                    onDraftChange("endDate", event.target.value);
                                    onDraftChange("useEntirePeriod", false);
                                }}
                                value={draftFilters.endDate}
                            />
                        </div>

                        <button
                            type="button"
                            onClick={() => {
                                onDraftChange("startDate", "");
                                onDraftChange("endDate", "");
                                onDraftChange("useEntirePeriod", true);
                            }}
                            className={`min-h-16 w-full rounded-2xl border px-5 text-base font-semibold transition ${
                                draftFilters.useEntirePeriod
                                    ? "border-[#9B7BFF] bg-[#9B7BFF]/15 text-[#E7DEFF]"
                                    : "border-white/10 bg-slate-950/35 text-slate-300 hover:border-white/20 hover:bg-white/10 hover:text-slate-100"
                            }`}
                        >
                            <span className="inline-flex items-center gap-3">
                                <span
                                    className={`inline-flex h-6 w-6 items-center justify-center rounded-full border ${
                                        draftFilters.useEntirePeriod
                                            ? "border-[#9B7BFF] bg-[#9B7BFF] text-white"
                                            : "border-white/15 text-transparent"
                                    }`}
                                    aria-hidden="true"
                                >
                                    <svg
                                        xmlns="http://www.w3.org/2000/svg"
                                        viewBox="0 0 24 24"
                                        fill="none"
                                        stroke="currentColor"
                                        strokeWidth="2.5"
                                        strokeLinecap="round"
                                        strokeLinejoin="round"
                                        className="h-4 w-4"
                                    >
                                        <path d="M20 6 9 17l-5-5" />
                                    </svg>
                                </span>
                                <span>Весь период</span>
                            </span>
                        </button>
                    </div>
                </FilterSection>

                <FilterSection
                    title="Причина выезда"
                    description="Показываем только причины, которые есть в текущей выборке архива."
                >
                    <SelectField
                        name="ticket_reason"
                        onChange={(event) => onDraftChange("reasonTitle", event.target.value)}
                        value={draftFilters.reasonTitle}
                    >
                        <option value="">Все причины</option>
                        {reasonOptions.map((option) => (
                            <option key={option} value={option}>
                                {option}
                            </option>
                        ))}
                    </SelectField>
                </FilterSection>

                {isClientArchive ? (
                    <FilterSection
                        title="Оборудование"
                        description="Фильтр по приборам клиента на основе названий из текущего списка тикетов."
                    >
                        <SelectField
                            name="ticket_device"
                            onChange={(event) => onDraftChange("deviceName", event.target.value)}
                            value={draftFilters.deviceName}
                        >
                            <option value="">Все приборы</option>
                            {deviceOptions.map((option) => (
                                <option key={option} value={option}>
                                    {option}
                                </option>
                            ))}
                        </SelectField>
                    </FilterSection>
                ) : null}

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
    entityType,
    groupedTickets,
    groupingLabel,
    isError,
    isLoading,
    loadingMessage,
    onChangeTab,
    onOpenFilters,
    onOpenTicket,
    onPageChange,
    pagination,
    sortLabel,
    tickets,
    ticketsErrorMessage,
    ticketsHeading,
}) {
    const [expandedGroups, setExpandedGroups] = useState(() => new Set());
    const totalPages = pagination.total > 0 ? Math.ceil(pagination.total / pagination.limit) : 0;
    const currentPage = totalPages > 0 ? Math.floor(pagination.offset / pagination.limit) + 1 : 0;
    const currentRangeStart = pagination.total > 0 ? pagination.offset + 1 : 0;
    const currentRangeEnd = pagination.total > 0 ? pagination.offset + pagination.pageItemsCount : 0;
    const visibleGroupsSignature = groupedTickets
        .map((group) => `${group.label}:${group.items.map((ticket) => ticket.id).join(",")}`)
        .join("|");

    useEffect(() => {
        setExpandedGroups(new Set());
    }, [visibleGroupsSignature]);

    function handleToggleGroup(groupLabel) {
        setExpandedGroups((currentValue) => {
            const nextValue = new Set(currentValue);

            if (nextValue.has(groupLabel)) {
                nextValue.delete(groupLabel);
            } else {
                nextValue.add(groupLabel);
            }

            return nextValue;
        });
    }

    return (
        <section className="space-y-4">
            <h2 className="text-2xl font-bold tracking-tight text-slate-50 sm:text-3xl">{ticketsHeading}</h2>

            {!isLoading && !isError ? (
                <p className="text-sm text-slate-300">
                    На странице после фильтров: <span className="font-semibold text-slate-100">{tickets.length}</span>
                    <span className="mx-2 text-slate-500">•</span>
                    Всего в архиве: <span className="font-semibold text-slate-100">{pagination.total}</span>
                    <span className="mx-2 text-slate-500">•</span>
                    Группировка: <span className="font-semibold text-slate-100">{groupingLabel}</span>
                    <span className="mx-2 text-slate-500">•</span>
                    Сортировка: <span className="font-semibold text-slate-100">{sortLabel}</span>
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
                            <div className="sticky top-4 z-10">
                                <button
                                    type="button"
                                    onClick={() => handleToggleGroup(group.label)}
                                    aria-expanded={expandedGroups.has(group.label)}
                                    className="w-full rounded-2xl border border-white/10 bg-slate-950/85 px-4 py-3 text-left shadow-lg shadow-black/20 backdrop-blur transition hover:border-white/20 hover:bg-slate-950"
                                >
                                    <div className="flex items-center justify-between gap-4">
                                        <div className="min-w-0">
                                            <p className="text-sm font-semibold text-slate-200 sm:text-base">{group.label}</p>
                                            <p className="mt-1 text-xs text-slate-400">
                                                {expandedGroups.has(group.label) ? "Скрыть записи" : "Показать записи"}
                                            </p>
                                        </div>

                                        <div className="flex shrink-0 items-center gap-3">
                                            <span className="rounded-full border border-white/10 bg-white/5 px-3 py-1 text-xs font-semibold text-slate-300">
                                                {group.items.length}
                                            </span>
                                            <svg
                                                xmlns="http://www.w3.org/2000/svg"
                                                viewBox="0 0 24 24"
                                                fill="none"
                                                stroke="currentColor"
                                                strokeWidth="1.9"
                                                strokeLinecap="round"
                                                strokeLinejoin="round"
                                                className={`h-5 w-5 text-slate-300 transition-transform ${
                                                    expandedGroups.has(group.label) ? "rotate-180" : ""
                                                }`}
                                                aria-hidden="true"
                                            >
                                                <path d="m6 9 6 6 6-6" />
                                            </svg>
                                        </div>
                                    </div>
                                </button>
                            </div>

                            {expandedGroups.has(group.label) ? (
                                <div className="grid gap-3">
                                    {group.items.map((ticket) =>
                                        entityType === "profile" ? (
                                            <ProfileTicketCard key={ticket.id} ticket={ticket} onOpenTicket={onOpenTicket} />
                                        ) : (
                                            <TicketCardWithExecutor
                                                key={ticket.id}
                                                ticket={ticket}
                                                executor={{
                                                    department: ticket.executorDepartment,
                                                    name: ticket.executorName,
                                                }}
                                                onOpenTicket={onOpenTicket}
                                            />
                                        ),
                                    )}
                                </div>
                            ) : null}
                        </section>
                    ))}
                </div>
            ) : null}

            {!isLoading && !isError && tickets.length === 0 ? (
                <div className="rounded-3xl border border-white/10 bg-white/5 p-6">
                    <p className="text-sm text-slate-300">{emptyMessage}</p>
                </div>
            ) : null}

            {!isLoading && !isError && totalPages > 1 ? (
                <div className="flex flex-col gap-3 rounded-3xl border border-white/10 bg-slate-950/25 p-4 shadow-lg shadow-black/10 backdrop-blur sm:flex-row sm:items-center sm:justify-between">
                    <div className="space-y-1">
                        <p className="text-sm font-semibold text-slate-100">
                            Страница {currentPage} из {totalPages}
                        </p>
                        <p className="text-sm text-slate-300">
                            Записи {currentRangeStart}-{currentRangeEnd} из {pagination.total}
                        </p>
                    </div>

                    <div className="flex gap-3">
                        <button
                            type="button"
                            onClick={() => onPageChange(-1)}
                            disabled={!pagination.hasPrev}
                            className="min-h-12 rounded-2xl border border-white/10 bg-white/5 px-4 text-sm font-semibold text-slate-100 transition hover:border-white/20 hover:bg-white/10 disabled:cursor-not-allowed disabled:opacity-50"
                        >
                            Назад
                        </button>
                        <button
                            type="button"
                            onClick={() => onPageChange(1)}
                            disabled={!pagination.hasNext}
                            className="min-h-12 rounded-2xl bg-[#6A3BF2] px-4 text-sm font-semibold text-white transition hover:bg-[#7C52F5] disabled:cursor-not-allowed disabled:opacity-50"
                        >
                            Дальше
                        </button>
                    </div>
                </div>
            ) : null}
        </section>
    );
}

export function TicketArchivePage({ entityType }) {
    const navigate = useNavigate();
    const { clientId, deviceId, userId } = useParams();
    const [activeTab, setActiveTab] = useState("closed");
    const [activeGrouping, setActiveGrouping] = useState("months");
    const [activeDeviceName, setActiveDeviceName] = useState("");
    const [activeSortBy, setActiveSortBy] = useState("newest");
    const [isFilterSheetOpen, setIsFilterSheetOpen] = useState(false);
    const [filterDraft, setFilterDraft] = useState({
        deviceName: "",
        endDate: "",
        groupBy: "months",
        reasonTitle: "",
        startDate: "",
        sortBy: "newest",
        useEntirePeriod: true,
    });
    const [activePeriod, setActivePeriod] = useState({
        endDate: "",
        startDate: "",
        useEntirePeriod: true,
    });
    const [activeReasonTitle, setActiveReasonTitle] = useState("");
    const [page, setPage] = useState(1);
    const config = archiveConfigByEntity[entityType] || archiveConfigByEntity.client;
    const isClientArchive = entityType === "client";
    const isProfileArchive = entityType === "profile";
    const entityId = isClientArchive ? clientId : isProfileArchive ? userId : deviceId;
    const archiveOffset = (page - 1) * ARCHIVE_PAGE_SIZE;
    const activeStatus = resolveStatusFilter(activeTab);
    const activeStartDate = activePeriod.useEntirePeriod ? "" : activePeriod.startDate;
    const activeEndDate = activePeriod.useEntirePeriod ? "" : activePeriod.endDate;
    const draftStartDate = filterDraft.useEntirePeriod ? "" : filterDraft.startDate;
    const draftEndDate = filterDraft.useEntirePeriod ? "" : filterDraft.endDate;

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
        skip: !entityId || isClientArchive || isProfileArchive,
    });
    const {
        data: profile,
        isError: isProfileError,
        isFetching: isProfileFetching,
        isLoading: isProfileLoading,
    } = useGetProfileByIdQuery(entityId, {
        skip: !entityId || !isProfileArchive,
    });
    const {
        data: clientTicketsPage,
        isError: isClientTicketsError,
        isFetching: isClientTicketsFetching,
        isLoading: isClientTicketsLoading,
    } = useGetClientTicketsPageQuery(
        {
            clientId: entityId,
            deviceName: activeDeviceName,
            endDate: activeEndDate,
            limit: ARCHIVE_PAGE_SIZE,
            offset: archiveOffset,
            reasonTitle: activeReasonTitle,
            sortBy: activeSortBy,
            startDate: activeStartDate,
            status: activeStatus,
        },
        {
            skip: !entityId || !isClientArchive,
        },
    );
    const {
        data: deviceTicketsPage,
        isError: isDeviceTicketsError,
        isFetching: isDeviceTicketsFetching,
        isLoading: isDeviceTicketsLoading,
    } = useGetDeviceTicketsPageQuery(
        {
            deviceId: entityId,
            deviceName: activeDeviceName,
            endDate: activeEndDate,
            limit: ARCHIVE_PAGE_SIZE,
            offset: archiveOffset,
            reasonTitle: activeReasonTitle,
            sortBy: activeSortBy,
            startDate: activeStartDate,
            status: activeStatus,
        },
        {
            skip: !entityId || isClientArchive || isProfileArchive,
        },
    );
    const {
        data: profileTicketsPage,
        isError: isProfileTicketsError,
        isFetching: isProfileTicketsFetching,
        isLoading: isProfileTicketsLoading,
    } = useGetProfileTicketsPageQuery(
        {
            deviceName: activeDeviceName,
            endDate: activeEndDate,
            limit: ARCHIVE_PAGE_SIZE,
            offset: archiveOffset,
            reasonTitle: activeReasonTitle,
            sortBy: activeSortBy,
            startDate: activeStartDate,
            status: activeStatus,
            userId: entityId,
        },
        {
            skip: !entityId || !isProfileArchive,
        },
    );
    const {
        data: clientArchiveFacets,
    } = useGetClientTicketArchiveFacetsQuery(
        {
            clientId: entityId,
            deviceName: filterDraft.deviceName,
            endDate: draftEndDate,
            reasonTitle: filterDraft.reasonTitle,
            startDate: draftStartDate,
            status: activeStatus,
        },
        {
            skip: !entityId || !isClientArchive,
        },
    );
    const {
        data: deviceArchiveFacets,
    } = useGetDeviceTicketArchiveFacetsQuery(
        {
            deviceId: entityId,
            endDate: draftEndDate,
            reasonTitle: filterDraft.reasonTitle,
            startDate: draftStartDate,
            status: activeStatus,
        },
        {
            skip: !entityId || isClientArchive || isProfileArchive,
        },
    );
    const {
        data: profileArchiveFacets,
    } = useGetProfileTicketArchiveFacetsQuery(
        {
            deviceName: filterDraft.deviceName,
            endDate: draftEndDate,
            reasonTitle: filterDraft.reasonTitle,
            startDate: draftStartDate,
            status: activeStatus,
            userId: entityId,
        },
        {
            skip: !entityId || !isProfileArchive,
        },
    );

    const entity = isClientArchive ? client : isProfileArchive ? profile : device;
    const ticketsPage = isClientArchive ? clientTicketsPage : isProfileArchive ? profileTicketsPage : deviceTicketsPage;
    const archiveFacets = isClientArchive
        ? clientArchiveFacets
        : isProfileArchive
          ? profileArchiveFacets
          : deviceArchiveFacets;
    const tickets = ticketsPage?.items || [];
    const isEntityError = isClientArchive ? isClientError : isProfileArchive ? isProfileError : isDeviceError;
    const isEntityLoading = isClientArchive
        ? isClientLoading || isClientFetching
        : isProfileArchive
          ? isProfileLoading || isProfileFetching
          : isDeviceLoading || isDeviceFetching;
    const isTicketsError = isClientArchive
        ? isClientTicketsError
        : isProfileArchive
          ? isProfileTicketsError
          : isDeviceTicketsError;
    const isTicketsLoading = isClientArchive
        ? isClientTicketsLoading || isClientTicketsFetching
        : isProfileArchive
          ? isProfileTicketsLoading || isProfileTicketsFetching
          : isDeviceTicketsLoading || isDeviceTicketsFetching;
    const pagination = {
        hasNext: ticketsPage?.hasNext || false,
        hasPrev: ticketsPage?.hasPrev || false,
        limit: ticketsPage?.limit || ARCHIVE_PAGE_SIZE,
        offset: ticketsPage?.offset || 0,
        pageItemsCount: tickets.length,
        total: ticketsPage?.total || 0,
    };
    const entityTitle = isClientArchive
        ? entity?.title?.trim() || config.entityFallbackTitle
        : isProfileArchive
          ? entity?.name?.trim() || entity?.username?.trim() || config.entityFallbackTitle
          : entity?.title?.trim() || config.entityFallbackTitle;
    const entityMeta = isClientArchive
        ? entity?.address?.trim() || "Адрес не указан"
        : isProfileArchive
          ? entity?.department?.trim() || "Отдел не указан"
          : entity?.serialNumber?.trim()
            ? `С/Н: ${entity.serialNumber.trim()}`
            : "Серийный номер не указан";
    const reasonOptions = includeSelectedOption(archiveFacets?.reasonTitles || [], filterDraft.reasonTitle);
    const deviceOptions = includeSelectedOption(archiveFacets?.deviceNames || [], filterDraft.deviceName);
    const groupedTickets = groupTickets(tickets, activeGrouping);

    useEffect(() => {
        setPage(1);
    }, [
        activeDeviceName,
        activeEndDate,
        activeReasonTitle,
        activeSortBy,
        activeStartDate,
        activeStatus,
        entityId,
        isClientArchive,
    ]);

    useEffect(() => {
        if (pagination.total > 0 && archiveOffset >= pagination.total) {
            setPage(Math.ceil(pagination.total / ARCHIVE_PAGE_SIZE));
        }
    }, [archiveOffset, pagination.total]);

    function handleBack() {
        navigate(-1);
    }

    function handleDraftChange(key, value) {
        setFilterDraft((currentValue) => ({
            ...currentValue,
            [key]: value,
        }));
    }

    function handleResetDraft() {
        setFilterDraft({
            deviceName: "",
            endDate: "",
            groupBy: "months",
            reasonTitle: "",
            startDate: "",
            sortBy: "newest",
            useEntirePeriod: true,
        });
    }

    function handleApplyDraft() {
        setActiveDeviceName(filterDraft.deviceName);
        setActiveGrouping(filterDraft.groupBy);
        setActiveSortBy(filterDraft.sortBy);
        setActivePeriod({
            endDate: filterDraft.endDate,
            startDate: filterDraft.startDate,
            useEntirePeriod: filterDraft.useEntirePeriod,
        });
        setActiveReasonTitle(filterDraft.reasonTitle);
        setIsFilterSheetOpen(false);
    }

    function handlePageChange(direction) {
        setPage((currentValue) => Math.max(1, currentValue + direction));
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
                        entityType={entityType}
                        groupedTickets={groupedTickets}
                        groupingLabel={resolveGroupingLabel(activeGrouping)}
                        isError={isTicketsError}
                        isLoading={isTicketsLoading}
                        loadingMessage={config.loadingTicketsMessage}
                        onChangeTab={setActiveTab}
                        onOpenFilters={() => setIsFilterSheetOpen(true)}
                        onOpenTicket={(ticketId) => navigate(routePaths.ticketById(ticketId))}
                        onPageChange={handlePageChange}
                        pagination={pagination}
                        sortLabel={resolveSortLabel(activeSortBy)}
                        tickets={tickets}
                        ticketsErrorMessage={config.ticketsErrorMessage}
                        ticketsHeading={config.ticketsHeading}
                    />
                ) : null}
            </section>

            <ArchiveFilterSheet
                deviceOptions={deviceOptions}
                draftFilters={filterDraft}
                entityTitle={entityTitle}
                isOpen={isFilterSheetOpen}
                isClientArchive={isClientArchive}
                onApply={handleApplyDraft}
                onClose={() => setIsFilterSheetOpen(false)}
                onDraftChange={handleDraftChange}
                onReset={handleResetDraft}
                reasonOptions={reasonOptions}
            />
        </PageShell>
    );
}
