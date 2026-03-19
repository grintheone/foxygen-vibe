import { useEffect } from "react";

function updateEditorSearchParam(setSearchParams, key, value, options) {
  setSearchParams((currentParams) => {
    const nextParams = new URLSearchParams(currentParams);
    nextParams.set(key, value);
    return nextParams;
  }, options);
}

export function useEditorSearchParamSelection({
  items,
  selectedId,
  paramKey,
  setSearchParams,
  isDirty,
  confirmMessage = "У вас есть несохраненные изменения. Перейти к другой записи?",
}) {
  useEffect(() => {
    if (items.length === 0 || selectedId) {
      return;
    }

    updateEditorSearchParam(setSearchParams, paramKey, items[0].id, { replace: true });
  }, [items, paramKey, selectedId, setSearchParams]);

  function handleSelect(nextId) {
    if (!nextId || nextId === selectedId) {
      return;
    }

    if (isDirty && !window.confirm(confirmMessage)) {
      return;
    }

    updateEditorSearchParam(setSearchParams, paramKey, nextId);
  }

  return handleSelect;
}

export function useLoadedEditorRecord({
  record,
  loadedRecordId,
  setLoadedRecordId,
  setFormState,
  toFormState,
  onRecordLoad,
}) {
  useEffect(() => {
    if (!record || record.id === loadedRecordId) {
      return;
    }

    setFormState(toFormState(record));
    setLoadedRecordId(record.id);
    onRecordLoad?.(record);
  }, [loadedRecordId, onRecordLoad, record, setFormState, setLoadedRecordId, toFormState]);
}

export function useUnsavedChangesWarning(isDirty) {
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
}
