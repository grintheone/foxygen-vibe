import { useDeferredValue, useRef, useState } from "react";
import { useNavigate, useSearchParams } from "react-router";
import {
  useGetEditorClassificatorByIdQuery,
  useGetEditorClassificatorsQuery,
  useGetEditorManufacturersQuery,
  useGetEditorResearchTypesQuery,
  usePatchEditorClassificatorMutation,
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
  editorTextareaClassName,
  EditorSidebar,
  EditorWorkspace,
  useSyncedSidebarHeight,
} from "./editor-shared";
import { useEditorSearchParamSelection, useLoadedEditorRecord, useUnsavedChangesWarning } from "./editor-hooks";

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

function normalizeEditorLines(value) {
  return value
    .split("\n")
    .map((item) => item.trim())
    .filter(Boolean);
}

function toEditorLines(value) {
  return Array.isArray(value) ? value.join("\n") : "";
}

function ClassificatorListItem({ classificator, isActive, onClick }) {
  const title = classificator.title?.trim() || "Без названия";
  const meta = [classificator.manufacturerTitle?.trim(), classificator.researchTypeTitle?.trim()].filter(Boolean).join(" • ");

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
          <p className="text-sm text-slate-400">{meta || "Производитель и тип исследования пока не указаны."}</p>
        </div>
        <div className="shrink-0 text-right text-xs text-slate-400">
          <p>{classificator.deviceCount} устройств</p>
          <p className="mt-1">
            {classificator.attachmentCount} влож. • {classificator.imageCount} изобр.
          </p>
        </div>
      </div>
    </button>
  );
}

function getClassificatorFormState(classificator) {
  return {
    attachments: toEditorLines(classificator.attachments),
    images: toEditorLines(classificator.images),
    maintenanceRegulations: formatEditorJson(classificator.maintenanceRegulations),
    manufacturer: classificator.manufacturer || "",
    registrationCertificate: formatEditorJson(classificator.registrationCertificate),
    researchType: classificator.researchType || "",
    title: classificator.title || "",
  };
}

