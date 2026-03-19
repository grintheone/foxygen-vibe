import { useDeferredValue, useEffect, useRef, useState } from "react";
import { useNavigate, useSearchParams } from "react-router";
import {
  useGetEditorContactByIdQuery,
  useGetEditorContactsQuery,
  useGetEditorClientsQuery,
  usePatchEditorContactMutation,
} from "../../../shared/api/editor-api";
import { routePaths } from "../../../shared/config/routes";
import { PageShell } from "../../../shared/ui/page-shell";
import { SelectField } from "../../../shared/ui/select-field";
import { StatusMessage } from "../../../shared/ui/status-message";
import { BackButton, EditorFormField, SummaryCard, useSyncedSidebarHeight } from "./editor-shared";

function ContactListItem({ contact, isActive, onClick }) {
  const title = contact.name?.trim() || "Без имени";
  const meta = [contact.position?.trim(), contact.clientName?.trim()].filter(Boolean).join(" • ");

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
      <div className="space-y-2">
        <p className="text-base font-semibold text-white">{title}</p>
        <p className="text-sm text-slate-400">{meta || "Должность и клиент пока не указаны."}</p>
        <p className="text-xs text-slate-500">{contact.phone?.trim() || contact.email?.trim() || "Без контактов"}</p>
      </div>
    </button>
  );
}

