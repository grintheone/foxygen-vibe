import { useDeferredValue, useRef, useState } from "react";
import { useNavigate, useSearchParams } from "react-router";
import {
  useGetEditorClientByIdQuery,
  useGetEditorClientsQuery,
  useGetEditorRegionsQuery,
  usePatchEditorClientMutation,
} from "../../../shared/api/editor-api";
import { routePaths } from "../../../shared/config/routes";
import { PageShell } from "../../../shared/ui/page-shell";
import { SelectField } from "../../../shared/ui/select-field";
import { StatusMessage } from "../../../shared/ui/status-message";
import {
  BackButton,
  EditorContextItem,
  EditorContextPanel,
  EditorContextSection,
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
  useSyncedSidebarHeight,
} from "./editor-shared";
import { useEditorSearchParamSelection, useLoadedEditorRecord, useUnsavedChangesWarning } from "./editor-hooks";

function ClientListItem({ client, isActive, onClick }) {
  const title = client.title?.trim() || "Без названия";
  const meta = [client.regionTitle?.trim(), client.address?.trim()].filter(Boolean).join(" • ");

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
        <div>
          <p className="text-base font-semibold text-white">{title}</p>
          <p className="mt-2 text-sm text-slate-400">{meta || "Адрес и регион пока не указаны."}</p>
        </div>
        <div className="shrink-0 text-right text-xs text-slate-400">
          <p>{client.contactCount} контактов</p>
          <p className="mt-1">{client.activeAgreementCount} активных договоров</p>
        </div>
      </div>
    </button>
  );
}

function resolveLocationFields(location) {
  const items = Array.isArray(location) ? location : location ? [location] : [];

  for (const item of items) {
    const latitude = Number(item?.lat ?? item?.latitude);
    const longitude = Number(item?.lng ?? item?.lon ?? item?.longitude);

    if (Number.isFinite(latitude) && Number.isFinite(longitude)) {
      return {
        latitude: String(latitude),
        longitude: String(longitude),
      };
    }
  }

  return {
    latitude: "",
    longitude: "",
  };
}

function buildLocationJson(latitude, longitude) {
  const normalizedLatitude = latitude.trim();
  const normalizedLongitude = longitude.trim();

  if (!normalizedLatitude && !normalizedLongitude) {
    return "{}";
  }

  const parsedLatitude = Number(normalizedLatitude);
  const parsedLongitude = Number(normalizedLongitude);

  if (!Number.isFinite(parsedLatitude) || !Number.isFinite(parsedLongitude)) {
    return null;
  }

  return JSON.stringify(
    {
      lat: parsedLatitude,
      lng: parsedLongitude,
    },
    null,
    2,
  );
}

function getClientFormState(client) {
  return {
    address: client.address || "",
    ...resolveLocationFields(client.location),
    region: client.region || "",
    title: client.title || "",
  };
}