export function EditorClassificatorsPage() {
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const editorPaneRef = useRef(null);
  const selectedClassificatorId = searchParams.get("classificatorId") || "";
  const [searchValue, setSearchValue] = useState("");
  const deferredSearchValue = useDeferredValue(searchValue.trim());
  const [formState, setFormState] = useState({
    attachments: "",
    images: "",
    maintenanceRegulations: "{}",
    manufacturer: "",
    registrationCertificate: "{}",
    researchType: "",
    title: "",
  });
  const [loadedClassificatorId, setLoadedClassificatorId] = useState("");
  const [feedback, setFeedback] = useState({
    message: "",
    tone: "idle",
  });
  const [patchEditorClassificator, { isLoading: isSaving }] = usePatchEditorClassificatorMutation();
  const {
    data: classificators = [],
    error: classificatorsError,
    isFetching: isClassificatorsFetching,
    isLoading: isClassificatorsLoading,
  } = useGetEditorClassificatorsQuery({
    limit: 50,
    q: deferredSearchValue,
  });
  const {
    data: manufacturers = [],
    error: manufacturersError,
    isFetching: isManufacturersFetching,
    isLoading: isManufacturersLoading,
  } = useGetEditorManufacturersQuery();
  const {
    data: researchTypes = [],
    error: researchTypesError,
    isFetching: isResearchTypesFetching,
    isLoading: isResearchTypesLoading,
  } = useGetEditorResearchTypesQuery();
  const {
    data: selectedClassificator,
    error: selectedClassificatorError,
    isFetching: isClassificatorFetching,
    isLoading: isClassificatorLoading,
  } = useGetEditorClassificatorByIdQuery(selectedClassificatorId, {
    skip: !selectedClassificatorId,
  });

  const initialRegistrationCertificate = formatEditorJson(selectedClassificator?.registrationCertificate);
  const initialMaintenanceRegulations = formatEditorJson(selectedClassificator?.maintenanceRegulations);
  const initialAttachments = toEditorLines(selectedClassificator?.attachments);
  const initialImages = toEditorLines(selectedClassificator?.images);
  const isDirty =
    Boolean(selectedClassificatorId) &&
    (formState.title !== (selectedClassificator?.title || "") ||
      formState.manufacturer !== (selectedClassificator?.manufacturer || "") ||
      formState.researchType !== (selectedClassificator?.researchType || "") ||
      formState.registrationCertificate !== initialRegistrationCertificate ||
      formState.maintenanceRegulations !== initialMaintenanceRegulations ||
      formState.attachments !== initialAttachments ||
      formState.images !== initialImages);
  const selectedManufacturerTitle =
    manufacturers.find((manufacturer) => manufacturer.id === formState.manufacturer)?.title ||
    (formState.manufacturer === selectedClassificator?.manufacturer ? selectedClassificator?.manufacturerTitle?.trim() : "") ||
    "Не указан";
  const selectedResearchTypeTitle =
    researchTypes.find((researchType) => researchType.id === formState.researchType)?.title ||
    (formState.researchType === selectedClassificator?.researchType ? selectedClassificator?.researchTypeTitle?.trim() : "") ||
    "Не указан";
  const attachmentItems = normalizeEditorLines(formState.attachments);
  const imageItems = normalizeEditorLines(formState.images);
  const sidebarHeight = useSyncedSidebarHeight(editorPaneRef);
  const handleSelectClassificator = useEditorSearchParamSelection({
    isDirty,
    items: classificators,
    paramKey: "classificatorId",
    selectedId: selectedClassificatorId,
    setSearchParams,
  });

  useLoadedEditorRecord({
    loadedRecordId: loadedClassificatorId,
    onRecordLoad: () =>
      setFeedback({
        message: "",
        tone: "idle",
      }),
    record: selectedClassificator,
    setFormState,
    setLoadedRecordId: setLoadedClassificatorId,
    toFormState: getClassificatorFormState,
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
    if (!selectedClassificatorId) {
      return;
    }

    if (!formState.title.trim()) {
      setFeedback({
        message: "Название классификатора обязательно.",
        tone: "error",
      });
      return;
    }

    try {
      JSON.parse(formState.registrationCertificate || "{}");
    } catch {
      setFeedback({
        message: "Регистрационное удостоверение должно содержать валидный JSON.",
        tone: "error",
      });
      return;
    }

    try {
      JSON.parse(formState.maintenanceRegulations || "{}");
    } catch {
      setFeedback({
        message: "Регламент обслуживания должен содержать валидный JSON.",
        tone: "error",
      });
      return;
    }

    try {
      const updatedClassificator = await patchEditorClassificator({
        classificatorId: selectedClassificatorId,
        patch: {
          attachments: attachmentItems,
          images: imageItems,
          maintenanceRegulations: formState.maintenanceRegulations,
          manufacturer: formState.manufacturer,
          registrationCertificate: formState.registrationCertificate,
          researchType: formState.researchType,
          title: formState.title,
        },
      }).unwrap();

      setFormState(getClassificatorFormState(updatedClassificator));
      setLoadedClassificatorId(updatedClassificator.id);
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
          title="Классификаторы"
          leadingAction={<BackButton onClick={handleBack} />}
        />

        <EditorWorkspace
          sidebar={
            <EditorSidebar
              height={sidebarHeight}
              footer={isClassificatorsFetching ? "Обновляем список..." : `Показано ${classificators.length} записей.`}
            >
              <div className="space-y-4">
                <EditorListHeader title="Карточки классификаторов" />
                <EditorSearchField
                  value={searchValue}
                  onChange={(event) => setSearchValue(event.target.value)}
                  placeholder="Название, производитель, тип исследования"
                />
                <EditorListError error={classificatorsError} fallbackMessage="Не удалось загрузить список классификаторов." />
              </div>

              <div className="min-h-0 flex-1 space-y-3 overflow-y-auto pr-1">
                {isClassificatorsLoading ? <EditorNoticeCard message="Загружаем классификаторы..." /> : null}
                {!isClassificatorsLoading && classificators.length === 0 ? (
                  <EditorNoticeCard message="По текущему запросу ничего не найдено." />
                ) : null}

                {classificators.map((classificator) => (
                  <ClassificatorListItem
                    key={classificator.id}
                    classificator={classificator}
                    isActive={classificator.id === selectedClassificatorId}
                    onClick={() => handleSelectClassificator(classificator.id)}
                  />
                ))}
              </div>
            </EditorSidebar>
          }
        >
          <EditorPane editorPaneRef={editorPaneRef}>
            {!selectedClassificatorId ? (
              <EditorNoticeCard dashed message="Выберите классификатор слева, чтобы открыть карточку редактора." />
            ) : null}

            {selectedClassificatorId && isClassificatorLoading ? (
              <EditorNoticeCard message="Загружаем карточку классификатора..." />
            ) : null}

            {selectedClassificatorId && selectedClassificatorError ? (
              <EditorListError error={selectedClassificatorError} fallbackMessage="Не удалось загрузить карточку классификатора." />
            ) : null}

            {selectedClassificator ? (
              <>
                <EditorRecordHeader
                  id={selectedClassificator.id}
                  isDirty={isDirty}
                  isSaving={isSaving}
                  onSave={handleSave}
                  title={selectedClassificator.title?.trim() || "Без названия"}
                  titleLabel="Карточка классификатора"
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
                        placeholder="Введите название классификатора"
                        className={editorFieldClassName}
                      />
                    </EditorFormField>

                    <EditorFormField label="Производитель">
                      <div className="mt-3">
                        <SelectField
                          name="manufacturer"
                          value={formState.manufacturer}
                          onChange={handleFormChange}
                          disabled={isManufacturersLoading}
                          className={editorSelectClassName}
                        >
                          <option value="">Не указан</option>
                          {manufacturers.map((manufacturer) => (
                            <option key={manufacturer.id} value={manufacturer.id}>
                              {manufacturer.title || "Без названия"}
                            </option>
                          ))}
                        </SelectField>
                      </div>
                    </EditorFormField>

                    <EditorFormField label="Тип исследования">
                      <div className="mt-3">
                        <SelectField
                          name="researchType"
                          value={formState.researchType}
                          onChange={handleFormChange}
                          disabled={isResearchTypesLoading}
                          className={editorSelectClassName}
                        >
                          <option value="">Не указан</option>
                          {researchTypes.map((researchType) => (
                            <option key={researchType.id} value={researchType.id}>
                              {researchType.title || "Без названия"}
                            </option>
                          ))}
                        </SelectField>
                      </div>
                    </EditorFormField>

                    <EditorFormField
                      label="Регистрационное удостоверение"
                      hint="Поле хранится как JSONB. Можно оставить `{}` или передать объект с реквизитами."
                    >
                      <textarea
                        name="registrationCertificate"
                        value={formState.registrationCertificate}
                        onChange={handleFormChange}
                        rows="10"
                        spellCheck={false}
                        className={editorTextareaClassName}
                      />
                    </EditorFormField>

                    <EditorFormField
                      label="Регламент обслуживания"
                      hint="Поле хранится как JSONB и подходит для структурированных правил обслуживания."
                    >
                      <textarea
                        name="maintenanceRegulations"
                        value={formState.maintenanceRegulations}
                        onChange={handleFormChange}
                        rows="10"
                        spellCheck={false}
                        className={editorTextareaClassName}
                      />
                    </EditorFormField>

                    <EditorFormField
                      label="Вложения"
                      hint="По одной строке на элемент. Сохраняется как массив строк."
                    >
                      <textarea
                        name="attachments"
                        value={formState.attachments}
                        onChange={handleFormChange}
                        rows="6"
                        spellCheck={false}
                        placeholder="manual.pdf&#10;specification.docx"
                        className={editorTextareaClassName}
                      />
                    </EditorFormField>

                    <EditorFormField
                      label="Изображения"
                      hint="По одной строке на элемент. Сохраняется как массив строк."
                    >
                      <textarea
                        name="images"
                        value={formState.images}
                        onChange={handleFormChange}
                        rows="6"
                        spellCheck={false}
                        placeholder="photo-1.jpg&#10;photo-2.jpg"
                        className={editorTextareaClassName}
                      />
                    </EditorFormField>
                  </div>

                  <EditorContextPanel
                    title="Контекст классификатора"
                    footer={
                      <>
                        {manufacturersError ? (
                          <p className="text-xs text-rose-300">
                            {typeof manufacturersError?.data === "string"
                              ? manufacturersError.data
                              : "Не удалось загрузить список производителей."}
                          </p>
                        ) : null}
                        {researchTypesError ? (
                          <p className="text-xs text-rose-300">
                            {typeof researchTypesError?.data === "string"
                              ? researchTypesError.data
                              : "Не удалось загрузить список типов исследования."}
                          </p>
                        ) : null}
                      </>
                    }
                  >
                    <EditorContextSection title="Основное">
                      <EditorContextItem label="Производитель" value={selectedManufacturerTitle} />
                      <EditorContextItem label="Тип исследования" value={selectedResearchTypeTitle} />
                      <EditorContextItem label="Связанных устройств" value={selectedClassificator.deviceCount} />
                      <EditorContextItem label="Вложения" value={attachmentItems.length} />
                      <EditorContextItem label="Изображения" value={imageItems.length} />
                    </EditorContextSection>

                    <EditorContextSection title="JSON превью">
                      <pre className="overflow-x-auto rounded-2xl border border-white/10 bg-black/20 p-4 text-xs text-slate-300">
                        {formState.registrationCertificate}
                      </pre>
                    </EditorContextSection>
                  </EditorContextPanel>
                </section>

                <p className="text-xs text-slate-500">
                  {isClassificatorFetching
                    ? "Обновляем карточку..."
                    : isManufacturersFetching
                      ? "Обновляем список производителей..."
                      : isResearchTypesFetching
                        ? "Обновляем список типов исследования..."
                        : "Изменения применяются к полям классификатора, его JSON-документам и массивам вложений."}
                </p>
              </>
            ) : null}
          </EditorPane>
        </EditorWorkspace>
      </section>
    </PageShell>
  );
}
