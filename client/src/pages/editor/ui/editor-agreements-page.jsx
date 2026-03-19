import { useDeferredValue, useRef, useState } from "react";
import { useNavigate, useSearchParams } from "react-router";
import {
  useGetEditorAgreementByIdQuery,
  useGetEditorAgreementsQuery,
  useGetEditorClientsQuery,
  useGetEditorDevicesQuery,
  usePatchEditorAgreementMutation,
} from "../../../shared/api/editor-api";
import { routePaths } from "../../../shared/config/routes";
import { AsyncSearchSelect } from "../../../shared/ui/async-search-select";
import { PageShell } from "../../../shared/ui/page-shell";
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

function formatAgreementDateTime(value) {
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

function toIsoDateTimeValue(value) {
  const normalizedValue = value.trim();
  if (!normalizedValue) {
    return "";
  }

  const date = new Date(normalizedValue);
  if (Number.isNaN(date.getTime())) {
    return null;
  }

  return date.toISOString();
}

function getDeviceOptionLabel(device) {
  const title = device.title?.trim() || "Без классификатора";
  const serialNumber = device.serialNumber?.trim();
  return serialNumber ? `${title} • ${serialNumber}` : title;
}

function getClientOptionLabel(client) {
  return client?.title?.trim() || "Без названия";
}

function createClientSelection(clientId, title) {
  if (!clientId) {
    return null;
  }

  return {
    id: clientId,
    label: title?.trim() || "Без названия",
  };
}

function createDeviceSelection(deviceId, title, serialNumber) {
  if (!deviceId) {
    return null;
  }

  return {
    id: deviceId,
    label: getDeviceOptionLabel({
      serialNumber,
      title,
    }),
  };
}

function getAgreementDeviceLabel(agreement) {
  if (!agreement) {
    return "Не выбрано";
  }

  const title = agreement.deviceTitle?.trim() || "Без классификатора";
  const serialNumber = agreement.deviceSerialNumber?.trim();
  return agreement.device ? (serialNumber ? `${title} • ${serialNumber}` : title) : "Не выбрано";
}

function AgreementListItem({ agreement, isActive, onClick }) {
  const title = agreement.number ? `Договор #${agreement.number}` : "Договор без номера";
  const meta = [agreement.actualClientName?.trim(), getAgreementDeviceLabel(agreement)].filter(Boolean).join(" • ");

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
          <p className="text-sm text-slate-400">{meta || "Клиент и устройство пока не указаны."}</p>
          <p className="text-xs text-slate-500">
            {agreement.assignedAt ? `Назначен: ${formatAgreementDateTime(agreement.assignedAt)}` : "Дата начала не указана"}
          </p>
        </div>
        <div className="shrink-0 text-right text-xs text-slate-400">
          <p>{agreement.isActive ? "Активный" : "Неактивный"}</p>
          <p className="mt-1">{agreement.onWarranty ? "На гарантии" : "Без гарантии"}</p>
        </div>
      </div>
    </button>
  );
}

function getAgreementFormState(agreement) {
  return {
    actualClient: agreement.actualClient || "",
    assignedAt: toDateTimeLocalValue(agreement.assignedAt),
    device: agreement.device || "",
    distributor: agreement.distributor || "",
    finishedAt: toDateTimeLocalValue(agreement.finishedAt),
    isActive: Boolean(agreement.isActive),
    onWarranty: Boolean(agreement.onWarranty),
  };
}

