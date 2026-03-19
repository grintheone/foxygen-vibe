import { useDeferredValue, useRef, useState } from "react";
import { useNavigate, useSearchParams } from "react-router";
import {
  useGetEditorClassificatorsQuery,
  useGetEditorDeviceByIdQuery,
  useGetEditorDevicesQuery,
  usePatchEditorDeviceMutation,
} from "../../../shared/api/editor-api";
import { routePaths } from "../../../shared/config/routes";
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
  EditorSidebar,
  editorTextareaClassName,
  EditorWorkspace,
  SummaryCard,
  useSyncedSidebarHeight,
} from "./editor-shared";
import { useEditorSearchParamSelection, useLoadedEditorRecord, useUnsavedChangesWarning } from "./editor-hooks";

function DeviceListItem({ device, isActive, onClick }) {
  const title = device.title?.trim() || "Классификатор не указан";
  const meta = [device.serialNumber?.trim() ? `С/Н: ${device.serialNumber.trim()}` : "", device.clientName?.trim()]
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
          <p className="text-sm text-slate-400">{meta || "Серийный номер и клиент пока не указаны."}</p>
        </div>
        <div className="shrink-0 text-right text-xs text-slate-400">
          <p>{device.connectedToLis ? "LIS" : "Без LIS"}</p>
          <p className="mt-1">{device.isUsed ? "Б/У" : "Новое"}</p>
        </div>
      </div>
    </button>
  );
}

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

function formatEditorJson(value) {
  if (!value) {
    return "{}";
  }

  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return "{}";
  }
}

function getDeviceFormState(device) {
  return {
    classificator: device.classificator || "",
    connectedToLis: Boolean(device.connectedToLis),
    isUsed: Boolean(device.isUsed),
    properties: formatEditorJson(device.properties),
    serialNumber: device.serialNumber || "",
  };
}

