import { useDeferredValue, useEffect, useState } from "react";
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
import { BackButton, EditorFormField, SummaryCard } from "./editor-shared";

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

export function EditorClientsPage() {
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const selectedClientId = searchParams.get("clientId") || "";
  const [searchValue, setSearchValue] = useState("");
  const deferredSearchValue = useDeferredValue(searchValue.trim());
  const [formState, setFormState] = useState({
    address: "",
    location: "{}",
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

  const initialLocationValue = formatEditorJson(selectedClient?.location);
  const isDirty =
    Boolean(selectedClientId) &&
    (formState.title !== (selectedClient?.title || "") ||
      formState.address !== (selectedClient?.address || "") ||
      formState.location !== initialLocationValue ||
      formState.region !== (selectedClient?.region || ""));
  const selectedRegionTitle =
    regions.find((region) => region.id === formState.region)?.title || selectedClient?.regionTitle?.trim() || "Не указан";

  function handleBack() {
    navigate(-1);
  }

  function handleOpenChooser() {
    navigate(routePaths.editor);
  }

  useEffect(() => {
    if (clients.length === 0 || selectedClientId) {
      return;
    }

    setSearchParams((currentParams) => {
      const nextParams = new URLSearchParams(currentParams);
      nextParams.set("clientId", clients[0].id);
      return nextParams;
    });
  }, [clients, selectedClientId, setSearchParams]);

  useEffect(() => {
    if (!selectedClient || selectedClient.id === loadedClientId) {
      return;
    }

    setFormState({
      address: selectedClient.address || "",
      location: formatEditorJson(selectedClient.location),
      region: selectedClient.region || "",
      title: selectedClient.title || "",
    });
    setLoadedClientId(selectedClient.id);
    setFeedback({
      message: "",
      tone: "idle",
    });
  }, [loadedClientId, selectedClient]);

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

  function handleSelectClient(nextClientId) {
    if (!nextClientId || nextClientId === selectedClientId) {
      return;
    }

    if (isDirty && !window.confirm("У вас есть несохраненные изменения. Перейти к другой записи?")) {
      return;
    }

    setSearchParams((currentParams) => {
      const nextParams = new URLSearchParams(currentParams);
      nextParams.set("clientId", nextClientId);
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

    try {
      JSON.parse(formState.location);
    } catch {
      setFeedback({
        message: "Location должен быть валидным JSON.",
        tone: "error",
      });
      return;
    }

    try {
      const updatedClient = await patchEditorClient({
        clientId: selectedClientId,
        patch: {
          address: formState.address,
          location: formState.location,
          region: formState.region,
          title: formState.title,
        },
      }).unwrap();

      setFormState({
        address: updatedClient.address || "",
        location: formatEditorJson(updatedClient.location),
        region: updatedClient.region || "",
        title: updatedClient.title || "",
      });
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
        <header className="rounded-3xl border border-white/10 bg-slate-950/35 p-6 shadow-xl shadow-black/20 backdrop-blur">
          <div className="flex flex-wrap items-start justify-between gap-4">
            <div>
              <p className="text-xs font-semibold uppercase tracking-[0.3em] text-slate-400">Редактор</p>
              <h1 className="mt-3 text-3xl font-bold tracking-tight text-white sm:text-4xl">Клиенты</h1>
              <p className="mt-3 max-w-2xl text-base text-slate-300">
                Здесь можно редактировать клиентскую карточку, регион и JSON-поля без прямого доступа к базе.
              </p>
            </div>
            <div className="flex items-center gap-3">
              <button
                type="button"
                onClick={handleOpenChooser}
                className="rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-sm font-semibold text-slate-100 transition hover:border-white/20 hover:bg-white/10"
              >
                Сущности
              </button>
              <BackButton onClick={handleBack} />
            </div>
          </div>
        </header>

        <section className="grid gap-6 xl:h-[calc(100vh-18rem)] xl:grid-cols-[360px_minmax(0,1fr)]">
          <aside className="flex h-full min-h-0 flex-col space-y-4 overflow-hidden rounded-[2rem] border border-white/10 bg-white/10 p-5 shadow-2xl shadow-[#6A3BF2]/15 backdrop-blur-xl">
            <div>
              <p className="text-xs font-semibold uppercase tracking-[0.32em] text-cyan-200">Список</p>
              <h2 className="mt-3 text-2xl font-bold tracking-tight text-white">Клиентские записи</h2>
            </div>

            <label className="block">
              <span className="text-xs font-semibold uppercase tracking-[0.22em] text-slate-400">Поиск</span>
              <input
                type="search"
                value={searchValue}
                onChange={(event) => setSearchValue(event.target.value)}
                placeholder="Название или адрес"
                className="mt-3 w-full rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3 text-sm text-white outline-none transition placeholder:text-slate-500 focus:border-cyan-200/40 focus:bg-slate-950/60"
              />
            </label>

            {clientsError ? (
              <StatusMessage
                feedback={{
                  message:
                    typeof clientsError?.data === "string"
                      ? clientsError.data
                      : "Не удалось загрузить список клиентов.",
                  tone: "error",
                }}
              />
            ) : null}

            <div className="min-h-0 flex-1 space-y-3 overflow-y-auto pr-1">
              {isClientsLoading ? (
                <div className="rounded-3xl border border-white/10 bg-slate-950/25 p-5 text-sm text-slate-300">
                  Загружаем клиентов...
                </div>
              ) : null}

              {!isClientsLoading && clients.length === 0 ? (
                <div className="rounded-3xl border border-white/10 bg-slate-950/25 p-5 text-sm text-slate-300">
                  По текущему запросу ничего не найдено.
                </div>
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

            <p className="text-xs text-slate-500">
              {isClientsFetching ? "Обновляем список..." : `Показано ${clients.length} записей.`}
            </p>
          </aside>

          <section className="flex h-full min-h-0 flex-col space-y-6 overflow-y-auto rounded-[2rem] border border-white/10 bg-slate-950/30 p-6 shadow-2xl shadow-black/20 backdrop-blur-xl">
            {!selectedClientId ? (
              <div className="rounded-3xl border border-dashed border-white/15 bg-white/5 p-8 text-slate-300">
                Выберите клиента слева, чтобы открыть карточку редактора.
              </div>
            ) : null}

            {selectedClientId && isClientLoading ? (
              <div className="rounded-3xl border border-white/10 bg-white/5 p-8 text-slate-300">
                Загружаем карточку клиента...
              </div>
            ) : null}

            {selectedClientId && selectedClientError ? (
              <StatusMessage
                feedback={{
                  message:
                    typeof selectedClientError?.data === "string"
                      ? selectedClientError.data
                      : "Не удалось загрузить карточку клиента.",
                  tone: "error",
                }}
              />
            ) : null}

            {selectedClient ? (
              <>
                <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
                  <div>
                    <p className="text-xs font-semibold uppercase tracking-[0.32em] text-cyan-200">Карточка клиента</p>
                    <h2 className="mt-3 text-3xl font-bold tracking-tight text-white">
                      {selectedClient.title?.trim() || "Без названия"}
                    </h2>
                    <p className="mt-3 text-sm text-slate-400">ID: {selectedClient.id}</p>
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
                  <SummaryCard label="Контакты" value={selectedClient.contactCount} />
                  <SummaryCard label="Активные договоры" value={selectedClient.activeAgreementCount} />
                  <SummaryCard label="Регион" value={selectedRegionTitle} />
                </div>

                <section className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_320px]">
                  <div className="space-y-5 rounded-3xl border border-white/10 bg-white/5 p-5">
                    <EditorFormField label="Название">
                      <input
                        type="text"
                        name="title"
                        value={formState.title}
                        onChange={handleFormChange}
                        placeholder="Введите название клиента"
                        className="mt-3 w-full rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3 text-sm text-white outline-none transition placeholder:text-slate-500 focus:border-cyan-200/40 focus:bg-slate-950/60"
                      />
                    </EditorFormField>

                    <EditorFormField label="Адрес" hint="Можно оставить пустым, если адрес еще не заполнен.">
                      <textarea
                        name="address"
                        value={formState.address}
                        onChange={handleFormChange}
                        rows="5"
                        placeholder="Укажите адрес клиента"
                        className="mt-3 w-full rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3 text-sm text-white outline-none transition placeholder:text-slate-500 focus:border-cyan-200/40 focus:bg-slate-950/60"
                      />
                    </EditorFormField>

                    <EditorFormField label="Регион" hint="Выберите регион из существующего справочника.">
                      <div className="mt-3">
                        <SelectField
                          name="region"
                          value={formState.region}
                          onChange={handleFormChange}
                          disabled={isRegionsLoading}
                          className="min-h-[3.25rem] bg-slate-950/40 px-4 py-3 text-sm"
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
                      label="Location JSON"
                      hint="Поддерживается любой валидный JSON. Пустой объект можно хранить как `{}`."
                    >
                      <textarea
                        name="location"
                        value={formState.location}
                        onChange={handleFormChange}
                        rows="10"
                        spellCheck={false}
                        className="mt-3 w-full rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3 font-mono text-sm text-white outline-none transition focus:border-cyan-200/40 focus:bg-slate-950/60"
                      />
                    </EditorFormField>
                  </div>

                  <aside className="space-y-4 rounded-3xl border border-white/10 bg-slate-950/35 p-5">
                    <div>
                      <p className="text-xs font-semibold uppercase tracking-[0.22em] text-slate-400">Связанные поля</p>
                      <div className="mt-4 space-y-3 text-sm text-slate-300">
                        <p>
                          <span className="text-slate-500">Регион:</span> {selectedRegionTitle}
                        </p>
                        <p>
                          <span className="text-slate-500">ЛИС:</span> {selectedClient.laboratorySystem || "Не привязана"}
                        </p>
                        <p>
                          <span className="text-slate-500">Менеджеров:</span> {selectedClient.manager?.length || 0}
                        </p>
                      </div>
                    </div>

                    <div>
                      <p className="text-xs font-semibold uppercase tracking-[0.22em] text-slate-400">Location JSON</p>
                      <pre className="mt-4 overflow-x-auto rounded-2xl border border-white/10 bg-black/20 p-4 text-xs text-slate-300">
                        {formState.location}
                      </pre>
                    </div>

                    <p className="text-xs text-slate-500">
                      ЛИС и менеджеры пока доступны только для просмотра. Region и location уже можно редактировать.
                    </p>
                    {regionsError ? (
                      <p className="text-xs text-rose-300">
                        {typeof regionsError?.data === "string" ? regionsError.data : "Не удалось загрузить список регионов."}
                      </p>
                    ) : null}
                  </aside>
                </section>

                <p className="text-xs text-slate-500">
                  {isClientFetching
                    ? "Обновляем карточку..."
                    : isRegionsFetching
                      ? "Обновляем список регионов..."
                      : "Изменения применяются к базовым полям клиента, региону и JSON-полю location."}
                </p>
              </>
            ) : null}
          </section>
        </section>
      </section>
    </PageShell>
  );
}
