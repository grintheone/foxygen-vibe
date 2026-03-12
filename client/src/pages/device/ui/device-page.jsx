import { useEffect, useRef, useState } from "react";
import { useNavigate, useParams } from "react-router";
import { useAuth } from "../../../features/auth";
import { TicketCardWithExecutor } from "../../dashboard/ui/ticket-card-with-executor";
import {
    useAddCommentMutation,
    useCreateTicketMutation,
    isMissingCommentReferenceError,
    useGetClientContactsQuery,
    useGetCommentsQuery,
    useGetDeviceAgreementsQuery,
    useGetDeviceByIdQuery,
    useGetDeviceTicketsQuery,
    useGetDepartmentMembersQuery,
    useGetTicketReasonsQuery,
} from "../../../shared/api/tickets-api";
import { routePaths } from "../../../shared/config/routes";
import { BottomPageAction } from "../../../shared/ui/bottom-page-action";
import { PageShell } from "../../../shared/ui/page-shell";
import { SelectField } from "../../../shared/ui/select-field";
import { SlideOverSheet } from "../../../shared/ui/slide-over-sheet";

function formatCommentDate(value) {
    if (!value) {
        return "";
    }

    const date = new Date(value);
    if (Number.isNaN(date.getTime())) {
        return "";
    }

    const day = String(date.getDate()).padStart(2, "0");
    const month = String(date.getMonth() + 1).padStart(2, "0");
    const year = date.getFullYear();
    const hours = String(date.getHours()).padStart(2, "0");
    const minutes = String(date.getMinutes()).padStart(2, "0");

    return `${day}.${month}.${year} ${hours}:${minutes}`;
}

function formatShortDate(value) {
    if (!value) {
        return "";
    }

    const date = new Date(value);
    if (Number.isNaN(date.getTime())) {
        return "";
    }

    const day = String(date.getDate()).padStart(2, "0");
    const month = String(date.getMonth() + 1).padStart(2, "0");
    const year = date.getFullYear();

    return `${day}.${month}.${year}`;
}

function formatAgreementRange(agreement) {
    if (!agreement) {
        return "Срок не указан";
    }

    const assignedAt = formatShortDate(agreement.assigned_at);
    const finishedAt = formatShortDate(agreement.finished_at);

    if (!assignedAt && !finishedAt) {
        return "Срок не указан";
    }

    return `с ${assignedAt || "—"} до ${finishedAt || "—"}`;
}

function formatPropertyValue(value) {
    if (typeof value === "boolean") {
        return value ? "Да" : "Нет";
    }

    if (typeof value === "number") {
        return String(value);
    }

    if (typeof value === "string") {
        const normalized = value.trim();
        return normalized || "Не указано";
    }

    if (Array.isArray(value)) {
        return (
            value
                .map((item) => formatPropertyValue(item))
                .filter((item) => item && item !== "Не указано")
                .join(", ") || "Не указано"
        );
    }

    if (value && typeof value === "object") {
        return JSON.stringify(value);
    }

    return "Не указано";
}

function buildPropertyEntries(properties) {
    if (!properties || typeof properties !== "object" || Array.isArray(properties)) {
        return [];
    }

    return Object.entries(properties)
        .filter(([, value]) => value !== null && value !== undefined && value !== "")
        .map(([key, value]) => ({
            label: key,
            value: formatPropertyValue(value),
        }));
}

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

function DeviceHeader({ serialNumber, title, onBack }) {
    return (
        <header className="rounded-3xl border border-white/10 bg-slate-950/35 p-6 shadow-xl shadow-black/20 backdrop-blur">
            <BackButton onClick={onBack} />
            <div className="mt-5 text-left">
                <p className="text-xs font-semibold uppercase tracking-[0.3em] text-slate-400">Оборудование</p>
                <h1 className="mt-3 text-3xl font-bold tracking-tight text-slate-50 sm:text-4xl">
                    {title || "Устройство"}
                </h1>
                <p className="mt-3 text-base text-slate-300 sm:text-lg">
                    С/Н: <span className="font-semibold text-slate-100">{serialNumber || "Не указано"}</span>
                </p>
            </div>
        </header>
    );
}

