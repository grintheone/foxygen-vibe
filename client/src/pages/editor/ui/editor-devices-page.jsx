import { useDeferredValue, useEffect, useRef, useState } from "react";
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
import { BackButton, EditorFormField, SummaryCard, useSyncedSidebarHeight } from "./editor-shared";

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

  function handleBack() {
    navigate(routePaths.editor);
  }

  useEffect(() => {
    if (devices.length === 0 || selectedDeviceId) {
      return;
    }

    setSearchParams((currentParams) => {
      const nextParams = new URLSearchParams(currentParams);
      nextParams.set("deviceId", devices[0].id);
      return nextParams;
    }, { replace: true });
  }, [devices, selectedDeviceId, setSearchParams]);

  useEffect(() => {
    if (!selectedDevice || selectedDevice.id === loadedDeviceId) {
      return;
    }

    setFormState({
      classificator: selectedDevice.classificator || "",
      connectedToLis: Boolean(selectedDevice.connectedToLis),
      isUsed: Boolean(selectedDevice.isUsed),
      properties: formatEditorJson(selectedDevice.properties),
      serialNumber: selectedDevice.serialNumber || "",
    });
    setLoadedDeviceId(selectedDevice.id);
    setFeedback({
      message: "",
      tone: "idle",
    });
  }, [loadedDeviceId, selectedDevice]);

  useEffect(() => {
    function handleBeforeUnload(event) {
      if (!isDirty) {
        return;
      }

      event.preventDefault();
      event.returnValue = "";
    }

    window.addEventListener("beforeunload", handleBeforeUnload);

    return () => {
      window.removeEventListener("beforeunload", handleBeforeUnload);
    };
  }, [isDirty]);

  function handleSelectDevice(nextDeviceId) {
    if (!nextDeviceId || nextDeviceId === selectedDeviceId) {
      return;
    }

    if (isDirty && !window.confirm("У вас есть несохраненные изменения. Перейти к другой записи?")) {
      return;
    }

    setSearchParams((currentParams) => {
      const nextParams = new URLSearchParams(currentParams);
      nextParams.set("deviceId", nextDeviceId);
      return nextParams;
    });
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

      setFormState({
        classificator: updatedDevice.classificator || "",
        connectedToLis: Boolean(updatedDevice.connectedToLis),
        isUsed: Boolean(updatedDevice.isUsed),
        properties: formatEditorJson(updatedDevice.properties),
        serialNumber: updatedDevice.serialNumber || "",
      });
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
        <header className="rounded-3xl border border-white/10 bg-slate-950/35 p-6 shadow-xl shadow-black/20 backdrop-blur">
          <div className="flex flex-wrap items-start justify-between gap-4">
            <div>
              <p className="text-xs font-semibold uppercase tracking-[0.3em] text-slate-400">Редактор</p>
              <h1 className="mt-3 text-3xl font-bold tracking-tight text-white sm:text-4xl">Устройства</h1>
              <p className="mt-3 max-w-2xl text-base text-slate-300">
                Здесь можно редактировать устройство, его классификатор, серийный номер и JSON-параметры без прямого
                доступа к базе.
              </p>
            </div>
            <BackButton onClick={handleBack} />
          </div>
        </header>

        <section className="grid gap-6 xl:grid-cols-[360px_minmax(0,1fr)]">
          <aside
            style={sidebarHeight ? { height: `${sidebarHeight}px` } : undefined}
            className="grid min-h-0 grid-rows-[auto_minmax(0,1fr)_auto] gap-4 overflow-hidden rounded-[2rem] border border-white/10 bg-white/10 p-5 shadow-2xl shadow-[#6A3BF2]/15 backdrop-blur-xl"
          >
            <div className="space-y-4">
              <div>
                <p className="text-xs font-semibold uppercase tracking-[0.32em] text-cyan-200">Список</p>
                <h2 className="mt-3 text-2xl font-bold tracking-tight text-white">Карточки устройств</h2>
              </div>

              <label className="block">
                <span className="text-xs font-semibold uppercase tracking-[0.22em] text-slate-400">Поиск</span>
                <input
                  type="search"
                  value={searchValue}
                  onChange={(event) => setSearchValue(event.target.value)}
                  placeholder="Классификатор, серийный номер, клиент"
                  className="mt-3 w-full rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3 text-sm text-white outline-none transition placeholder:text-slate-500 focus:border-cyan-200/40 focus:bg-slate-950/60"
                />
              </label>

              {devicesError ? (
                <StatusMessage
                  feedback={{
                    message:
                      typeof devicesError?.data === "string"
                        ? devicesError.data
                        : "Не удалось загрузить список устройств.",
                    tone: "error",
                  }}
                />
              ) : null}
            </div>

            <div className="min-h-0 flex-1 space-y-3 overflow-y-auto pr-1">
              {isDevicesLoading ? (
                <div className="rounded-3xl border border-white/10 bg-slate-950/25 p-5 text-sm text-slate-300">
                  Загружаем устройства...
                </div>
              ) : null}

              {!isDevicesLoading && devices.length === 0 ? (
                <div className="rounded-3xl border border-white/10 bg-slate-950/25 p-5 text-sm text-slate-300">
                  По текущему запросу ничего не найдено.
                </div>
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

            <p className="self-end text-xs text-slate-500">
              {isDevicesFetching ? "Обновляем список..." : `Показано ${devices.length} записей.`}
            </p>
          </aside>

          <section
            ref={editorPaneRef}
            className="space-y-6 rounded-[2rem] border border-white/10 bg-slate-950/30 p-6 shadow-2xl shadow-black/20 backdrop-blur-xl"
          >
            {!selectedDeviceId ? (
              <div className="rounded-3xl border border-dashed border-white/15 bg-white/5 p-8 text-slate-300">
                Выберите устройство слева, чтобы открыть карточку редактора.
              </div>
            ) : null}

            {selectedDeviceId && isDeviceLoading ? (
              <div className="rounded-3xl border border-white/10 bg-white/5 p-8 text-slate-300">
                Загружаем карточку устройства...
              </div>
            ) : null}

            {selectedDeviceId && selectedDeviceError ? (
              <StatusMessage
                feedback={{
                  message:
                    typeof selectedDeviceError?.data === "string"
                      ? selectedDeviceError.data
                      : "Не удалось загрузить карточку устройства.",
                  tone: "error",
                }}
              />
            ) : null}

            {selectedDevice ? (
              <>
                <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
                  <div>
                    <p className="text-xs font-semibold uppercase tracking-[0.32em] text-cyan-200">Карточка устройства</p>
                    <h2 className="mt-3 text-3xl font-bold tracking-tight text-white">
                      {selectedDevice.title?.trim() || "Классификатор не указан"}
                    </h2>
                    <p className="mt-3 text-sm text-slate-400">ID: {selectedDevice.id}</p>
                  </div>

                  <div className="flex flex-wrap items-center gap-3">
                    {isDirty ? (
                      <span className="rounded-full border border-amber-300/25 bg-amber-400/10 px-4 py-2 text-sm font-semibold text-amber-100">
                        Есть несохраненные изменения
                      </span>
                    ) : (
                      <span className="rounded-full border border-emerald-300/20 bg-emerald-400/10 px-4 py-2 text-sm font-semibold text-emerald-100">
                        Все изменения сохранены
                      </span>
                    )}
                    <button
                      type="button"
                      onClick={handleSave}
                      disabled={isSaving || !isDirty}
                      className={`rounded-2xl px-5 py-3 text-sm font-semibold transition ${
                        isSaving || !isDirty
                          ? "cursor-not-allowed border border-white/10 bg-white/5 text-slate-500"
                          : "border border-cyan-200/30 bg-cyan-400/15 text-cyan-50 hover:border-cyan-100/45 hover:bg-cyan-400/20"
                      }`}
                    >
                      {isSaving ? "Сохраняем..." : "Сохранить"}
                    </button>
                  </div>
                </div>

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
                          className="min-h-[3.25rem] bg-slate-950/40 px-4 py-3 text-sm"
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
                        className="mt-3 w-full rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3 text-sm text-white outline-none transition placeholder:text-slate-500 focus:border-cyan-200/40 focus:bg-slate-950/60"
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
                        className="mt-3 w-full rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3 font-mono text-sm text-white outline-none transition placeholder:text-slate-500 focus:border-cyan-200/40 focus:bg-slate-950/60"
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
          </section>
        </section>
      </section>
    </PageShell>
  );
}
