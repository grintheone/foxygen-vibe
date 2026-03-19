import { useDeferredValue, useRef, useState } from "react";
import { useNavigate, useSearchParams } from "react-router";
import {
  useGetEditorClientsQuery,
  useGetEditorContactByIdQuery,
  useGetEditorContactsQuery,
  usePatchEditorContactMutation,
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
  EditorWorkspace,
  SummaryCard,
  useSyncedSidebarHeight,
} from "./editor-shared";
import { useEditorSearchParamSelection, useLoadedEditorRecord, useUnsavedChangesWarning } from "./editor-hooks";

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

function getContactFormState(contact) {
  return {
    client: contact.client || "",
    email: contact.email || "",
    name: contact.name || "",
    phone: contact.phone || "",
    position: contact.position || "",
  };
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
  const handleSelectContact = useEditorSearchParamSelection({
    isDirty,
    items: contacts,
    paramKey: "contactId",
    selectedId: selectedContactId,
    setSearchParams,
  });

  useLoadedEditorRecord({
    loadedRecordId: loadedContactId,
    onRecordLoad: () =>
      setFeedback({
        message: "",
        tone: "idle",
      }),
    record: selectedContact,
    setFormState,
    setLoadedRecordId: setLoadedContactId,
    toFormState: getContactFormState,
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

      setFormState(getContactFormState(updatedContact));
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
        <EditorPageHeader
          title="Контакты"
          description="Здесь можно редактировать контактные лица и быстро перепривязывать их к клиентам."
          action={<BackButton onClick={handleBack} />}
        />

        <EditorWorkspace
          sidebar={
            <EditorSidebar
              height={sidebarHeight}
              footer={isContactsFetching ? "Обновляем список..." : `Показано ${contacts.length} записей.`}
            >
              <div className="space-y-4">
                <EditorListHeader title="Контактные лица" />
                <EditorSearchField
                  value={searchValue}
                  onChange={(event) => setSearchValue(event.target.value)}
                  placeholder="Имя, телефон, email, клиент"
                />
                <EditorListError error={contactsError} fallbackMessage="Не удалось загрузить список контактов." />
              </div>

              <div className="min-h-0 flex-1 space-y-3 overflow-y-auto pr-1">
                {isContactsLoading ? <EditorNoticeCard message="Загружаем контакты..." /> : null}
                {!isContactsLoading && contacts.length === 0 ? (
                  <EditorNoticeCard message="По текущему запросу ничего не найдено." />
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
            </EditorSidebar>
          }
        >
          <EditorPane editorPaneRef={editorPaneRef}>
            {!selectedContactId ? (
              <EditorNoticeCard dashed message="Выберите контакт слева, чтобы открыть карточку редактора." />
            ) : null}

            {selectedContactId && isContactLoading ? <EditorNoticeCard message="Загружаем карточку контакта..." /> : null}

            {selectedContactId && selectedContactError ? (
              <EditorListError error={selectedContactError} fallbackMessage="Не удалось загрузить карточку контакта." />
            ) : null}

            {selectedContact ? (
              <>
                <EditorRecordHeader
                  id={selectedContact.id}
                  isDirty={isDirty}
                  isSaving={isSaving}
                  onSave={handleSave}
                  title={selectedContact.name?.trim() || "Без имени"}
                  titleLabel="Карточка контакта"
                />

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
                        className={editorFieldClassName}
                      />
                    </EditorFormField>

                    <EditorFormField label="Должность">
                      <input
                        type="text"
                        name="position"
                        value={formState.position}
                        onChange={handleFormChange}
                        placeholder="Например, старшая медсестра"
                        className={editorFieldClassName}
                      />
                    </EditorFormField>

                    <EditorFormField label="Телефон">
                      <input
                        type="text"
                        name="phone"
                        value={formState.phone}
                        onChange={handleFormChange}
                        placeholder="Введите номер телефона"
                        className={editorFieldClassName}
                      />
                    </EditorFormField>

                    <EditorFormField label="Email">
                      <input
                        type="email"
                        name="email"
                        value={formState.email}
                        onChange={handleFormChange}
                        placeholder="Введите email"
                        className={editorFieldClassName}
                      />
                    </EditorFormField>

                    <EditorFormField label="Клиент" hint="Контакт обязательно должен быть привязан к клиенту.">
                      <div className="mt-3">
                        <SelectField
                          name="client"
                          value={formState.client}
                          onChange={handleFormChange}
                          disabled={isClientOptionsLoading}
                          className={editorSelectClassName}
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
          </EditorPane>
        </EditorWorkspace>
      </section>
    </PageShell>
  );
}
