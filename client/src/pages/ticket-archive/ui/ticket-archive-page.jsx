import { useRef, useState } from "react";
import { useNavigate, useParams } from "react-router";
import {
    useGetClientByIdQuery,
    useGetClientTicketsQuery,
    useGetDeviceByIdQuery,
    useGetDeviceTicketsQuery,
} from "../../../shared/api/tickets-api";
import { routePaths } from "../../../shared/config/routes";
import { PageShell } from "../../../shared/ui/page-shell";
import { SelectField } from "../../../shared/ui/select-field";
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

function filterTicketsByPeriod(tickets, period) {
    if (period?.useEntirePeriod) {
        return tickets;
    }

    if (!period?.startDate && !period?.endDate) {
        return tickets;
    }

    const startTimestamp = period.startDate ? new Date(`${period.startDate}T00:00:00`).getTime() : null;
    const endTimestamp = period.endDate ? new Date(`${period.endDate}T23:59:59`).getTime() : null;

    return tickets.filter((ticket) => {
        const timelineDate = resolveTicketTimelineDate(ticket);
        if (!timelineDate) {
            return false;
        }

        const ticketTimestamp = new Date(timelineDate).getTime();
        if (Number.isNaN(ticketTimestamp)) {
            return false;
        }

        if (startTimestamp !== null && ticketTimestamp < startTimestamp) {
            return false;
        }

        if (endTimestamp !== null && ticketTimestamp > endTimestamp) {
            return false;
        }

        return true;
    });
}

function filterTicketsByReason(tickets, reasonTitle) {
    if (!reasonTitle) {
        return tickets;
    }

    return tickets.filter((ticket) => (ticket?.reasonTitle?.trim() || ticket?.reason?.trim() || "") === reasonTitle);
}

function filterTicketsByDevice(tickets, deviceName) {
    if (!deviceName) {
        return tickets;
    }

    return tickets.filter((ticket) => (ticket?.deviceName?.trim() || "") === deviceName);
}

function getReasonOptions(tickets) {
    return Array.from(
        new Set(
            tickets
                .map((ticket) => ticket?.reasonTitle?.trim() || ticket?.reason?.trim() || "")
                .filter(Boolean),
        ),
    ).sort((left, right) => left.localeCompare(right, "ru"));
}

function getDeviceOptions(tickets) {
    return Array.from(
        new Set(
            tickets
                .map((ticket) => ticket?.deviceName?.trim() || "")
                .filter(Boolean),
        ),
    ).sort((left, right) => left.localeCompare(right, "ru"));
}

function sortTicketsByDate(tickets, sortBy) {
    const direction = sortBy === "oldest" ? 1 : -1;

    return [...tickets].sort((left, right) => {
        const leftTimestamp = new Date(resolveTicketTimelineDate(left) || 0).getTime();
        const rightTimestamp = new Date(resolveTicketTimelineDate(right) || 0).getTime();
        const safeLeftTimestamp = Number.isNaN(leftTimestamp) ? 0 : leftTimestamp;
        const safeRightTimestamp = Number.isNaN(rightTimestamp) ? 0 : rightTimestamp;

        if (safeLeftTimestamp !== safeRightTimestamp) {
            return (safeLeftTimestamp - safeRightTimestamp) * direction;
        }

        const leftNumber = Number(left?.number || 0);
        const rightNumber = Number(right?.number || 0);
        if (leftNumber !== rightNumber) {
            return (leftNumber - rightNumber) * direction;
        }

        return String(left?.id || "").localeCompare(String(right?.id || ""));
    });
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
    groupedTickets,
    groupingLabel,
    isError,
    isLoading,
    loadingMessage,
    onChangeTab,
    onOpenFilters,
    onOpenTicket,
    sortLabel,
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
    const draftPeriodTickets = filterTicketsByPeriod(tickets, {
        endDate: filterDraft.endDate,
        startDate: filterDraft.startDate,
        useEntirePeriod: filterDraft.useEntirePeriod,
    });
    const draftTabTickets = filterTicketsByTab(draftPeriodTickets, activeTab);
    const draftReasonTickets = filterTicketsByReason(draftTabTickets, filterDraft.reasonTitle);
    const reasonOptions = getReasonOptions(filterTicketsByDevice(draftTabTickets, filterDraft.deviceName));
    const deviceOptions = getDeviceOptions(draftReasonTickets);
    const periodFilteredTickets = filterTicketsByPeriod(tickets, activePeriod);
    const tabFilteredTickets = filterTicketsByTab(periodFilteredTickets, activeTab);
    const reasonFilteredTickets = filterTicketsByReason(tabFilteredTickets, activeReasonTitle);
    const deviceFilteredTickets = filterTicketsByDevice(reasonFilteredTickets, activeDeviceName);
    const filteredTickets = sortTicketsByDate(deviceFilteredTickets, activeSortBy);
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
                        sortLabel={resolveSortLabel(activeSortBy)}
                        tickets={filteredTickets}
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
