import { useEffect, useRef, useState } from "react";
import {
  useGetClientContactsQuery,
  useGetDepartmentMembersQuery,
  useGetTicketReasonsQuery,
} from "../../../../shared/api/tickets-api";
import { SelectField } from "../../../../shared/ui/select-field";
import { SlideOverSheet } from "../../../../shared/ui/slide-over-sheet";

function formatTicketDateForInput(value) {
  if (!value) {
    return "";
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "";
  }

  return date.toISOString().slice(0, 10);
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

function buildContactLabel(ticket) {
  return [ticket?.contactName, ticket?.contactPosition].filter(Boolean).join(" • ") || "Контакт не указан";
}

function buildReasonLabel(ticket) {
  return ticket?.resolvedReason?.trim() || "Причина не указана";
}

export function TicketAssignmentSheet({
  isOpen,
  isSubmitting,
  onClose,
  onSubmitAssign,
  submitError,
  ticket,
}) {
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
      clientId: ticket?.client,
      limit: 100,
    },
    {
      skip: !isOpen || !ticket?.client,
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
  const deviceTitle = ticket?.deviceName?.trim() || "Устройство";
  const serialNumber = ticket?.deviceSerialNumber?.trim() || "Не указано";
  const clientName = ticket?.clientName?.trim() || "Клиент не указан";
  const clientAddress = ticket?.clientAddress?.trim() || "Адрес не указан";
  const trimmedDescription = descriptionValue.trim();
  const hasDateRangeError = Boolean(
    assignedStartValue &&
      assignedEndValue &&
      assignedStartValue > assignedEndValue,
  );
  const isFormComplete = Boolean(
    ticket?.id &&
      ticketReasonId &&
      trimmedDescription &&
      contactId &&
      executorId &&
      assignedStartValue &&
      assignedEndValue,
  );
  const isSubmitDisabled = isSubmitting || !isFormComplete || hasDateRangeError;
  const selectedReason =
    ticketReasons.find((reason) => reason.id === ticketReasonId) || null;
  const selectedContact =
    clientContacts.find((contact) => contact.id === contactId) || null;
  const activeDepartmentMembers = departmentMembers.filter(
    (member) => !member.isDisabled,
  );

  useEffect(() => {
    if (!isOpen) {
      setLocalError("");
      return;
    }

    setAssignedEndValue(formatTicketDateForInput(ticket?.assigned_end || ticket?.assigned_start));
    setAssignedStartValue(formatTicketDateForInput(ticket?.assigned_start || ticket?.assigned_end));
    setContactId(ticket?.contactPerson || "");
    setDescriptionValue(ticket?.description || "");
    setExecutorId(ticket?.executor || "");
    setIsUrgent(Boolean(ticket?.urgent));
    setLocalError("");
    setTicketReasonId(ticket?.reason || "");
  }, [isOpen, ticket]);

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

  async function handleAssignSubmit(event) {
    event.preventDefault();

    if (!ticket?.id) {
      setLocalError("Не удалось определить тикет для назначения.");
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
      await onSubmitAssign({
        assigned_end: assignedEndValue,
        assigned_start: assignedStartValue,
        contact_person: contactId,
        description: trimmedDescription,
        executor: executorId,
        reason: ticketReasonId,
        status: "assigned",
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
      closeLabel="Закрыть назначение инженера"
      eyebrow="Тикет"
      panelClassName="lg:w-[42rem] xl:w-[46rem]"
      title="Назначить инженера"
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

        <form className="space-y-8" onSubmit={handleAssignSubmit}>
          <div className="space-y-3">
            <label htmlFor="ticket-assignment-reason" className="block text-3xl font-semibold text-slate-100">
              Причина
            </label>

            <SelectField
              id="ticket-assignment-reason"
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
              {ticketReasonId && !selectedReason ? (
                <option value={ticketReasonId}>{buildReasonLabel(ticket)}</option>
              ) : null}
              {ticketReasons.map((reason) => (
                <option key={reason.id} value={reason.id}>
                  {reason.title}
                </option>
              ))}
            </SelectField>

            {isTicketReasonsFetching ? (
              <p className="text-sm text-slate-400">Загружаем причины...</p>
            ) : null}
            {isTicketReasonsError ? (
              <p className="text-sm text-rose-200">Не удалось загрузить причины тикетов.</p>
            ) : null}
          </div>

          <div className="space-y-3">
            <label
              htmlFor="ticket-assignment-description"
              className="block text-3xl font-semibold text-slate-100"
            >
              Описание
            </label>

            <textarea
              id="ticket-assignment-description"
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
            <label htmlFor="ticket-assignment-contact" className="block text-3xl font-semibold text-slate-100">
              Контакт
            </label>

            <SelectField
              id="ticket-assignment-contact"
              name="contact"
              value={contactId}
              onChange={(event) => {
                setLocalError("");
                setContactId(event.target.value);
              }}
              disabled={isSubmitting || !ticket?.client || isClientContactsFetching || isClientContactsError}
              className="text-xl"
            >
              <option value="">
                {!ticket?.client
                  ? "У клиента нет идентификатора"
                  : isClientContactsFetching
                    ? "Загружаем контакты..."
                    : clientContacts.length > 0
                      ? "Выберите контакт"
                      : "Контакты не найдены"}
              </option>
              {contactId && !selectedContact ? (
                <option value={contactId}>{buildContactLabel(ticket)}</option>
              ) : null}
              {clientContacts.map((contact) => (
                <option key={contact.id} value={contact.id}>
                  {[contact.name, contact.position].filter(Boolean).join(" • ") || contact.id}
                </option>
              ))}
            </SelectField>

            {isClientContactsFetching ? (
              <p className="text-sm text-slate-400">Загружаем контакты...</p>
            ) : null}
            {isClientContactsError ? (
              <p className="text-sm text-rose-200">Не удалось загрузить контакты клиента.</p>
            ) : null}
          </div>

          <div className="space-y-3">
            <label htmlFor="ticket-assignment-executor" className="block text-3xl font-semibold text-slate-100">
              Исполнитель
            </label>

            <SelectField
              id="ticket-assignment-executor"
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
                  : activeDepartmentMembers.length > 0
                    ? "Выберите исполнителя"
                    : "Доступные сотрудники не найдены"}
              </option>
              {departmentMembers.map((member) => (
                <option key={member.id} value={member.id} disabled={member.isDisabled}>
                  {member.isDisabled
                    ? `${member.name?.trim() || member.username} • отключен`
                    : member.name?.trim() || member.username}
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
                htmlFor="ticket-assignment-assigned-start"
                className="block text-2xl font-semibold text-slate-100"
              >
                Начать до
              </label>

              <input
                ref={assignedStartInputRef}
                id="ticket-assignment-assigned-start"
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
                htmlFor="ticket-assignment-assigned-end"
                className="block text-2xl font-semibold text-slate-100"
              >
                Закончить до
              </label>

              <input
                ref={assignedEndInputRef}
                id="ticket-assignment-assigned-end"
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
              <span>{isSubmitting ? "Сохраняем..." : "Назначить инженера"}</span>
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
