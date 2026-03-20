import { useDeferredValue, useRef, useState } from "react";
import { useNavigate, useSearchParams } from "react-router";
import {
  useGetEditorAccountsQuery,
  useGetEditorClientsQuery,
  useGetEditorDevicesQuery,
  useGetEditorTicketByIdQuery,
  useGetEditorTicketsQuery,
  useGetEditorTicketStatusesQuery,
  useGetEditorTicketTypesQuery,
  usePatchEditorTicketMutation,
} from "../../../shared/api/editor-api";
import {
  useGetClientContactsQuery,
  useGetDepartmentsQuery,
  useGetTicketReasonsQuery,
} from "../../../shared/api/tickets-api";
import { routePaths } from "../../../shared/config/routes";
import { AsyncSearchSelect } from "../../../shared/ui/async-search-select";
import { PageShell } from "../../../shared/ui/page-shell";
import { SelectField } from "../../../shared/ui/select-field";
import { StatusMessage } from "../../../shared/ui/status-message";
import {
  BackButton,
  editorFieldClassName,
  EditorFormField,
  EditorListError,
  EditorListHeader,
  EditorNoticeCard,
  EditorPageHeader,
  EditorPane,
  EditorRecordHeader,
  EditorSearchField,
  editorSelectClassName,
  editorTextareaClassName,
  EditorSidebar,
  EditorWorkspace,
  SummaryCard,
  useSyncedSidebarHeight,
} from "./editor-shared";
import { useEditorSearchParamSelection, useLoadedEditorRecord, useUnsavedChangesWarning } from "./editor-hooks";

function EditorCheckboxField({ checked, label, name, onChange }) {
  return (
    <label className="inline-flex cursor-pointer items-center gap-3 rounded-2xl border border-white/10 bg-slate-950/35 px-4 py-3 text-sm text-slate-100">
      <input
        type="checkbox"
        name={name}
        checked={checked}
        onChange={onChange}
        className="h-5 w-5 rounded border border-slate-400/60 bg-transparent text-[#6A3BF2] accent-[#6A3BF2]"
      />
      <span>{label}</span>
    </label>
  );
}

function formatTicketDateTime(value) {
  if (!value) {
    return "Не указано";
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "Некорректная дата";
  }

  const day = String(date.getDate()).padStart(2, "0");
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const year = date.getFullYear();
  const hours = String(date.getHours()).padStart(2, "0");
  const minutes = String(date.getMinutes()).padStart(2, "0");
  return `${day}.${month}.${year} ${hours}:${minutes}`;
}

function toDateTimeLocalValue(value) {
  if (!value) {
    return "";
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "";
  }

  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  const hours = String(date.getHours()).padStart(2, "0");
  const minutes = String(date.getMinutes()).padStart(2, "0");
  return `${year}-${month}-${day}T${hours}:${minutes}`;
}

function getDeviceOptionLabel(device) {
  const title = device.title?.trim() || "Без устройства";
  const serialNumber = device.serialNumber?.trim();
  return serialNumber ? `${title} • ${serialNumber}` : title;
}

function getAccountOptionLabel(account) {
  return account.title?.trim() || account.username?.trim() || "Без имени";
}

function getAccountOptionDescription(account) {
  return [account.username?.trim(), account.departmentTitle?.trim()].filter(Boolean).join(" • ");
}

function createSimpleSelection(id, label) {
  if (!id) {
    return null;
  }

  return {
    id,
    label: label?.trim() || "Без названия",
  };
}

function createDeviceSelection(id, title, serialNumber) {
  if (!id) {
    return null;
  }

  return {
    id,
    label: getDeviceOptionLabel({
      serialNumber,
      title,
    }),
  };
}

function createAccountSelection(id, title, username, departmentTitle) {
  if (!id) {
    return null;
  }

  return {
    id,
    label: getAccountOptionLabel({
      departmentTitle,
      title,
      username,
    }),
  };
}

function createContactSelection(id, name) {
  if (!id) {
    return null;
  }

  return {
    id,
    label: name?.trim() || "Контакт без имени",
  };
}