function DeviceStatCard({ label, value }) {
    return (
        <article className="rounded-3xl border border-white/10 bg-slate-950/35 p-5 shadow-xl shadow-black/20 backdrop-blur">
            <p className="text-xs font-semibold uppercase tracking-[0.24em] text-slate-400">{label}</p>
            <p className="mt-3 text-xl font-semibold tracking-tight text-slate-50 sm:text-2xl">
                {value || "Не указано"}
            </p>
        </article>
    );
}

function DeviceOverviewSection({ device, propertyEntries }) {
    const agreementLabel = device?.agreementNumber ? `Договор #${device.agreementNumber}` : "Договор не найден";
    const agreementMeta = [
        device?.agreementType?.trim() || "",
        device?.agreement ? (device.isActiveAgreement ? "Активный" : "Неактивный") : "",
        device?.agreement ? (device.onWarranty ? "Гарантия" : "Без гарантии") : "",
    ]
        .filter(Boolean)
        .join(" • ");

    return (
        <section className="space-y-4">
            <h2 className="text-2xl font-bold tracking-tight text-slate-50 sm:text-3xl">Сведения</h2>

            <div className="grid gap-3 sm:grid-cols-2">
                <DeviceStatCard label="Серийный номер" value={device?.serialNumber || "Не указано"} />
                <DeviceStatCard label="LIS" value={device?.connectedToLis ? "Подключено" : "Не подключено"} />
                <DeviceStatCard label="Статус" value={device?.isUsed ? "В эксплуатации" : "Не используется"} />
                <DeviceStatCard
                    label="Договор"
                    value={agreementMeta ? `${agreementLabel} • ${agreementMeta}` : agreementLabel}
                />
            </div>

            <div className="rounded-3xl border border-white/10 bg-slate-950/35 p-6 shadow-xl shadow-black/20 backdrop-blur">
                <h3 className="text-lg font-semibold tracking-tight text-slate-100 sm:text-2xl">Параметры</h3>

                {propertyEntries.length > 0 ? (
                    <div className="mt-5 grid gap-3">
                        {propertyEntries.map((entry) => (
                            <div
                                key={entry.label}
                                className="flex flex-col gap-1 rounded-2xl border border-white/10 bg-white/5 px-4 py-3 sm:flex-row sm:items-center sm:justify-between sm:gap-4"
                            >
                                <p className="text-sm font-semibold uppercase tracking-[0.18em] text-slate-400">
                                    {entry.label}
                                </p>
                                <p className="text-base text-slate-100 sm:text-right">{entry.value}</p>
                            </div>
                        ))}
                    </div>
                ) : (
                    <p className="mt-4 text-sm text-slate-300">Дополнительные параметры не указаны.</p>
                )}
            </div>
        </section>
    );
}

function DeviceServiceSection({ agreement, isError, isLoading, onOpenClient, onOpenExpiredAgreements }) {
    return (
        <section className="space-y-4">
            <h2 className="text-2xl font-bold tracking-tight text-slate-50 sm:text-3xl">Сервисные услуги</h2>

            {isLoading ? (
                <div className="rounded-3xl border border-white/10 bg-white/5 p-6">
                    <p className="text-sm text-slate-300">Загрузка сервисных услуг...</p>
                </div>
            ) : null}

            {isError ? (
                <div className="rounded-3xl border border-rose-300/30 bg-rose-500/10 p-6">
                    <p className="text-sm text-rose-100">Не удалось загрузить сервисные услуги.</p>
                </div>
            ) : null}

            {!isLoading && !isError && agreement?.client ? (
                <>
                    <button
                        type="button"
                        onClick={() => onOpenClient(agreement.client)}
                        className="w-full rounded-3xl border border-white/10 bg-slate-950/35 p-6 text-left shadow-xl shadow-black/20 backdrop-blur transition hover:border-white/20 hover:bg-slate-950/45"
                    >
                        <p className="text-xs font-semibold uppercase tracking-[0.24em] text-slate-400">
                            Активная услуга
                        </p>
                        <p className="mt-3 text-2xl font-semibold tracking-tight text-slate-50">
                            {agreement.clientName || "Не указано"}
                        </p>
                        <p className="mt-2 text-lg text-slate-400">{agreement.clientAddress || "Адрес не указан"}</p>
                        <p className="mt-6 text-base font-medium text-slate-200">{formatAgreementRange(agreement)}</p>
                    </button>

                    <button
                        type="button"
                        onClick={() => onOpenExpiredAgreements(agreement.client)}
                        className="inline-flex items-center gap-3 rounded-2xl px-2 py-1 text-lg font-semibold text-[#8B5CFF] transition hover:text-[#A27BFF]"
                    >
                        <span>Истекшие сервисные услуги</span>
                        <svg
                            xmlns="http://www.w3.org/2000/svg"
                            viewBox="0 0 24 24"
                            fill="none"
                            stroke="currentColor"
                            strokeWidth="2.2"
                            strokeLinecap="round"
                            strokeLinejoin="round"
                            className="h-5 w-5"
                            aria-hidden="true"
                        >
                            <path d="M9 6l6 6-6 6" />
                        </svg>
                    </button>
                </>
            ) : null}

            {!isLoading && !isError && !agreement?.client ? (
                <div className="rounded-3xl border border-white/10 bg-white/5 p-6">
                    <p className="text-sm text-slate-300">Для этого устройства не найдено активных сервисных услуг.</p>
                </div>
            ) : null}
        </section>
    );
}