export function EditorDevicesPage() {
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const editorPaneRef = useRef(null);
  const selectedDeviceId = searchParams.get("deviceId") || "";
  const [searchValue, setSearchValue] = useState("");
  const deferredSearchValue = useDeferredValue(searchValue.trim());
  const [formState, setFormState] = useState({
    classificator: "",
    connectedToLis: false,
    isUsed: false,
    properties: "{}",
    serialNumber: "",
  });
  const [loadedDeviceId, setLoadedDeviceId] = useState("");
  const [feedback, setFeedback] = useState({
    message: "",
    tone: "idle",
  });
  const [patchEditorDevice, { isLoading: isSaving }] = usePatchEditorDeviceMutation();
  const {
    data: devices = [],
    error: devicesError,
    isFetching: isDevicesFetching,
    isLoading: isDevicesLoading,
  } = useGetEditorDevicesQuery({
    limit: 50,
    q: deferredSearchValue,
  });
  const {
    data: classificators = [],
    error: classificatorsError,
    isFetching: isClassificatorsFetching,
    isLoading: isClassificatorsLoading,
  } = useGetEditorClassificatorsQuery();
  const {
    data: selectedDevice,
    error: selectedDeviceError,
    isFetching: isDeviceFetching,
    isLoading: isDeviceLoading,
  } = useGetEditorDeviceByIdQuery(selectedDeviceId, {
    skip: !selectedDeviceId,
  });

  const initialProperties = formatEditorJson(selectedDevice?.properties);
  const isDirty =
    Boolean(selectedDeviceId) &&
    (formState.classificator !== (selectedDevice?.classificator || "") ||
      formState.serialNumber !== (selectedDevice?.serialNumber || "") ||
      formState.properties !== initialProperties ||
      formState.connectedToLis !== Boolean(selectedDevice?.connectedToLis) ||
      formState.isUsed !== Boolean(selectedDevice?.isUsed));
  const selectedClassificatorTitle =
    classificators.find((classificator) => classificator.id === formState.classificator)?.title ||
    selectedDevice?.title?.trim() ||
    "Не указан";
  const selectedClientTitle = selectedDevice?.clientName?.trim() || "Не указан";
  const selectedAgreementTitle = selectedDevice?.agreementNumber
    ? `Договор #${selectedDevice.agreementNumber}`
    : "Не найден";
  const sidebarHeight = useSyncedSidebarHeight(editorPaneRef);
  const handleSelectDevice = useEditorSearchParamSelection({
    isDirty,
    items: devices,
    paramKey: "deviceId",
    selectedId: selectedDeviceId,
    setSearchParams,
  });

  useLoadedEditorRecord({
    loadedRecordId: loadedDeviceId,
    onRecordLoad: () =>
      setFeedback({
        message: "",
        tone: "idle",
      }),
    record: selectedDevice,
    setFormState,
    setLoadedRecordId: setLoadedDeviceId,
    toFormState: getDeviceFormState,
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
    if (!selectedDeviceId) {
      return;
    }

    try {
      JSON.parse(formState.properties || "{}");
    } catch {
      setFeedback({
        message: "Поле параметров должно содержать валидный JSON.",
        tone: "error",
      });
      return;
    }

    try {
      const updatedDevice = await patchEditorDevice({
        deviceId: selectedDeviceId,
        patch: formState,
      }).unwrap();

      setFormState(getDeviceFormState(updatedDevice));
      setLoadedDeviceId(updatedDevice.id);
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
          title="Устройства"
          description="Здесь можно редактировать устройство, его классификатор, серийный номер и JSON-параметры без прямого доступа к базе."
          action={<BackButton onClick={handleBack} />}
        />

        <EditorWorkspace
          sidebar={
            <EditorSidebar
              height={sidebarHeight}
              footer={isDevicesFetching ? "Обновляем список..." : `Показано ${devices.length} записей.`}
            >
              <div className="space-y-4">
                <EditorListHeader title="Карточки устройств" />
                <EditorSearchField
                  value={searchValue}
                  onChange={(event) => setSearchValue(event.target.value)}
                  placeholder="Классификатор, серийный номер, клиент"
                />
                <EditorListError error={devicesError} fallbackMessage="Не удалось загрузить список устройств." />
              </div>

              <div className="min-h-0 flex-1 space-y-3 overflow-y-auto pr-1">
                {isDevicesLoading ? <EditorNoticeCard message="Загружаем устройства..." /> : null}
                {!isDevicesLoading && devices.length === 0 ? (
                  <EditorNoticeCard message="По текущему запросу ничего не найдено." />
                ) : null}

                {devices.map((device) => (
                  <DeviceListItem
                    key={device.id}
                    device={device}
                    isActive={device.id === selectedDeviceId}
                    onClick={() => handleSelectDevice(device.id)}
                  />
                ))}
              </div>
            </EditorSidebar>
          }
        >
          <EditorPane editorPaneRef={editorPaneRef}>
            {!selectedDeviceId ? (
              <EditorNoticeCard dashed message="Выберите устройство слева, чтобы открыть карточку редактора." />
            ) : null}

            {selectedDeviceId && isDeviceLoading ? <EditorNoticeCard message="Загружаем карточку устройства..." /> : null}

            {selectedDeviceId && selectedDeviceError ? (
              <EditorListError error={selectedDeviceError} fallbackMessage="Не удалось загрузить карточку устройства." />
            ) : null}

            {selectedDevice ? (
              <>
                <EditorRecordHeader
                  id={selectedDevice.id}
                  isDirty={isDirty}
                  isSaving={isSaving}
                  onSave={handleSave}
                  title={selectedDevice.title?.trim() || "Классификатор не указан"}
                  titleLabel="Карточка устройства"
                />

                {feedback.message ? <StatusMessage feedback={feedback} /> : null}

                <div className="grid gap-4 md:grid-cols-3">
                  <SummaryCard label="Классификатор" value={selectedClassificatorTitle} />
                  <SummaryCard label="Клиент" value={selectedClientTitle} />
                  <SummaryCard label="Договор" value={selectedAgreementTitle} />
                </div>

                <section className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_320px]">
                  <div className="space-y-5 rounded-3xl border border-white/10 bg-white/5 p-5">
                    <EditorFormField label="Классификатор" hint="Здесь меняется отображаемое название устройства.">
                      <div className="mt-3">
                        <SelectField
                          name="classificator"
                          value={formState.classificator}
                          onChange={handleFormChange}
                          disabled={isClassificatorsLoading}
                          className={editorSelectClassName}
                        >
                          <option value="">Не указан</option>
                          {classificators.map((classificator) => (
                            <option key={classificator.id} value={classificator.id}>
                              {classificator.title || "Без названия"}
                            </option>
                          ))}
                        </SelectField>
                      </div>
                    </EditorFormField>

                    <EditorFormField label="Серийный номер">
                      <input
                        type="text"
                        name="serialNumber"
                        value={formState.serialNumber}
                        onChange={handleFormChange}
                        placeholder="Введите серийный номер"
                        className={editorFieldClassName}
                      />
                    </EditorFormField>

                    <EditorFormField
                      label="Параметры JSON"
                      hint="Поле сохраняется как JSONB. Можно оставить `{}` или передать полноценный объект параметров."
                    >
                      <textarea
                        name="properties"
                        value={formState.properties}
                        onChange={handleFormChange}
                        rows="12"
                        spellCheck={false}
                        placeholder='{\n  "model": "XP-1000"\n}'
                        className={editorTextareaClassName}
                      />
                    </EditorFormField>

                    <div className="flex flex-wrap gap-3">
                      <EditorCheckboxField
                        name="connectedToLis"
                        checked={formState.connectedToLis}
                        onChange={handleCheckboxChange}
                        label="Подключено к LIS"
                      />
                      <EditorCheckboxField
                        name="isUsed"
                        checked={formState.isUsed}
                        onChange={handleCheckboxChange}
                        label="Б/У устройство"
                      />
                    </div>
                  </div>

                  <aside className="space-y-4 rounded-3xl border border-white/10 bg-slate-950/35 p-5">
                    <div>
                      <p className="text-xs font-semibold uppercase tracking-[0.22em] text-slate-400">Текущий контекст</p>
                      <div className="mt-4 space-y-3 text-sm text-slate-300">
                        <p>
                          <span className="text-slate-500">Клиент:</span> {selectedClientTitle}
                        </p>
                        <p>
                          <span className="text-slate-500">Адрес:</span> {selectedDevice.clientAddress?.trim() || "Не указан"}
                        </p>
                        <p>
                          <span className="text-slate-500">Договор:</span> {selectedAgreementTitle}
                        </p>
                        <p>
                          <span className="text-slate-500">Статус договора:</span>{" "}
                          {selectedDevice.agreement
                            ? selectedDevice.isActiveAgreement
                              ? "Активный"
                              : "Неактивный"
                            : "Не найден"}
                        </p>
                        <p>
                          <span className="text-slate-500">Гарантия:</span>{" "}
                          {selectedDevice.agreement ? (selectedDevice.onWarranty ? "Да" : "Нет") : "Не указана"}
                        </p>
                      </div>
                    </div>

                    <div>
                      <p className="text-xs font-semibold uppercase tracking-[0.22em] text-slate-400">Служебные флаги</p>
                      <div className="mt-4 space-y-3 text-sm text-slate-300">
                        <p>
                          <span className="text-slate-500">LIS:</span> {formState.connectedToLis ? "Подключено" : "Не подключено"}
                        </p>
                        <p>
                          <span className="text-slate-500">Состояние:</span> {formState.isUsed ? "Б/У" : "Новое"}
                        </p>
                      </div>
                    </div>

                    <p className="text-xs text-slate-500">
                      Карточка изменяет только базовые поля устройства. Клиент и договор здесь показываются как
                      справочный контекст.
                    </p>
                    {classificatorsError ? (
                      <p className="text-xs text-rose-300">
                        {typeof classificatorsError?.data === "string"
                          ? classificatorsError.data
                          : "Не удалось загрузить список классификаторов."}
                      </p>
                    ) : null}
                  </aside>
                </section>

                <p className="text-xs text-slate-500">
                  {isDeviceFetching
                    ? "Обновляем карточку..."
                    : isClassificatorsFetching
                      ? "Обновляем список классификаторов..."
                      : "Изменения применяются к базовым полям устройства и не трогают связанные договоры."}
                </p>
              </>
            ) : null}
          </EditorPane>
        </EditorWorkspace>
      </section>
    </PageShell>
  );
}