export function EditorContactsPage() {
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const editorPaneRef = useRef(null);
  const selectedContactId = searchParams.get("contactId") || "";
  const [searchValue, setSearchValue] = useState("");
  const deferredSearchValue = useDeferredValue(searchValue.trim());
  const [formState, setFormState] = useState({
    client: "",
    email: "",
    name: "",
    phone: "",
    position: "",
  });
  const [loadedContactId, setLoadedContactId] = useState("");
  const [feedback, setFeedback] = useState({
    message: "",
    tone: "idle",
  });
  const [patchEditorContact, { isLoading: isSaving }] = usePatchEditorContactMutation();
  const {
    data: contacts = [],
    error: contactsError,
    isFetching: isContactsFetching,
    isLoading: isContactsLoading,
  } = useGetEditorContactsQuery({
    limit: 50,
    q: deferredSearchValue,
  });
  const {
    data: clientOptions = [],
    error: clientOptionsError,
    isFetching: isClientOptionsFetching,
    isLoading: isClientOptionsLoading,
  } = useGetEditorClientsQuery({
    limit: 100,
    q: "",
  });
  const {
    data: selectedContact,
    error: selectedContactError,
    isFetching: isContactFetching,
    isLoading: isContactLoading,
  } = useGetEditorContactByIdQuery(selectedContactId, {
    skip: !selectedContactId,
  });

  const isDirty =
    Boolean(selectedContactId) &&
    (formState.client !== (selectedContact?.client || "") ||
      formState.email !== (selectedContact?.email || "") ||
      formState.name !== (selectedContact?.name || "") ||
      formState.phone !== (selectedContact?.phone || "") ||
      formState.position !== (selectedContact?.position || ""));
  const selectedClientTitle =
    clientOptions.find((client) => client.id === formState.client)?.title || selectedContact?.clientName?.trim() || "Не выбран";
  const sidebarHeight = useSyncedSidebarHeight(editorPaneRef);

  function handleBack() {
    navigate(routePaths.editor);
  }

  useEffect(() => {
    if (contacts.length === 0 || selectedContactId) {
      return;
    }

    setSearchParams((currentParams) => {
      const nextParams = new URLSearchParams(currentParams);
      nextParams.set("contactId", contacts[0].id);
      return nextParams;
    });
  }, [contacts, selectedContactId, setSearchParams]);

  useEffect(() => {
    if (!selectedContact || selectedContact.id === loadedContactId) {
      return;
    }

    setFormState({
      client: selectedContact.client || "",
      email: selectedContact.email || "",
      name: selectedContact.name || "",
      phone: selectedContact.phone || "",
      position: selectedContact.position || "",
    });
    setLoadedContactId(selectedContact.id);
    setFeedback({
      message: "",
      tone: "idle",
    });
  }, [loadedContactId, selectedContact]);

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

  function handleSelectContact(nextContactId) {
    if (!nextContactId || nextContactId === selectedContactId) {
      return;
    }

    if (isDirty && !window.confirm("У вас есть несохраненные изменения. Перейти к другой записи?")) {
      return;
    }

    setSearchParams((currentParams) => {
      const nextParams = new URLSearchParams(currentParams);
      nextParams.set("contactId", nextContactId);
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
    if (!selectedContactId) {
      return;
    }

    if (!formState.name.trim()) {
      setFeedback({
        message: "Имя контакта обязательно.",
        tone: "error",
      });
      return;
    }

    if (!formState.client) {
      setFeedback({
        message: "Выберите клиента для контакта.",
        tone: "error",
      });
      return;
    }

    try {
      const updatedContact = await patchEditorContact({
        contactId: selectedContactId,
        patch: formState,
      }).unwrap();

      setFormState({
        client: updatedContact.client || "",
        email: updatedContact.email || "",
        name: updatedContact.name || "",
        phone: updatedContact.phone || "",
        position: updatedContact.position || "",
      });
      setLoadedContactId(updatedContact.id);
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
              <h1 className="mt-3 text-3xl font-bold tracking-tight text-white sm:text-4xl">Контакты</h1>
              <p className="mt-3 max-w-2xl text-base text-slate-300">
                Здесь можно редактировать контактные лица и быстро перепривязывать их к клиентам.
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
                <h2 className="mt-3 text-2xl font-bold tracking-tight text-white">Контактные лица</h2>
              </div>

              <label className="block">
                <span className="text-xs font-semibold uppercase tracking-[0.22em] text-slate-400">Поиск</span>
                <input
                  type="search"
                  value={searchValue}
                  onChange={(event) => setSearchValue(event.target.value)}
                  placeholder="Имя, телефон, email, клиент"
                  className="mt-3 w-full rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3 text-sm text-white outline-none transition placeholder:text-slate-500 focus:border-cyan-200/40 focus:bg-slate-950/60"
                />
              </label>

              {contactsError ? (
                <StatusMessage
                  feedback={{
                    message:
                      typeof contactsError?.data === "string"
                        ? contactsError.data
                        : "Не удалось загрузить список контактов.",
                    tone: "error",
                  }}
                />
              ) : null}
            </div>

            <div className="min-h-0 flex-1 space-y-3 overflow-y-auto pr-1">
              {isContactsLoading ? (
                <div className="rounded-3xl border border-white/10 bg-slate-950/25 p-5 text-sm text-slate-300">
                  Загружаем контакты...
                </div>
              ) : null}

              {!isContactsLoading && contacts.length === 0 ? (
                <div className="rounded-3xl border border-white/10 bg-slate-950/25 p-5 text-sm text-slate-300">
                  По текущему запросу ничего не найдено.
                </div>
              ) : null}

              {contacts.map((contact) => (
                <ContactListItem
                  key={contact.id}
                  contact={contact}
                  isActive={contact.id === selectedContactId}
                  onClick={() => handleSelectContact(contact.id)}
                />
              ))}
            </div>

            <p className="self-end text-xs text-slate-500">
              {isContactsFetching ? "Обновляем список..." : `Показано ${contacts.length} записей.`}
            </p>
          </aside>

          <section
            ref={editorPaneRef}
            className="space-y-6 rounded-[2rem] border border-white/10 bg-slate-950/30 p-6 shadow-2xl shadow-black/20 backdrop-blur-xl"
          >
            {!selectedContactId ? (
              <div className="rounded-3xl border border-dashed border-white/15 bg-white/5 p-8 text-slate-300">
                Выберите контакт слева, чтобы открыть карточку редактора.
              </div>
            ) : null}

            {selectedContactId && isContactLoading ? (
              <div className="rounded-3xl border border-white/10 bg-white/5 p-8 text-slate-300">
                Загружаем карточку контакта...
              </div>
            ) : null}

            {selectedContactId && selectedContactError ? (
              <StatusMessage
                feedback={{
                  message:
                    typeof selectedContactError?.data === "string"
                      ? selectedContactError.data
                      : "Не удалось загрузить карточку контакта.",
                  tone: "error",
                }}
              />
            ) : null}

            {selectedContact ? (
              <>
                <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
                  <div>
                    <p className="text-xs font-semibold uppercase tracking-[0.32em] text-cyan-200">Карточка контакта</p>
                    <h2 className="mt-3 text-3xl font-bold tracking-tight text-white">
                      {selectedContact.name?.trim() || "Без имени"}
                    </h2>
                    <p className="mt-3 text-sm text-slate-400">ID: {selectedContact.id}</p>
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
                  <SummaryCard label="Клиент" value={selectedClientTitle} />
                  <SummaryCard label="Телефон" value={formState.phone?.trim() || "Не указан"} />
                  <SummaryCard label="Email" value={formState.email?.trim() || "Не указан"} />
                </div>

                <section className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_320px]">
                  <div className="space-y-5 rounded-3xl border border-white/10 bg-white/5 p-5">
                    <EditorFormField label="Имя">
                      <input
                        type="text"
                        name="name"
                        value={formState.name}
                        onChange={handleFormChange}
                        placeholder="Введите имя контакта"
                        className="mt-3 w-full rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3 text-sm text-white outline-none transition placeholder:text-slate-500 focus:border-cyan-200/40 focus:bg-slate-950/60"
                      />
                    </EditorFormField>

                    <EditorFormField label="Должность">
                      <input
                        type="text"
                        name="position"
                        value={formState.position}
                        onChange={handleFormChange}
                        placeholder="Например, старшая медсестра"
                        className="mt-3 w-full rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3 text-sm text-white outline-none transition placeholder:text-slate-500 focus:border-cyan-200/40 focus:bg-slate-950/60"
                      />
                    </EditorFormField>

                    <EditorFormField label="Телефон">
                      <input
                        type="text"
                        name="phone"
                        value={formState.phone}
                        onChange={handleFormChange}
                        placeholder="Введите номер телефона"
                        className="mt-3 w-full rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3 text-sm text-white outline-none transition placeholder:text-slate-500 focus:border-cyan-200/40 focus:bg-slate-950/60"
                      />
                    </EditorFormField>

                    <EditorFormField label="Email">
                      <input
                        type="email"
                        name="email"
                        value={formState.email}
                        onChange={handleFormChange}
                        placeholder="Введите email"
                        className="mt-3 w-full rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3 text-sm text-white outline-none transition placeholder:text-slate-500 focus:border-cyan-200/40 focus:bg-slate-950/60"
                      />
                    </EditorFormField>

                    <EditorFormField label="Клиент" hint="Контакт обязательно должен быть привязан к клиенту.">
                      <div className="mt-3">
                        <SelectField
                          name="client"
                          value={formState.client}
                          onChange={handleFormChange}
                          disabled={isClientOptionsLoading}
                          className="min-h-[3.25rem] bg-slate-950/40 px-4 py-3 text-sm"
                        >
                          <option value="">Выберите клиента</option>
                          {clientOptions.map((client) => (
                            <option key={client.id} value={client.id}>
                              {client.title || "Без названия"}
                            </option>
                          ))}
                        </SelectField>
                      </div>
                    </EditorFormField>
                  </div>

                  <aside className="space-y-4 rounded-3xl border border-white/10 bg-slate-950/35 p-5">
                    <div>
                      <p className="text-xs font-semibold uppercase tracking-[0.22em] text-slate-400">Связанные поля</p>
                      <div className="mt-4 space-y-3 text-sm text-slate-300">
                        <p>
                          <span className="text-slate-500">Клиент:</span> {selectedClientTitle}
                        </p>
                        <p>
                          <span className="text-slate-500">Телефон:</span> {formState.phone?.trim() || "Не указан"}
                        </p>
                        <p>
                          <span className="text-slate-500">Email:</span> {formState.email?.trim() || "Не указан"}
                        </p>
                      </div>
                    </div>

                    <p className="text-xs text-slate-500">
                      Эта карточка покрывает базовые поля контакта и его привязку к клиенту.
                    </p>
                    {clientOptionsError ? (
                      <p className="text-xs text-rose-300">
                        {typeof clientOptionsError?.data === "string"
                          ? clientOptionsError.data
                          : "Не удалось загрузить список клиентов."}
                      </p>
                    ) : null}
                  </aside>
                </section>

                <p className="text-xs text-slate-500">
                  {isContactFetching
                    ? "Обновляем карточку..."
                    : isClientOptionsFetching
                      ? "Обновляем список клиентов..."
                      : "Изменения применяются к базовым полям контакта и его привязке к клиенту."}
                </p>
              </>
            ) : null}
          </section>
        </section>
      </section>
    </PageShell>
  );
}