function getTicketFormState(ticket) {
  return {
    assignedEnd: toDateTimeLocalValue(ticket?.assignedEnd),
    assignedStart: toDateTimeLocalValue(ticket?.assignedStart),
    client: ticket?.client || "",
    closedAt: toDateTimeLocalValue(ticket?.closedAt),
    contactPerson: ticket?.contactPerson || "",
    department: ticket?.department || "",
    description: ticket?.description || "",
    device: ticket?.device || "",
    doubleSigned: Boolean(ticket?.doubleSigned),
    executor: ticket?.executor || "",
    reason: ticket?.reason || "",
    result: ticket?.result || "",
    status: ticket?.status || "",
    ticketType: ticket?.ticketType || "",
    urgent: Boolean(ticket?.urgent),
    workfinishedAt: toDateTimeLocalValue(ticket?.workfinishedAt),
    workstartedAt: toDateTimeLocalValue(ticket?.workstartedAt),
  };
}

function areTicketFormStatesEqual(left, right) {
  return Object.keys(left).every((key) => left[key] === right[key]);
}

function TicketListItem({ isActive, onClick, ticket }) {
  const title = ticket.number ? `Тикет #${ticket.number}` : "Тикет без номера";
  const meta = [
    ticket.resolvedReason?.trim(),
    ticket.clientName?.trim(),
    getDeviceOptionLabel({
      serialNumber: ticket.deviceSerialNumber,
      title: ticket.deviceTitle,
    }),
  ]
    .filter(Boolean)
    .join(" • ");

  return (
    <button
      type="button"
      onClick={onClick}
      className={`w-full rounded-3xl border p-4 text-left transition ${
        isActive
          ? "border-cyan-200/35 bg-cyan-400/10 shadow-lg shadow-cyan-500/10"
          : "border-white/10 bg-slate-950/25 hover:border-white/20 hover:bg-white/10"
      }`}
    >
      <div className="flex items-start justify-between gap-4">
        <div className="space-y-2">
          <p className="text-base font-semibold text-white">{title}</p>
          <p className="text-sm text-slate-400">{meta || "Причина, клиент и устройство пока не указаны."}</p>
          <p className="text-xs text-slate-500">
            {ticket.createdAt ? `Создан: ${formatTicketDateTime(ticket.createdAt)}` : "Дата создания не указана"}
          </p>
        </div>
        <div className="shrink-0 text-right text-xs text-slate-400">
          <p>{ticket.statusTitle?.trim() || ticket.status || "Без статуса"}</p>
          <p className="mt-1">{ticket.urgent ? "Срочный" : "Плановый"}</p>
        </div>
      </div>
    </button>
  );
}