export function EditorClientsPage() {
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const editorPaneRef = useRef(null);
  const selectedClientId = searchParams.get("clientId") || "";
  const [searchValue, setSearchValue] = useState("");
  const deferredSearchValue = useDeferredValue(searchValue.trim());
  const [formState, setFormState] = useState({
    address: "",
    latitude: "",
    longitude: "",
    region: "",
    title: "",
  });
  const [loadedClientId, setLoadedClientId] = useState("");
  const [feedback, setFeedback] = useState({
    message: "",
    tone: "idle",
  });
  const [patchEditorClient, { isLoading: isSaving }] = usePatchEditorClientMutation();
  const {
    data: clients = [],
    error: clientsError,
    isFetching: isClientsFetching,
    isLoading: isClientsLoading,
  } = useGetEditorClientsQuery({
    limit: 50,
    q: deferredSearchValue,
  });
  const {
    data: regions = [],
    error: regionsError,
    isFetching: isRegionsFetching,
    isLoading: isRegionsLoading,
  } = useGetEditorRegionsQuery();
  const {
    data: selectedClient,
    error: selectedClientError,
    isFetching: isClientFetching,
    isLoading: isClientLoading,
  } = useGetEditorClientByIdQuery(selectedClientId, {
    skip: !selectedClientId,
  });

  const initialLocationFields = resolveLocationFields(selectedClient?.location);
  const isDirty =
    Boolean(selectedClientId) &&
    (formState.title !== (selectedClient?.title || "") ||
      formState.address !== (selectedClient?.address || "") ||
      formState.latitude !== initialLocationFields.latitude ||
      formState.longitude !== initialLocationFields.longitude ||
      formState.region !== (selectedClient?.region || ""));
  const selectedRegionTitle =
    regions.find((region) => region.id === formState.region)?.title || selectedClient?.regionTitle?.trim() || "Не указан";
  const sidebarHeight = useSyncedSidebarHeight(editorPaneRef);
  const locationPreview = buildLocationJson(formState.latitude, formState.longitude) || "Некорректные координаты";
  const handleSelectClient = useEditorSearchParamSelection({
    isDirty,
    items: clients,
    paramKey: "clientId",
    selectedId: selectedClientId,
    setSearchParams,
  });

  useLoadedEditorRecord({
    loadedRecordId: loadedClientId,
    onRecordLoad: () =>
      setFeedback({
        message: "",
        tone: "idle",
      }),
    record: selectedClient,
    setFormState,
    setLoadedRecordId: setLoadedClientId,
    toFormState: getClientFormState,
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

  async function handleSave() {
    if (!selectedClientId) {
      return;
    }

    if (!formState.title.trim()) {
      setFeedback({
        message: "Название клиента обязательно.",
        tone: "error",
      });
      return;
    }

    const locationJson = buildLocationJson(formState.latitude, formState.longitude);
    if (locationJson === null) {
      setFeedback({
        message: "Широта и долгота должны быть числами.",
        tone: "error",
      });
      return;
    }

    try {
      const updatedClient = await patchEditorClient({
        clientId: selectedClientId,
        patch: {
          address: formState.address,
          location: locationJson,
          region: formState.region,
          title: formState.title,
        },
      }).unwrap();

      setFormState(getClientFormState(updatedClient));
      setLoadedClientId(updatedClient.id);
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
          title="Клиенты"
          leadingAction={<BackButton onClick={handleBack} />}
        />

        <EditorWorkspace
          sidebar={
            <EditorSidebar
              height={sidebarHeight}
              footer={isClientsFetching ? "Обновляем список..." : `Показано ${clients.length} записей.`}
            >
              <div className="space-y-4">
                <EditorListHeader title="Клиентские записи" />
                <EditorSearchField
                  value={searchValue}
                  onChange={(event) => setSearchValue(event.target.value)}
                  placeholder="Название или адрес"
                />
                <EditorListError error={clientsError} fallbackMessage="Не удалось загрузить список клиентов." />
              </div>

              <div className="min-h-0 flex-1 space-y-3 overflow-y-auto pr-1">
                {isClientsLoading ? <EditorNoticeCard message="Загружаем клиентов..." /> : null}
                {!isClientsLoading && clients.length === 0 ? (
                  <EditorNoticeCard message="По текущему запросу ничего не найдено." />
                ) : null}

                {clients.map((client) => (
                  <ClientListItem
                    key={client.id}
                    client={client}
                    isActive={client.id === selectedClientId}
                    onClick={() => handleSelectClient(client.id)}
                  />
                ))}
              </div>
            </EditorSidebar>
          }
        >
          <EditorPane editorPaneRef={editorPaneRef}>
            {!selectedClientId ? (
              <EditorNoticeCard dashed message="Выберите клиента слева, чтобы открыть карточку редактора." />
            ) : null}

            {selectedClientId && isClientLoading ? <EditorNoticeCard message="Загружаем карточку клиента..." /> : null}

            {selectedClientId && selectedClientError ? (
              <EditorListError error={selectedClientError} fallbackMessage="Не удалось загрузить карточку клиента." />
            ) : null}

            {selectedClient ? (
              <>
                <EditorRecordHeader
                  id={selectedClient.id}
                  isDirty={isDirty}
                  isSaving={isSaving}
                  onSave={handleSave}
                  title={selectedClient.title?.trim() || "Без названия"}
                  titleLabel="Карточка клиента"
                />

                {feedback.message ? <StatusMessage feedback={feedback} /> : null}

                <section className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_320px]">
                  <div className="space-y-5 rounded-3xl border border-white/10 bg-white/5 p-5">
                    <EditorFormField label="Название">
                      <input
                        type="text"
                        name="title"
                        value={formState.title}
                        onChange={handleFormChange}
                        placeholder="Введите название клиента"
                        className={editorFieldClassName}
                      />
                    </EditorFormField>

                    <EditorFormField label="Адрес" hint="Можно оставить пустым, если адрес еще не заполнен.">
                      <textarea
                        name="address"
                        value={formState.address}
                        onChange={handleFormChange}
                        rows="5"
                        placeholder="Укажите адрес клиента"
                        className={editorTextareaClassName}
                      />
                    </EditorFormField>

                    <EditorFormField label="Регион" hint="Выберите регион из существующего справочника.">
                      <div className="mt-3">
                        <SelectField
                          name="region"
                          value={formState.region}
                          onChange={handleFormChange}
                          disabled={isRegionsLoading}
                          className={editorSelectClassName}
                        >
                          <option value="">Не указан</option>
                          {regions.map((region) => (
                            <option key={region.id} value={region.id}>
                              {region.title || "Без названия"}
                            </option>
                          ))}
                        </SelectField>
                      </div>
                    </EditorFormField>

                    <EditorFormField
                      label="Широта"
                      hint="Введите числовое значение latitude. Например: `55.7558`."
                    >
                      <input
                        type="number"
                        inputMode="decimal"
                        step="any"
                        name="latitude"
                        value={formState.latitude}
                        onChange={handleFormChange}
                        placeholder="Например, 55.7558"
                        className={editorFieldClassName}
                      />
                    </EditorFormField>

                    <EditorFormField
                      label="Долгота"
                      hint="Введите числовое значение longitude. Например: `37.6176`."
                    >
                      <input
                        type="number"
                        inputMode="decimal"
                        step="any"
                        name="longitude"
                        value={formState.longitude}
                        onChange={handleFormChange}
                        placeholder="Например, 37.6176"
                        className={editorFieldClassName}
                      />
                    </EditorFormField>
                  </div>

                  <EditorContextPanel
                    title="Контекст клиента"
                    footer={
                      <>
                        <p className="text-xs text-slate-500">
                          ЛИС и менеджеры пока доступны только для просмотра. Location собирается автоматически из широты
                          и долготы.
                        </p>
                        {regionsError ? (
                          <p className="text-xs text-rose-300">
                            {typeof regionsError?.data === "string"
                              ? regionsError.data
                              : "Не удалось загрузить список регионов."}
                          </p>
                        ) : null}
                      </>
                    }
                  >
                    <EditorContextSection title="Основное">
                      <EditorContextItem label="Контакты" value={selectedClient.contactCount} />
                      <EditorContextItem label="Активные договоры" value={selectedClient.activeAgreementCount} />
                      <EditorContextItem label="Регион" value={selectedRegionTitle} />
                      <EditorContextItem label="ЛИС" value={selectedClient.laboratorySystem || "Не привязана"} />
                      <EditorContextItem label="Менеджеров" value={selectedClient.manager?.length || 0} />
                    </EditorContextSection>

                    <EditorContextSection title="Координаты">
                      <EditorContextItem label="Широта" value={formState.latitude || "Не указана"} />
                      <EditorContextItem label="Долгота" value={formState.longitude || "Не указана"} />
                    </EditorContextSection>

                    <EditorContextSection title="Location JSON">
                      <pre className="overflow-x-auto rounded-2xl border border-white/10 bg-black/20 p-4 text-xs text-slate-300">
                        {locationPreview}
                      </pre>
                    </EditorContextSection>
                  </EditorContextPanel>
                </section>

                <p className="text-xs text-slate-500">
                  {isClientFetching
                    ? "Обновляем карточку..."
                    : isRegionsFetching
                      ? "Обновляем список регионов..."
                      : "Изменения применяются к базовым полям клиента, региону и координатам location."}
                </p>
              </>
            ) : null}
          </EditorPane>
        </EditorWorkspace>
      </section>
    </PageShell>
  );
}