function DeviceLatestTicketsSection({ deviceId, isError, isLoading, onOpenArchive, onOpenTicket, tickets }) {
    return (
        <section className="space-y-4">
            <h2 className="text-2xl font-bold tracking-tight text-slate-50 sm:text-3xl">Последние выезды</h2>

            {isLoading ? (
                <div className="rounded-3xl border border-white/10 bg-white/5 p-6">
                    <p className="text-sm text-slate-300">Загрузка последних выездов...</p>
                </div>
            ) : null}

            {isError ? (
                <div className="rounded-3xl border border-rose-300/30 bg-rose-500/10 p-6">
                    <p className="text-sm text-rose-100">Не удалось загрузить последние выезды.</p>
                </div>
            ) : null}

            {!isLoading && !isError && tickets.length > 0 ? (
                <div className="grid gap-3">
                    {tickets.map((ticket) => (
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
            ) : null}

            {!isLoading && !isError && tickets.length === 0 ? (
                <div className="rounded-3xl border border-white/10 bg-white/5 p-6">
                    <p className="text-sm text-slate-300">У этого устройства пока нет закрытых выездов.</p>
                </div>
            ) : null}

            <button
                type="button"
                onClick={() => onOpenArchive(deviceId)}
                className="inline-flex items-center gap-3 rounded-2xl px-2 py-1 text-lg font-semibold text-[#9B7BFF] transition hover:text-[#B49CFF]"
            >
                <span>Все выезды</span>
                <svg
                    xmlns="http://www.w3.org/2000/svg"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    strokeWidth="2.2"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    className="h-5 w-5"
                    aria-hidden="true"
                >
                    <path d="M5 12h14" />
                    <path d="m13 6 6 6-6 6" />
                </svg>
            </button>
        </section>
    );
}

function DeviceCommentsSection({
    comments,
    commentText,
    errorMessage,
    isError,
    isLoading,
    isSubmitting,
    onChangeText,
    onSubmit,
}) {
    return (
        <section className="space-y-4">
            <h2 className="text-2xl font-bold tracking-tight text-slate-50 sm:text-3xl">Комментарии</h2>

            {isLoading ? (
                <div className="rounded-3xl border border-white/10 bg-white/5 p-6">
                    <p className="text-sm text-slate-300">Загрузка комментариев...</p>
                </div>
            ) : null}

            {isError ? (
                <div className="rounded-3xl border border-rose-300/30 bg-rose-500/10 p-6">
                    <p className="text-sm text-rose-100">Не удалось загрузить комментарии.</p>
                </div>
            ) : null}

            {!isLoading && !isError && comments.length === 0 ? (
                <div className="rounded-3xl border border-white/10 bg-white/5 p-6">
                    <p className="text-sm text-slate-300">Пока нет комментариев.</p>
                </div>
            ) : null}

            {!isLoading && !isError && comments.length > 0 ? (
                <div className="grid gap-3">
                    {comments.map((comment) => (
                        <article
                            key={comment.id}
                            className="rounded-3xl border border-white/10 bg-slate-950/35 p-6 shadow-xl shadow-black/20 backdrop-blur"
                        >
                            <p className="text-lg leading-8 text-slate-100 sm:text-2xl sm:leading-10">
                                {comment.text || "—"}
                            </p>

                            <div className="mt-6 flex items-end justify-between gap-4">
                                <div className="flex items-center gap-4">
                                    <div
                                        className="h-10 w-10 rounded-full bg-slate-950 sm:h-12 sm:w-12"
                                        aria-hidden="true"
                                    />
                                    <div>
                                        <p className="text-lg font-semibold text-slate-100">
                                            {comment.authorName || "Не указано"}
                                        </p>
                                        <p className="text-sm text-slate-400 sm:text-lg">
                                            {comment.department || "Отдел не указан"}
                                        </p>
                                    </div>
                                </div>
                                <p className="shrink-0 text-sm text-slate-400 sm:text-lg">
                                    {formatCommentDate(comment.created_at)}
                                </p>
                            </div>
                        </article>
                    ))}
                </div>
            ) : null}

            <form
                onSubmit={onSubmit}
                className="flex items-end gap-3 rounded-[2rem] border border-white/10 bg-slate-950/35 p-3 shadow-xl shadow-black/20 backdrop-blur"
            >
                <textarea
                    value={commentText}
                    onChange={(event) => onChangeText(event.target.value)}
                    placeholder="Добавить комментарий"
                    rows={3}
                    className="min-h-[7rem] flex-1 resize-none rounded-[1.6rem] border border-white/10 bg-white/5 px-5 py-4 text-lg text-slate-100 outline-none transition placeholder:text-slate-400 focus:border-white/25"
                />
                <button
                    type="submit"
                    disabled={isSubmitting || !commentText.trim()}
                    className="inline-flex h-14 w-14 shrink-0 items-center justify-center rounded-full bg-[#6A3BF2] text-white transition hover:bg-[#7C52F5] disabled:cursor-not-allowed disabled:opacity-60"
                    aria-label="Отправить комментарий"
                >
                    <svg
                        xmlns="http://www.w3.org/2000/svg"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        strokeWidth="2.4"
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        className="h-6 w-6"
                        aria-hidden="true"
                    >
                        <path d="M3 11.5 20.5 4 13 21l-2.5-6.5L3 11.5Z" />
                    </svg>
                </button>
            </form>

            {errorMessage ? <p className="text-sm text-rose-200">{errorMessage}</p> : null}
        </section>
    );
}

function DeviceCreateTicketSheet({ device, isOpen, isSubmitting, onClose, onSubmitCreate, submitError }) {
    const {
        data: ticketReasons = [],
        isError: isTicketReasonsError,
        isFetching: isTicketReasonsFetching,
    } = useGetTicketReasonsQuery(undefined, {
        skip: !isOpen,
    });
    const {
        data: clientContacts = [],
        isError: isClientContactsError,
        isFetching: isClientContactsFetching,
    } = useGetClientContactsQuery(
        {
            clientId: device?.client,
            limit: 100,
        },
        {
            skip: !isOpen || !device?.client,
        },
    );
    const {
        data: departmentMembers = [],
        isError: isDepartmentMembersError,
        isFetching: isDepartmentMembersFetching,
    } = useGetDepartmentMembersQuery(undefined, {
        skip: !isOpen,
    });
    const [contactId, setContactId] = useState("");
    const [descriptionValue, setDescriptionValue] = useState("");
    const [executorId, setExecutorId] = useState("");
    const [isUrgent, setIsUrgent] = useState(false);
    const [localError, setLocalError] = useState("");
    const [assignedEndValue, setAssignedEndValue] = useState("");
    const [assignedStartValue, setAssignedStartValue] = useState("");
    const [ticketReasonId, setTicketReasonId] = useState("");
    const assignedEndInputRef = useRef(null);
    const assignedStartInputRef = useRef(null);
    const deviceTitle = device?.title?.trim() || "Устройство";
    const serialNumber = device?.serialNumber?.trim() || "Не указано";
    const clientName = device?.clientName?.trim() || "Клиент не указан";
    const clientAddress = device?.clientAddress?.trim() || "Адрес не указан";

    useEffect(() => {
        if (!isOpen) {
            setAssignedEndValue("");
            setAssignedStartValue("");
            setContactId("");
            setDescriptionValue("");
            setExecutorId("");
            setIsUrgent(false);
            setLocalError("");
            setTicketReasonId("");
        }
    }, [isOpen]);

    function handleAssignedStartChange(event) {
        setLocalError("");
        setAssignedStartValue(event.target.value);
    }

    function handleAssignedEndChange(event) {
        const nextAssignedEndValue = event.target.value;

        setLocalError("");
        setAssignedEndValue(nextAssignedEndValue);
        setAssignedStartValue((currentValue) => currentValue || nextAssignedEndValue);
    }

    function openDatePicker(inputRef) {
        const input = inputRef.current;
        if (!input) {
            return;
        }

        input.focus();
        if (typeof input.showPicker === "function") {
            input.showPicker();
        }
    }

    const trimmedDescription = descriptionValue.trim();
    const hasClientContext = Boolean(device?.client);
    const hasDateRangeError = Boolean(assignedStartValue && assignedEndValue && assignedStartValue > assignedEndValue);
    const isFormComplete = Boolean(
        device?.id &&
        device?.client &&
        ticketReasonId &&
        trimmedDescription &&
        contactId &&
        executorId &&
        assignedStartValue &&
        assignedEndValue,
    );
    const isSubmitDisabled = isSubmitting || !isFormComplete || hasDateRangeError;

    async function handleCreateTicketSubmit(event) {
        event.preventDefault();

        if (!device?.id) {
            setLocalError("Не удалось определить прибор для создания тикета.");
            return;
        }

        if (!hasClientContext) {
            setLocalError("Для создания тикета у прибора должен быть связан клиент.");
            return;
        }

        if (!isFormComplete) {
            setLocalError("Заполните все обязательные поля.");
            return;
        }

        if (hasDateRangeError) {
            setLocalError("Дата завершения не может быть раньше даты начала.");
            return;
        }

        setLocalError("");

        try {
            await onSubmitCreate({
                assigned_end: assignedEndValue,
                assigned_start: assignedStartValue,
                client: device.client,
                contact_person: contactId,
                description: trimmedDescription,
                device: device.id,
                executor: executorId,
                reason: ticketReasonId,
                urgent: isUrgent,
            });
        } catch {
            return;
        }
    }

    return (
        <SlideOverSheet
            isOpen={isOpen}
            onClose={onClose}
            closeLabel="Закрыть создание тикета"
            eyebrow="Новый тикет"
            panelClassName="lg:w-[42rem] xl:w-[46rem]"
            title="Создание тикета на прибор"
        >
            <div className="mt-8 space-y-6">
                <div className="rounded-2xl border border-white/10 bg-white/5 p-5">
                    <p className="text-sm text-slate-400">Прибор</p>
                    <p className="mt-2 text-lg font-semibold text-slate-100">{deviceTitle}</p>
                    <p className="mt-2 text-sm text-slate-300">С/Н: {serialNumber}</p>
                </div>

                <div className="rounded-2xl border border-white/10 bg-white/5 p-5">
                    <p className="text-sm text-slate-400">Клиент</p>
                    <p className="mt-2 text-lg font-semibold text-slate-100">{clientName}</p>
                    <p className="mt-2 text-sm text-slate-300">{clientAddress}</p>
                </div>

                <form className="space-y-8" onSubmit={handleCreateTicketSubmit}>
                    <div className="space-y-3">
                        <label htmlFor="device-ticket-reason" className="block text-3xl font-semibold text-slate-100">
                            Причина
                        </label>

                        <SelectField
                            id="device-ticket-reason"
                            name="reason"
                            value={ticketReasonId}
                            onChange={(event) => {
                                setLocalError("");
                                setTicketReasonId(event.target.value);
                            }}
                            disabled={isSubmitting || isTicketReasonsFetching || isTicketReasonsError}
                            className="text-xl"
                        >
                            <option value="">
                                {isTicketReasonsFetching ? "Загружаем причины..." : "Выберите причину"}
                            </option>
                            {ticketReasons.map((reason) => (
                                <option key={reason.id} value={reason.id}>
                                    {reason.title}
                                </option>
                            ))}
                        </SelectField>

                        {isTicketReasonsError ? (
                            <p className="text-sm text-rose-200">Не удалось загрузить причины тикетов.</p>
                        ) : null}
                    </div>

                    <div className="space-y-3">
                        <label
                            htmlFor="device-ticket-description"
                            className="block text-3xl font-semibold text-slate-100"
                        >
                            Описание
                        </label>

                        <textarea
                            id="device-ticket-description"
                            name="description"
                            value={descriptionValue}
                            onChange={(event) => {
                                setLocalError("");
                                setDescriptionValue(event.target.value);
                            }}
                            disabled={isSubmitting}
                            placeholder="Опишите задачу"
                            className="min-h-44 w-full resize-y rounded-2xl border border-slate-400/35 bg-transparent px-4 py-4 text-xl text-slate-100 outline-none transition placeholder:text-slate-400 focus:border-[#9fb5d6] focus:ring-2 focus:ring-[#9fb5d6]/30"
                        />
                    </div>

                    <div className="space-y-3">
                        <label htmlFor="device-ticket-contact" className="block text-3xl font-semibold text-slate-100">
                            Контакт
                        </label>

                        <SelectField
                            id="device-ticket-contact"
                            name="contact"
                            value={contactId}
                            onChange={(event) => {
                                setLocalError("");
                                setContactId(event.target.value);
                            }}
                            disabled={
                                isSubmitting || !device?.client || isClientContactsFetching || isClientContactsError
                            }
                            className="text-xl"
                        >
                            <option value="">
                                {!device?.client
                                    ? "У клиента нет идентификатора"
                                    : isClientContactsFetching
                                      ? "Загружаем контакты..."
                                      : clientContacts.length > 0
                                        ? "Выберите контакт"
                                        : "Контакты не найдены"}
                            </option>
                            {clientContacts.map((contact) => (
                                <option key={contact.id} value={contact.id}>
                                    {[contact.name, contact.position].filter(Boolean).join(" • ") || contact.id}
                                </option>
                            ))}
                        </SelectField>

                        {isClientContactsError ? (
                            <p className="text-sm text-rose-200">Не удалось загрузить контакты клиента.</p>
                        ) : null}
                    </div>

                    <div className="space-y-3">
                        <label htmlFor="device-ticket-executor" className="block text-3xl font-semibold text-slate-100">
                            Исполнитель
                        </label>

                        <SelectField
                            id="device-ticket-executor"
                            name="executor"
                            value={executorId}
                            onChange={(event) => {
                                setLocalError("");
                                setExecutorId(event.target.value);
                            }}
                            disabled={isSubmitting || isDepartmentMembersFetching || isDepartmentMembersError}
                            className="text-xl"
                        >
                            <option value="">
                                {isDepartmentMembersFetching
                                    ? "Загружаем исполнителей..."
                                    : departmentMembers.length > 0
                                      ? "Выберите исполнителя"
                                      : "Сотрудники отдела не найдены"}
                            </option>
                            {departmentMembers.map((member) => (
                                <option key={member.id} value={member.id}>
                                    {member.name?.trim() || member.username}
                                </option>
                            ))}
                        </SelectField>

                        {isDepartmentMembersError ? (
                            <p className="text-sm text-rose-200">Не удалось загрузить сотрудников отдела.</p>
                        ) : null}
                    </div>

                    <div className="grid gap-6 sm:grid-cols-2">
                        <div className="space-y-3">
                            <label
                                htmlFor="device-ticket-assigned-start"
                                className="block text-2xl font-semibold text-slate-100"
                            >
                                Начать до
                            </label>

                            <input
                                ref={assignedStartInputRef}
                                id="device-ticket-assigned-start"
                                name="assigned_start"
                                type="date"
                                value={assignedStartValue}
                                disabled={isSubmitting}
                                onClick={() => openDatePicker(assignedStartInputRef)}
                                onChange={handleAssignedStartChange}
                                className="min-h-16 w-full cursor-pointer rounded-2xl border border-slate-400/35 bg-slate-950 px-4 py-3 text-lg text-slate-100 outline-none transition focus:border-[#9fb5d6] focus:ring-2 focus:ring-[#9fb5d6]/30"
                            />
                        </div>

                        <div className="space-y-3">
                            <label
                                htmlFor="device-ticket-assigned-end"
                                className="block text-2xl font-semibold text-slate-100"
                            >
                                Закончить до
                            </label>

                            <input
                                ref={assignedEndInputRef}
                                id="device-ticket-assigned-end"
                                name="assigned_end"
                                type="date"
                                value={assignedEndValue}
                                disabled={isSubmitting}
                                onClick={() => openDatePicker(assignedEndInputRef)}
                                onChange={handleAssignedEndChange}
                                className="min-h-16 w-full cursor-pointer rounded-2xl border border-slate-400/35 bg-slate-950 px-4 py-3 text-lg text-slate-100 outline-none transition focus:border-[#9fb5d6] focus:ring-2 focus:ring-[#9fb5d6]/30"
                            />
                        </div>
                    </div>

                    <label className="inline-flex cursor-pointer items-center gap-3 select-none">
                        <input
                            type="checkbox"
                            name="urgent"
                            checked={isUrgent}
                            disabled={isSubmitting}
                            onChange={(event) => setIsUrgent(event.target.checked)}
                            className="h-5 w-5 rounded border border-slate-400/60 bg-transparent text-[#6A3BF2] accent-[#6A3BF2]"
                        />
                        <span className="text-lg text-slate-100">Срочно</span>
                    </label>

                    <div className="sticky -bottom-6 -mx-6 border-t border-white/15 bg-slate-950/95 px-6 py-5 backdrop-blur">
                        <button
                            type="submit"
                            disabled={isSubmitDisabled}
                            className="flex min-h-14 w-full items-center justify-center rounded-2xl bg-emerald-500 px-5 text-base font-semibold text-white transition hover:bg-emerald-400 disabled:cursor-not-allowed disabled:bg-emerald-600/70 sm:text-lg"
                        >
                            <span className="mr-2 inline-flex h-6 w-6 items-center justify-center rounded-full bg-white/20">
                                <svg
                                    xmlns="http://www.w3.org/2000/svg"
                                    viewBox="0 0 24 24"
                                    fill="none"
                                    stroke="currentColor"
                                    strokeWidth="2.5"
                                    strokeLinecap="round"
                                    strokeLinejoin="round"
                                    className="h-4 w-4"
                                    aria-hidden="true"
                                >
                                    <path d="M20 6 9 17l-5-5" />
                                </svg>
                            </span>
                            <span>{isSubmitting ? "Сохраняем..." : "Создать тикет"}</span>
                        </button>

                        {localError ? <p className="mt-2 text-center text-xs text-rose-200">{localError}</p> : null}
                        {submitError ? <p className="mt-2 text-center text-xs text-rose-200">{submitError}</p> : null}
                        {!submitError && !localError && hasDateRangeError ? (
                            <p className="mt-2 text-center text-xs text-rose-200">
                                Дата завершения не может быть раньше даты начала.
                            </p>
                        ) : null}
                    </div>
                </form>
            </div>
        </SlideOverSheet>
    );
}

export function DevicePage() {
    const navigate = useNavigate();
    const { deviceId } = useParams();
    const { session } = useAuth();
    const [commentText, setCommentText] = useState("");
    const [commentError, setCommentError] = useState("");
    const [createTicketError, setCreateTicketError] = useState("");
    const [isCreateTicketSheetOpen, setIsCreateTicketSheetOpen] = useState(false);
    const {
        data: device,
        isError,
        isFetching,
        isLoading,
    } = useGetDeviceByIdQuery(deviceId, {
        skip: !deviceId,
    });
    const {
        data: agreements = [],
        isError: isAgreementsError,
        isFetching: isAgreementsFetching,
        isLoading: isAgreementsLoading,
    } = useGetDeviceAgreementsQuery(
        {
            active: true,
            deviceId,
        },
        {
            skip: !deviceId,
        },
    );
    const {
        data: tickets = [],
        isError: isTicketsError,
        isFetching: isTicketsFetching,
        isLoading: isTicketsLoading,
    } = useGetDeviceTicketsQuery(
        {
            deviceId,
            limit: 2,
            status: "closed",
        },
        {
            skip: !deviceId,
        },
    );
    const {
        data: comments = [],
        error: commentsError,
        isError: isCommentsError,
        isFetching: isCommentsFetching,
        isLoading: isCommentsLoading,
    } = useGetCommentsQuery(deviceId, {
        skip: !deviceId,
    });
    const [addComment, { isLoading: isAddingComment }] = useAddCommentMutation();
    const [createTicket, { isLoading: isCreatingTicket }] = useCreateTicketMutation();

    const pageTitle = device?.title?.trim() || "Устройство";
    const serialNumber = device?.serialNumber?.trim() || "";
    const propertyEntries = buildPropertyEntries(device?.properties);
    const activeAgreement = agreements[0] || null;
    const canCreateTicket = session?.role === "admin" || session?.role === "coordinator";
    const hasCreateTicketWidget = canCreateTicket && !isLoading && !isFetching && !isError && Boolean(device);
    const hasMissingCommentReference = isMissingCommentReferenceError(commentsError);
    const isCommentsSectionError = isCommentsError && !hasMissingCommentReference;

    async function handleSubmitComment(event) {
        event.preventDefault();

        const nextComment = commentText.trim();
        if (!deviceId || !nextComment) {
            return;
        }

        setCommentError("");

        try {
            await addComment({
                referenceId: deviceId,
                text: nextComment,
            }).unwrap();
            setCommentText("");
        } catch (error) {
            if (typeof error?.data === "string") {
                setCommentError(error.data);
                return;
            }

            if (typeof error?.error === "string") {
                setCommentError(error.error);
                return;
            }

            setCommentError("Не удалось добавить комментарий.");
        }
    }

    function resolveCreateTicketErrorMessage(error) {
        if (!error) {
            return "Не удалось создать тикет.";
        }

        if (typeof error.data === "string") {
            return error.data;
        }

        if (typeof error.error === "string") {
            return error.error;
        }

        return "Не удалось создать тикет.";
    }

    async function handleCreateTicket(payload) {
        setCreateTicketError("");

        try {
            const createdTicket = await createTicket(payload).unwrap();
            setIsCreateTicketSheetOpen(false);
            navigate(routePaths.ticketById(createdTicket.id));
            return createdTicket;
        } catch (error) {
            const nextErrorMessage = resolveCreateTicketErrorMessage(error);
            setCreateTicketError(nextErrorMessage);
            throw error;
        }
    }

    return (
        <PageShell>
            <section
                className={`w-full space-y-6 transition ${hasCreateTicketWidget ? "pb-28" : ""} ${
                    isCreateTicketSheetOpen ? "brightness-75" : ""
                }`}
            >
                <DeviceHeader title={pageTitle} serialNumber={serialNumber} onBack={() => navigate(-1)} />

                {isLoading || isFetching ? (
                    <div className="rounded-3xl border border-white/10 bg-white/5 p-6">
                        <p className="text-sm text-slate-300">Загрузка устройства...</p>
                    </div>
                ) : null}

                {isError ? (
                    <div className="rounded-3xl border border-rose-300/30 bg-rose-500/10 p-6">
                        <p className="text-sm text-rose-100">Не удалось загрузить устройство.</p>
                    </div>
                ) : null}

                {!isLoading && !isFetching && !isError && device ? (
                    <>
                        <DeviceServiceSection
                            agreement={activeAgreement}
                            isError={isAgreementsError}
                            isLoading={isAgreementsLoading || isAgreementsFetching}
                            onOpenClient={(clientIdValue) => navigate(routePaths.clientById(clientIdValue))}
                            onOpenExpiredAgreements={(clientIdValue) =>
                                navigate(routePaths.clientAgreementsById(clientIdValue))
                            }
                        />
                        <DeviceOverviewSection device={device} propertyEntries={propertyEntries} />
                        <DeviceLatestTicketsSection
                            deviceId={deviceId}
                            tickets={tickets}
                            isError={isTicketsError}
                            isLoading={isTicketsLoading || isTicketsFetching}
                            onOpenArchive={(targetDeviceId) => navigate(routePaths.deviceArchiveById(targetDeviceId))}
                            onOpenTicket={(ticketId) => navigate(routePaths.ticketById(ticketId))}
                        />
                        <DeviceCommentsSection
                            comments={comments}
                            commentText={commentText}
                            errorMessage={commentError}
                            isError={isCommentsSectionError}
                            isLoading={isCommentsLoading || isCommentsFetching}
                            isSubmitting={isAddingComment}
                            onChangeText={setCommentText}
                            onSubmit={handleSubmitComment}
                        />
                    </>
                ) : null}
            </section>

            {hasCreateTicketWidget ? (
                <BottomPageAction
                    onClick={() => {
                        setCreateTicketError("");
                        setIsCreateTicketSheetOpen(true);
                    }}
                >
                    <span>Создать тикет на прибор</span>
                </BottomPageAction>
            ) : null}

            <DeviceCreateTicketSheet
                device={device}
                isOpen={isCreateTicketSheetOpen}
                isSubmitting={isCreatingTicket}
                onClose={() => {
                    setCreateTicketError("");
                    setIsCreateTicketSheetOpen(false);
                }}
                onSubmitCreate={handleCreateTicket}
                submitError={createTicketError}
            />
        </PageShell>
    );
}