export function EditorAgreementsPage() {
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const editorPaneRef = useRef(null);
  const selectedAgreementId = searchParams.get("agreementId") || "";
  const [searchValue, setSearchValue] = useState("");
  const deferredSearchValue = useDeferredValue(searchValue.trim());
  const [formState, setFormState] = useState({
    actualClient: "",
    assignedAt: "",
    device: "",
    distributor: "",
    finishedAt: "",
    isActive: false,
    onWarranty: false,
  });
  const [loadedAgreementId, setLoadedAgreementId] = useState("");
  const [actualClientSearchValue, setActualClientSearchValue] = useState("");
  const [distributorSearchValue, setDistributorSearchValue] = useState("");
  const [deviceSearchValue, setDeviceSearchValue] = useState("");
  const deferredActualClientSearchValue = useDeferredValue(actualClientSearchValue.trim());
  const deferredDistributorSearchValue = useDeferredValue(distributorSearchValue.trim());
  const deferredDeviceSearchValue = useDeferredValue(deviceSearchValue.trim());
  const [selectedActualClientOption, setSelectedActualClientOption] = useState(null);
  const [selectedDistributorOption, setSelectedDistributorOption] = useState(null);
  const [selectedDeviceOption, setSelectedDeviceOption] = useState(null);
  const [feedback, setFeedback] = useState({
    message: "",
    tone: "idle",
  });
  const [patchEditorAgreement, { isLoading: isSaving }] = usePatchEditorAgreementMutation();
  const {
    data: agreements = [],
    error: agreementsError,
    isFetching: isAgreementsFetching,
    isLoading: isAgreementsLoading,
  } = useGetEditorAgreementsQuery({
    limit: 50,
    q: deferredSearchValue,
  });
  const {
    data: actualClientOptions = [],
    error: actualClientOptionsError,
    isFetching: isActualClientOptionsFetching,
    isLoading: isActualClientOptionsLoading,
  } = useGetEditorClientsQuery({
    limit: 20,
    q: deferredActualClientSearchValue,
  }, {
    skip: !selectedAgreementId,
  });
  const {
    data: distributorOptions = [],
    error: distributorOptionsError,
    isFetching: isDistributorOptionsFetching,
    isLoading: isDistributorOptionsLoading,
  } = useGetEditorClientsQuery({
    limit: 20,
    q: deferredDistributorSearchValue,
  }, {
    skip: !selectedAgreementId,
  });
  const {
    data: deviceOptions = [],
    error: deviceOptionsError,
    isFetching: isDeviceOptionsFetching,
    isLoading: isDeviceOptionsLoading,
  } = useGetEditorDevicesQuery({
    limit: 20,
    q: deferredDeviceSearchValue,
  }, {
    skip: !selectedAgreementId,
  });
  const {
    data: selectedAgreement,
    error: selectedAgreementError,
    isFetching: isAgreementFetching,
    isLoading: isAgreementLoading,
  } = useGetEditorAgreementByIdQuery(selectedAgreementId, {
    skip: !selectedAgreementId,
  });

  const initialAssignedAt = toDateTimeLocalValue(selectedAgreement?.assignedAt);
  const initialFinishedAt = toDateTimeLocalValue(selectedAgreement?.finishedAt);
  const isDirty =
    Boolean(selectedAgreementId) &&
    (formState.actualClient !== (selectedAgreement?.actualClient || "") ||
      formState.distributor !== (selectedAgreement?.distributor || "") ||
      formState.device !== (selectedAgreement?.device || "") ||
      formState.assignedAt !== initialAssignedAt ||
      formState.finishedAt !== initialFinishedAt ||
      formState.isActive !== Boolean(selectedAgreement?.isActive) ||
      formState.onWarranty !== Boolean(selectedAgreement?.onWarranty));
  const selectedActualClientTitle =
    selectedActualClientOption?.label ||
    (formState.actualClient === selectedAgreement?.actualClient ? selectedAgreement?.actualClientName?.trim() : "") ||
    "Не выбран";
  const selectedDistributorTitle =
    selectedDistributorOption?.label ||
    (formState.distributor === selectedAgreement?.distributor ? selectedAgreement?.distributorName?.trim() : "") ||
    "Не указан";
  const selectedDeviceLabel =
    selectedDeviceOption?.label ||
    (formState.device === selectedAgreement?.device ? getAgreementDeviceLabel(selectedAgreement) : "") ||
    "Не выбрано";
  const sidebarHeight = useSyncedSidebarHeight(editorPaneRef);
  const handleSelectAgreement = useEditorSearchParamSelection({
    isDirty,
    items: agreements,
    paramKey: "agreementId",
    selectedId: selectedAgreementId,
    setSearchParams,
  });

  useLoadedEditorRecord({
    loadedRecordId: loadedAgreementId,
    onRecordLoad: (agreement) => {
      setSelectedActualClientOption(createClientSelection(agreement.actualClient, agreement.actualClientName));
      setSelectedDistributorOption(createClientSelection(agreement.distributor, agreement.distributorName));
      setSelectedDeviceOption(createDeviceSelection(agreement.device, agreement.deviceTitle, agreement.deviceSerialNumber));
      setFeedback({
        message: "",
        tone: "idle",
      });
    },
    record: selectedAgreement,
    setFormState,
    setLoadedRecordId: setLoadedAgreementId,
    toFormState: getAgreementFormState,
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

  function handleActualClientSelect(client) {
    setFormState((currentState) => ({
      ...currentState,
      actualClient: client?.id || "",
    }));
    setSelectedActualClientOption(client ? createClientSelection(client.id, getClientOptionLabel(client)) : null);
  }

  function handleDistributorSelect(client) {
    setFormState((currentState) => ({
      ...currentState,
      distributor: client?.id || "",
    }));
    setSelectedDistributorOption(client ? createClientSelection(client.id, getClientOptionLabel(client)) : null);
  }

  function handleDeviceSelect(device) {
    setFormState((currentState) => ({
      ...currentState,
      device: device?.id || "",
    }));
    setSelectedDeviceOption(device ? createDeviceSelection(device.id, device.title, device.serialNumber) : null);
  }

  async function handleSave() {
    if (!selectedAgreementId) {
      return;
    }

    if (!formState.actualClient) {
      setFeedback({
        message: "Выберите фактического клиента.",
        tone: "error",
      });
      return;
    }

    const assignedAt = toIsoDateTimeValue(formState.assignedAt);
    if (assignedAt === null) {
      setFeedback({
        message: "Дата начала должна быть корректной.",
        tone: "error",
      });
      return;
    }

    const finishedAt = toIsoDateTimeValue(formState.finishedAt);
    if (finishedAt === null) {
      setFeedback({
        message: "Дата завершения должна быть корректной.",
        tone: "error",
      });
      return;
    }

    try {
      const updatedAgreement = await patchEditorAgreement({
        agreementId: selectedAgreementId,
        patch: {
          actualClient: formState.actualClient,
          assignedAt,
          device: formState.device,
          distributor: formState.distributor,
          finishedAt,
          isActive: formState.isActive,
          onWarranty: formState.onWarranty,
        },
      }).unwrap();

      setFormState(getAgreementFormState(updatedAgreement));
      setLoadedAgreementId(updatedAgreement.id);
      setSelectedActualClientOption(createClientSelection(updatedAgreement.actualClient, updatedAgreement.actualClientName));
      setSelectedDistributorOption(createClientSelection(updatedAgreement.distributor, updatedAgreement.distributorName));
      setSelectedDeviceOption(
        createDeviceSelection(updatedAgreement.device, updatedAgreement.deviceTitle, updatedAgreement.deviceSerialNumber),
      );
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
          title="Договоры"
          description="Здесь можно исправлять связки договора с клиентом, дистрибьютором и устройством, а также служебные даты и флаги."
          action={<BackButton onClick={handleBack} />}
        />

        <EditorWorkspace
          sidebar={
            <EditorSidebar
              height={sidebarHeight}
              footer={isAgreementsFetching ? "Обновляем список..." : `Показано ${agreements.length} записей.`}
            >
              <div className="space-y-4">
                <EditorListHeader title="Договорные записи" />
                <EditorSearchField
                  value={searchValue}
                  onChange={(event) => setSearchValue(event.target.value)}
                  placeholder="Номер, клиент, дистрибьютор, устройство"
                />
                <EditorListError error={agreementsError} fallbackMessage="Не удалось загрузить список договоров." />
              </div>

              <div className="min-h-0 flex-1 space-y-3 overflow-y-auto pr-1">
                {isAgreementsLoading ? <EditorNoticeCard message="Загружаем договоры..." /> : null}
                {!isAgreementsLoading && agreements.length === 0 ? (
                  <EditorNoticeCard message="По текущему запросу ничего не найдено." />
                ) : null}

                {agreements.map((agreement) => (
                  <AgreementListItem
                    key={agreement.id}
                    agreement={agreement}
                    isActive={agreement.id === selectedAgreementId}
                    onClick={() => handleSelectAgreement(agreement.id)}
                  />
                ))}
              </div>
            </EditorSidebar>
          }
        >
          <EditorPane editorPaneRef={editorPaneRef}>
            {!selectedAgreementId ? (
              <EditorNoticeCard dashed message="Выберите договор слева, чтобы открыть карточку редактора." />
            ) : null}

            {selectedAgreementId && isAgreementLoading ? <EditorNoticeCard message="Загружаем карточку договора..." /> : null}

            {selectedAgreementId && selectedAgreementError ? (
              <EditorListError error={selectedAgreementError} fallbackMessage="Не удалось загрузить карточку договора." />
            ) : null}

            {selectedAgreement ? (
              <>
                <EditorRecordHeader
                  id={selectedAgreement.id}
                  isDirty={isDirty}
                  isSaving={isSaving}
                  onSave={handleSave}
                  title={selectedAgreement.number ? `Договор #${selectedAgreement.number}` : "Договор без номера"}
                  titleLabel="Карточка договора"
                />

                {feedback.message ? <StatusMessage feedback={feedback} /> : null}

                <div className="grid gap-4 md:grid-cols-3">
                  <SummaryCard label="Номер" value={selectedAgreement.number ? `#${selectedAgreement.number}` : "Не указан"} />
                  <SummaryCard label="Клиент" value={selectedActualClientTitle} />
                  <SummaryCard label="Устройство" value={selectedDeviceLabel} />
                </div>

                <section className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_320px]">
                  <div className="space-y-5 rounded-3xl border border-white/10 bg-white/5 p-5">
                    <EditorFormField label="Фактический клиент" hint="Основная привязка договора. Поле обязательно.">
                      <div className="mt-3">
                        <AsyncSearchSelect
                          value={formState.actualClient}
                          selectedLabel={selectedActualClientTitle}
                          onSelect={handleActualClientSelect}
                          onSearchChange={setActualClientSearchValue}
                          options={actualClientOptions}
                          getOptionLabel={getClientOptionLabel}
                          getOptionDescription={(client) => client.address?.trim() || client.regionTitle?.trim() || ""}
                          placeholder="Выберите клиента"
                          searchPlaceholder="Поиск по названию или адресу"
                          emptyMessage="Клиенты не найдены."
                          clearLabel="Очистить клиента"
                          disabled={isActualClientOptionsLoading && actualClientOptions.length === 0}
                          isLoading={isActualClientOptionsLoading || isActualClientOptionsFetching}
                          errorMessage={
                            actualClientOptionsError
                              ? typeof actualClientOptionsError?.data === "string"
                                ? actualClientOptionsError.data
                                : "Не удалось загрузить клиентов."
                              : ""
                          }
                        />
                      </div>
                    </EditorFormField>

                    <EditorFormField label="Дистрибьютор" hint="Необязательная привязка к клиенту-поставщику.">
                      <div className="mt-3">
                        <AsyncSearchSelect
                          value={formState.distributor}
                          selectedLabel={selectedDistributorTitle}
                          onSelect={handleDistributorSelect}
                          onSearchChange={setDistributorSearchValue}
                          options={distributorOptions}
                          getOptionLabel={getClientOptionLabel}
                          getOptionDescription={(client) => client.address?.trim() || client.regionTitle?.trim() || ""}
                          placeholder="Не указан"
                          searchPlaceholder="Поиск дистрибьютора"
                          emptyMessage="Клиенты не найдены."
                          clearLabel="Убрать дистрибьютора"
                          disabled={isDistributorOptionsLoading && distributorOptions.length === 0}
                          isLoading={isDistributorOptionsLoading || isDistributorOptionsFetching}
                          errorMessage={
                            distributorOptionsError
                              ? typeof distributorOptionsError?.data === "string"
                                ? distributorOptionsError.data
                                : "Не удалось загрузить клиентов."
                              : ""
                          }
                        />
                      </div>
                    </EditorFormField>

                    <EditorFormField label="Устройство" hint="Можно отвязать договор от устройства, если связь была ошибочной.">
                      <div className="mt-3">
                        <AsyncSearchSelect
                          value={formState.device}
                          selectedLabel={selectedDeviceLabel}
                          onSelect={handleDeviceSelect}
                          onSearchChange={setDeviceSearchValue}
                          options={deviceOptions}
                          getOptionLabel={getDeviceOptionLabel}
                          getOptionDescription={(device) => device.clientName?.trim() || ""}
                          placeholder="Не выбрано"
                          searchPlaceholder="Поиск по классификатору или серийному номеру"
                          emptyMessage="Устройства не найдены."
                          clearLabel="Отвязать устройство"
                          disabled={isDeviceOptionsLoading && deviceOptions.length === 0}
                          isLoading={isDeviceOptionsLoading || isDeviceOptionsFetching}
                          errorMessage={
                            deviceOptionsError
                              ? typeof deviceOptionsError?.data === "string"
                                ? deviceOptionsError.data
                                : "Не удалось загрузить устройства."
                              : ""
                          }
                        />
                      </div>
                    </EditorFormField>

                    <EditorFormField label="Дата начала">
                      <input
                        type="datetime-local"
                        name="assignedAt"
                        value={formState.assignedAt}
                        onChange={handleFormChange}
                        className={editorFieldClassName}
                      />
                    </EditorFormField>

                    <EditorFormField label="Дата завершения">
                      <input
                        type="datetime-local"
                        name="finishedAt"
                        value={formState.finishedAt}
                        onChange={handleFormChange}
                        className={editorFieldClassName}
                      />
                    </EditorFormField>

                    <div className="flex flex-wrap gap-3">
                      <EditorCheckboxField
                        name="isActive"
                        checked={formState.isActive}
                        onChange={handleCheckboxChange}
                        label="Активный договор"
                      />
                      <EditorCheckboxField
                        name="onWarranty"
                        checked={formState.onWarranty}
                        onChange={handleCheckboxChange}
                        label="На гарантии"
                      />
                    </div>
                  </div>

                  <aside className="space-y-4 rounded-3xl border border-white/10 bg-slate-950/35 p-5">
                    <div>
                      <p className="text-xs font-semibold uppercase tracking-[0.22em] text-slate-400">Контекст договора</p>
                      <div className="mt-4 space-y-3 text-sm text-slate-300">
                        <p>
                          <span className="text-slate-500">Номер:</span> {selectedAgreement.number ? `#${selectedAgreement.number}` : "Не указан"}
                        </p>
                        <p>
                          <span className="text-slate-500">Клиент:</span> {selectedActualClientTitle}
                        </p>
                        <p>
                          <span className="text-slate-500">Дистрибьютор:</span> {selectedDistributorTitle}
                        </p>
                        <p>
                          <span className="text-slate-500">Устройство:</span> {selectedDeviceLabel}
                        </p>
                      </div>
                    </div>

                    <div>
                      <p className="text-xs font-semibold uppercase tracking-[0.22em] text-slate-400">Статус</p>
                      <div className="mt-4 space-y-3 text-sm text-slate-300">
                        <p>
                          <span className="text-slate-500">Активность:</span> {formState.isActive ? "Активный" : "Неактивный"}
                        </p>
                        <p>
                          <span className="text-slate-500">Гарантия:</span> {formState.onWarranty ? "Да" : "Нет"}
                        </p>
                        <p>
                          <span className="text-slate-500">Начало:</span> {formatAgreementDateTime(formState.assignedAt)}
                        </p>
                        <p>
                          <span className="text-slate-500">Завершение:</span> {formatAgreementDateTime(formState.finishedAt)}
                        </p>
                      </div>
                    </div>

                    <p className="text-xs text-slate-500">
                      Номер договора здесь только для чтения. Карточка меняет связи и служебные поля, не создавая новый
                      договор.
                    </p>
                    {actualClientOptionsError || distributorOptionsError ? (
                      <p className="text-xs text-rose-300">
                        {typeof actualClientOptionsError?.data === "string"
                          ? actualClientOptionsError.data
                          : typeof distributorOptionsError?.data === "string"
                            ? distributorOptionsError.data
                          : "Не удалось загрузить список клиентов."}
                      </p>
                    ) : null}
                    {deviceOptionsError ? (
                      <p className="text-xs text-rose-300">
                        {typeof deviceOptionsError?.data === "string"
                          ? deviceOptionsError.data
                          : "Не удалось загрузить список устройств."}
                      </p>
                    ) : null}
                  </aside>
                </section>

                <p className="text-xs text-slate-500">
                  {isAgreementFetching
                    ? "Обновляем карточку..."
                    : isActualClientOptionsFetching || isDistributorOptionsFetching
                      ? "Обновляем результаты поиска клиентов..."
                      : isDeviceOptionsFetching
                        ? "Обновляем результаты поиска устройств..."
                        : "Изменения применяются к связям договора, его датам и служебным флагам."}
                </p>
              </>
            ) : null}
          </EditorPane>
        </EditorWorkspace>
      </section>
    </PageShell>
  );
}