export function EditorTicketsPage() {
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const editorPaneRef = useRef(null);
  const selectedTicketId = searchParams.get("ticketId") || "";
  const [searchValue, setSearchValue] = useState("");
  const deferredSearchValue = useDeferredValue(searchValue.trim());
  const [formState, setFormState] = useState(getTicketFormState(null));
  const [loadedTicketId, setLoadedTicketId] = useState("");
  const [clientSearchValue, setClientSearchValue] = useState("");
  const [deviceSearchValue, setDeviceSearchValue] = useState("");
  const [executorSearchValue, setExecutorSearchValue] = useState("");
  const deferredClientSearchValue = useDeferredValue(clientSearchValue.trim());
  const deferredDeviceSearchValue = useDeferredValue(deviceSearchValue.trim());
  const deferredExecutorSearchValue = useDeferredValue(executorSearchValue.trim());
  const [selectedClientOption, setSelectedClientOption] = useState(null);
  const [selectedDeviceOption, setSelectedDeviceOption] = useState(null);
  const [selectedContactOption, setSelectedContactOption] = useState(null);
  const [selectedExecutorOption, setSelectedExecutorOption] = useState(null);
  const [feedback, setFeedback] = useState({
    message: "",
    tone: "idle",
  });
  const [patchEditorTicket, { isLoading: isSaving }] = usePatchEditorTicketMutation();
  const {
    data: tickets = [],
    error: ticketsError,
    isFetching: isTicketsFetching,
    isLoading: isTicketsLoading,
  } = useGetEditorTicketsQuery({
    limit: 50,
    q: deferredSearchValue,
  });
  const {
    data: selectedTicket,
    error: selectedTicketError,
    isFetching: isTicketFetching,
    isLoading: isTicketLoading,
  } = useGetEditorTicketByIdQuery(selectedTicketId, {
    skip: !selectedTicketId,
  });
  const { data: ticketStatuses = [], error: ticketStatusesError } = useGetEditorTicketStatusesQuery(undefined, {
    skip: !selectedTicketId,
  });
  const { data: ticketTypes = [], error: ticketTypesError } = useGetEditorTicketTypesQuery(undefined, {
    skip: !selectedTicketId,
  });
  const { data: ticketReasons = [], error: ticketReasonsError } = useGetTicketReasonsQuery(undefined, {
    skip: !selectedTicketId,
  });
  const { data: departments = [], error: departmentsError } = useGetDepartmentsQuery(undefined, {
    skip: !selectedTicketId,
  });
  const { data: clientOptions = [], error: clientOptionsError } = useGetEditorClientsQuery(
    {
      limit: 20,
      q: deferredClientSearchValue,
    },
    {
      skip: !selectedTicketId,
    },
  );
  const { data: deviceOptions = [], error: deviceOptionsError } = useGetEditorDevicesQuery(
    {
      limit: 20,
      q: deferredDeviceSearchValue,
    },
    {
      skip: !selectedTicketId,
    },
  );
  const { data: executorOptions = [], error: executorOptionsError } = useGetEditorAccountsQuery(
    {
      limit: 20,
      q: deferredExecutorSearchValue,
    },
    {
      skip: !selectedTicketId,
    },
  );
  const { data: contactOptions = [], error: contactOptionsError } = useGetClientContactsQuery(
    {
      clientId: formState.client,
      limit: 100,
    },
    {
      skip: !selectedTicketId || !formState.client,
    },
  );
  const initialFormState = getTicketFormState(selectedTicket);
  const isDirty = Boolean(selectedTicketId) && !areTicketFormStatesEqual(formState, initialFormState);
  const selectedStatusTitle =
    ticketStatuses.find((status) => status.id === formState.status)?.title ||
    (formState.status === selectedTicket?.status ? selectedTicket?.statusTitle?.trim() : "") ||
    formState.status ||
    "Не указан";
  const selectedReasonTitle =
    ticketReasons.find((reason) => reason.id === formState.reason)?.title ||
    (formState.reason === selectedTicket?.reason ? selectedTicket?.resolvedReason?.trim() : "") ||
    "Не указана";
  const selectedDepartmentTitle =
    departments.find((department) => department.id === formState.department)?.title ||
    (formState.department === selectedTicket?.department ? selectedTicket?.departmentTitle?.trim() : "") ||
    "Не указан";
  const selectedClientTitle =
    selectedClientOption?.label ||
    (formState.client === selectedTicket?.client ? selectedTicket?.clientName?.trim() : "") ||
    "Не указан";
  const selectedDeviceTitle =
    selectedDeviceOption?.label ||
    (formState.device === selectedTicket?.device
      ? getDeviceOptionLabel({
          serialNumber: selectedTicket?.deviceSerialNumber,
          title: selectedTicket?.deviceTitle,
        })
      : "") ||
    "Не выбрано";
  const selectedExecutorTitle =
    selectedExecutorOption?.label ||
    (formState.executor === selectedTicket?.executor ? selectedTicket?.executorName?.trim() : "") ||
    "Не назначен";
  const selectedContactTitle =
    selectedContactOption?.label ||
    (formState.contactPerson === selectedTicket?.contactPerson && formState.client === selectedTicket?.client
      ? selectedTicket?.contactName?.trim()
      : "") ||
    "Не выбран";
  const sidebarHeight = useSyncedSidebarHeight(editorPaneRef);
  const handleSelectTicket = useEditorSearchParamSelection({
    isDirty,
    items: tickets,
    paramKey: "ticketId",
    selectedId: selectedTicketId,
    setSearchParams,
  });

  function syncTicketSelections(ticket) {
    setSelectedClientOption(createSimpleSelection(ticket?.client, ticket?.clientName));
    setSelectedDeviceOption(createDeviceSelection(ticket?.device, ticket?.deviceTitle, ticket?.deviceSerialNumber));
    setSelectedContactOption(createContactSelection(ticket?.contactPerson, ticket?.contactName));
    setSelectedExecutorOption(createAccountSelection(ticket?.executor, ticket?.executorName, "", ticket?.departmentTitle));
  }

  useLoadedEditorRecord({
    loadedRecordId: loadedTicketId,
    onRecordLoad: (ticket) => {
      syncTicketSelections(ticket);
      setFeedback({
        message: "",
        tone: "idle",
      });
    },
    record: selectedTicket,
    setFormState,
    setLoadedRecordId: setLoadedTicketId,
    toFormState: getTicketFormState,
  });

  useUnsavedChangesWarning(isDirty);

  function handleBack() {
    navigate(routePaths.editor);
  }

  function handleFormChange(event) {
    const { name, value } = event.target;

    setFormState((currentState) => ({
      ...currentState,
      [name]: value,
    }));
  }

  function handleCheckboxChange(event) {
    const { checked, name } = event.target;

    setFormState((currentState) => ({
      ...currentState,
      [name]: checked,
    }));
  }

  async function handleSave() {
    if (!selectedTicketId) {
      return;
    }

    try {
      const updatedTicket = await patchEditorTicket({
        patch: formState,
        ticketId: selectedTicketId,
      }).unwrap();

      setFormState(getTicketFormState(updatedTicket));
      setLoadedTicketId(updatedTicket.id);
      syncTicketSelections(updatedTicket);
      setFeedback({
        message: "Изменения сохранены.",
        tone: "success",
      });
    } catch (error) {
      setFeedback({
        message: typeof error?.data === "string" ? error.data : "Не удалось сохранить изменения.",
        tone: "error",
      });
    }
  }

  return (
    <PageShell>
      <section className="w-full space-y-6">
        <EditorPageHeader
          title="Тикеты"
          description="Здесь можно редактировать карточку тикета: статус, причину, сроки, исполнителя, клиента, устройство и итог работы."
          action={<BackButton onClick={handleBack} />}
        />

        <EditorWorkspace
          sidebar={
            <EditorSidebar
              height={sidebarHeight}
              footer={isTicketsFetching ? "Обновляем список..." : `Показано ${tickets.length} записей.`}
            >
              <div className="space-y-4">
                <EditorListHeader title="Карточки тикетов" />
                <EditorSearchField
                  value={searchValue}
                  onChange={(event) => setSearchValue(event.target.value)}
                  placeholder="Номер, статус, причина, клиент, устройство"
                />
                <EditorListError error={ticketsError} fallbackMessage="Не удалось загрузить список тикетов." />
              </div>

              <div className="min-h-0 flex-1 space-y-3 overflow-y-auto pr-1">
                {isTicketsLoading ? <EditorNoticeCard message="Загружаем тикеты..." /> : null}
                {!isTicketsLoading && tickets.length === 0 ? (
                  <EditorNoticeCard message="По текущему запросу ничего не найдено." />
                ) : null}

                {tickets.map((ticket) => (
                  <TicketListItem
                    key={ticket.id}
                    ticket={ticket}
                    isActive={ticket.id === selectedTicketId}
                    onClick={() => handleSelectTicket(ticket.id)}
                  />
                ))}
              </div>
            </EditorSidebar>
          }
        >
          <EditorPane editorPaneRef={editorPaneRef}>
            {!selectedTicketId ? (
              <EditorNoticeCard dashed message="Выберите тикет слева, чтобы открыть карточку редактора." />
            ) : null}

            {selectedTicketId && isTicketLoading ? <EditorNoticeCard message="Загружаем карточку тикета..." /> : null}

            {selectedTicketId && selectedTicketError ? (
              <EditorListError error={selectedTicketError} fallbackMessage="Не удалось загрузить карточку тикета." />
            ) : null}

            {selectedTicket ? (
              <>
                <EditorRecordHeader
                  id={selectedTicket.id}
                  isDirty={isDirty}
                  isSaving={isSaving}
                  onSave={handleSave}
                  title={selectedTicket.number ? `Тикет #${selectedTicket.number}` : "Тикет без номера"}
                  titleLabel="Карточка тикета"
                />

                {feedback.message ? <StatusMessage feedback={feedback} /> : null}

                {isTicketFetching ? (
                  <EditorNoticeCard message="Обновляем данные тикета..." />
                ) : null}

                <div className="grid gap-4 md:grid-cols-3">
                  <SummaryCard label="Статус" value={selectedStatusTitle} />
                  <SummaryCard label="Клиент" value={selectedClientTitle} />
                  <SummaryCard label="Исполнитель" value={selectedExecutorTitle} />
                </div>

                <section className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_320px]">
                  <div className="space-y-5 rounded-3xl border border-white/10 bg-white/5 p-5">
                    <div className="grid gap-5 md:grid-cols-2">
                      <EditorFormField label="Статус">
                        <div className="mt-3">
                          <SelectField
                            name="status"
                            value={formState.status}
                            onChange={handleFormChange}
                            className={editorSelectClassName}
                          >
                            <option value="">Выберите статус</option>
                            {ticketStatuses.map((status) => (
                              <option key={status.id} value={status.id}>
                                {status.title || status.id}
                              </option>
                            ))}
                          </SelectField>
                        </div>
                      </EditorFormField>

                      <EditorFormField label="Причина">
                        <div className="mt-3">
                          <SelectField
                            name="reason"
                            value={formState.reason}
                            onChange={handleFormChange}
                            className={editorSelectClassName}
                          >
                            <option value="">Не указана</option>
                            {ticketReasons.map((reason) => (
                              <option key={reason.id} value={reason.id}>
                                {reason.title || reason.id}
                              </option>
                            ))}
                          </SelectField>
                        </div>
                      </EditorFormField>

                      <EditorFormField label="Тип тикета">
                        <div className="mt-3">
                          <SelectField
                            name="ticketType"
                            value={formState.ticketType}
                            onChange={handleFormChange}
                            className={editorSelectClassName}
                          >
                            <option value="">Не указан</option>
                            {ticketTypes.map((ticketType) => (
                              <option key={ticketType.id} value={ticketType.id}>
                                {ticketType.title || ticketType.id}
                              </option>
                            ))}
                          </SelectField>
                        </div>
                      </EditorFormField>

                      <EditorFormField label="Отдел">
                        <div className="mt-3">
                          <SelectField
                            name="department"
                            value={formState.department}
                            onChange={handleFormChange}
                            className={editorSelectClassName}
                          >
                            <option value="">Не указан</option>
                            {departments.map((department) => (
                              <option key={department.id} value={department.id}>
                                {department.title || "Без названия"}
                              </option>
                            ))}
                          </SelectField>
                        </div>
                      </EditorFormField>
                    </div>

                    <EditorFormField label="Клиент">
                      <div className="mt-3">
                        <AsyncSearchSelect
                          value={formState.client}
                          selectedLabel={selectedClientTitle}
                          options={clientOptions}
                          onSearchChange={setClientSearchValue}
                          onSelect={(option) => {
                            setSelectedClientOption(option ? { id: option.id, label: option.title || "Без названия" } : null);
                            setSelectedContactOption(null);
                            setFormState((currentState) => ({
                              ...currentState,
                              client: option?.id || "",
                              contactPerson: "",
                            }));
                          }}
                          getOptionLabel={(option) => option.title || "Без названия"}
                          errorMessage={clientOptionsError ? "Не удалось загрузить клиентов." : ""}
                          searchPlaceholder="Введите название клиента"
                          emptyMessage="Клиенты не найдены."
                        />
                      </div>
                    </EditorFormField>

                    <EditorFormField label="Устройство">
                      <div className="mt-3">
                        <AsyncSearchSelect
                          value={formState.device}
                          selectedLabel={selectedDeviceTitle}
                          options={deviceOptions}
                          onSearchChange={setDeviceSearchValue}
                          onSelect={(option) => {
                            setSelectedDeviceOption(option ? { id: option.id, label: getDeviceOptionLabel(option) } : null);
                            setFormState((currentState) => ({
                              ...currentState,
                              device: option?.id || "",
                            }));
                          }}
                          getOptionDescription={(option) => option.clientName?.trim() || ""}
                          getOptionLabel={getDeviceOptionLabel}
                          errorMessage={deviceOptionsError ? "Не удалось загрузить устройства." : ""}
                          searchPlaceholder="Введите название или серийный номер"
                          emptyMessage="Устройства не найдены."
                        />
                      </div>
                    </EditorFormField>

                    <div className="grid gap-5 md:grid-cols-2">
                      <EditorFormField label="Контакт">
                        <div className="mt-3">
                          <AsyncSearchSelect
                            value={formState.contactPerson}
                            selectedLabel={selectedContactTitle}
                            options={contactOptions}
                            onSelect={(option) => {
                              setSelectedContactOption(option ? { id: option.id, label: option.name || "Контакт без имени" } : null);
                              setFormState((currentState) => ({
                                ...currentState,
                                contactPerson: option?.id || "",
                              }));
                            }}
                            getOptionDescription={(option) => option.position?.trim() || ""}
                            getOptionLabel={(option) => option.name || "Контакт без имени"}
                            errorMessage={
                              !formState.client
                                ? "Сначала выберите клиента."
                                : contactOptionsError
                                  ? "Не удалось загрузить контакты."
                                  : ""
                            }
                            emptyMessage="Контакты не найдены."
                            searchPlaceholder="Контакты загружаются автоматически"
                            disabled={!formState.client}
                          />
                        </div>
                      </EditorFormField>

                      <EditorFormField label="Исполнитель">
                        <div className="mt-3">
                          <AsyncSearchSelect
                            value={formState.executor}
                            selectedLabel={selectedExecutorTitle}
                            options={executorOptions}
                            onSearchChange={setExecutorSearchValue}
                            onSelect={(option) => {
                              setSelectedExecutorOption(
                                option
                                  ? {
                                      id: option.id,
                                      label: getAccountOptionLabel(option),
                                    }
                                  : null,
                              );
                              setFormState((currentState) => ({
                                ...currentState,
                                executor: option?.id || "",
                              }));
                            }}
                            getOptionDescription={getAccountOptionDescription}
                            getOptionLabel={getAccountOptionLabel}
                            errorMessage={executorOptionsError ? "Не удалось загрузить сотрудников." : ""}
                            searchPlaceholder="Введите имя, логин или отдел"
                            emptyMessage="Сотрудники не найдены."
                          />
                        </div>
                      </EditorFormField>
                    </div>

                    <EditorFormField label="Описание">
                      <textarea
                        name="description"
                        value={formState.description}
                        onChange={handleFormChange}
                        className={editorTextareaClassName}
                        rows={6}
                        placeholder="Что происходит по тикету"
                      />
                    </EditorFormField>

                    <EditorFormField label="Результат работы">
                      <textarea
                        name="result"
                        value={formState.result}
                        onChange={handleFormChange}
                        className={editorTextareaClassName}
                        rows={6}
                        placeholder="Чем завершилась работа"
                      />
                    </EditorFormField>

                    <div className="grid gap-5 md:grid-cols-2">
                      <EditorFormField label="Плановый старт">
                        <input
                          type="datetime-local"
                          name="assignedStart"
                          value={formState.assignedStart}
                          onChange={handleFormChange}
                          className={editorFieldClassName}
                        />
                      </EditorFormField>

                      <EditorFormField label="Плановое завершение">
                        <input
                          type="datetime-local"
                          name="assignedEnd"
                          value={formState.assignedEnd}
                          onChange={handleFormChange}
                          className={editorFieldClassName}
                        />
                      </EditorFormField>

                      <EditorFormField label="Начало работ">
                        <input
                          type="datetime-local"
                          name="workstartedAt"
                          value={formState.workstartedAt}
                          onChange={handleFormChange}
                          className={editorFieldClassName}
                        />
                      </EditorFormField>

                      <EditorFormField label="Завершение работ">
                        <input
                          type="datetime-local"
                          name="workfinishedAt"
                          value={formState.workfinishedAt}
                          onChange={handleFormChange}
                          className={editorFieldClassName}
                        />
                      </EditorFormField>

                      <EditorFormField label="Закрыт">
                        <input
                          type="datetime-local"
                          name="closedAt"
                          value={formState.closedAt}
                          onChange={handleFormChange}
                          className={editorFieldClassName}
                        />
                      </EditorFormField>
                    </div>

                    <div className="flex flex-wrap gap-3">
                      <EditorCheckboxField
                        name="urgent"
                        checked={formState.urgent}
                        onChange={handleCheckboxChange}
                        label="Срочный тикет"
                      />
                      <EditorCheckboxField
                        name="doubleSigned"
                        checked={formState.doubleSigned}
                        onChange={handleCheckboxChange}
                        label="Двусторонний акт"
                      />
                    </div>
                  </div>

                  <aside className="space-y-4 rounded-3xl border border-white/10 bg-slate-950/25 p-5">
                    <SummaryCard label="Причина" value={selectedReasonTitle} />
                    <SummaryCard label="Устройство" value={selectedDeviceTitle} />
                    <SummaryCard label="Отдел" value={selectedDepartmentTitle} />
                    <SummaryCard label="Контакт" value={selectedContactTitle} />
                    <SummaryCard
                      label="Создан"
                      value={selectedTicket.createdAt ? formatTicketDateTime(selectedTicket.createdAt) : "Не указано"}
                    />
                  </aside>
                </section>

                {ticketStatusesError || ticketTypesError || ticketReasonsError || departmentsError ? (
                  <StatusMessage
                    feedback={{
                      message: "Часть справочников не загрузилась. Проверьте статус, тип, причину и отдел перед сохранением.",
                      tone: "error",
                    }}
                  />
                ) : null}
              </>
            ) : null}
          </EditorPane>
        </EditorWorkspace>
      </section>
    </PageShell>
  );
}
