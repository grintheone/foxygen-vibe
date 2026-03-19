import { useEffect, useId, useRef, useState } from "react";

export function AsyncSearchSelect({
  allowClear = true,
  clearLabel = "Очистить выбор",
  disabled = false,
  emptyMessage = "Ничего не найдено.",
  errorMessage = "",
  getOptionDescription,
  getOptionLabel,
  isLoading = false,
  onSearchChange,
  onSelect,
  options = [],
  placeholder = "Выберите значение",
  searchPlaceholder = "Начните вводить для поиска",
  selectedLabel = "",
  value = "",
}) {
  const listboxId = useId();
  const rootRef = useRef(null);
  const searchInputRef = useRef(null);
  const [isOpen, setIsOpen] = useState(false);
  const [searchValue, setSearchValue] = useState("");

  useEffect(() => {
    if (!isOpen) {
      setSearchValue("");
      onSearchChange?.("");
      return;
    }

    onSearchChange?.("");

    const frameId = window.requestAnimationFrame(() => {
      searchInputRef.current?.focus();
    });

    return () => {
      window.cancelAnimationFrame(frameId);
    };
  }, [isOpen, onSearchChange]);

  useEffect(() => {
    if (!isOpen) {
      return;
    }

    onSearchChange?.(searchValue);
  }, [isOpen, onSearchChange, searchValue]);

  useEffect(() => {
    if (!isOpen) {
      return undefined;
    }

    function handlePointerDown(event) {
      if (!rootRef.current?.contains(event.target)) {
        setIsOpen(false);
      }
    }

    function handleKeyDown(event) {
      if (event.key === "Escape") {
        setIsOpen(false);
      }
    }

    document.addEventListener("mousedown", handlePointerDown);
    document.addEventListener("keydown", handleKeyDown);

    return () => {
      document.removeEventListener("mousedown", handlePointerDown);
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [isOpen]);

  function handleToggle() {
    if (disabled) {
      return;
    }

    setIsOpen((currentState) => !currentState);
  }

  function handleSelect(option) {
    onSelect(option);
    setIsOpen(false);
  }

  function handleClear() {
    onSelect(null);
    setIsOpen(false);
  }

  return (
    <div ref={rootRef} className="relative">
      <button
        type="button"
        onClick={handleToggle}
        disabled={disabled}
        aria-expanded={isOpen}
        aria-haspopup="listbox"
        aria-controls={listboxId}
        className="flex min-h-[3.25rem] w-full items-center justify-between gap-3 rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3 text-left text-sm text-slate-100 outline-none transition hover:border-white/20 focus:border-cyan-200/40 focus:bg-slate-950/60 disabled:cursor-not-allowed disabled:opacity-70"
      >
        <span className={value ? "text-white" : "text-slate-500"}>{selectedLabel || placeholder}</span>
        <svg
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
          className={`h-4 w-4 shrink-0 text-slate-400 transition ${isOpen ? "rotate-180" : ""}`}
          aria-hidden="true"
        >
          <path d="m6 9 6 6 6-6" />
        </svg>
      </button>

      {isOpen ? (
        <div className="absolute inset-x-0 top-full z-30 mt-2 rounded-2xl border border-white/10 bg-slate-950/95 p-3 shadow-2xl shadow-black/40 backdrop-blur">
          <input
            ref={searchInputRef}
            type="search"
            value={searchValue}
            onChange={(event) => setSearchValue(event.target.value)}
            placeholder={searchPlaceholder}
            className="w-full rounded-2xl border border-white/10 bg-black/20 px-4 py-3 text-sm text-white outline-none transition placeholder:text-slate-500 focus:border-cyan-200/40 focus:bg-slate-950/60"
          />

          <div id={listboxId} role="listbox" className="mt-3 max-h-64 space-y-2 overflow-y-auto pr-1">
            {allowClear ? (
              <button
                type="button"
                onClick={handleClear}
                className={`block w-full rounded-2xl border px-4 py-3 text-left text-sm transition ${
                  !value
                    ? "border-cyan-200/35 bg-cyan-400/10 text-cyan-100"
                    : "border-white/10 bg-white/5 text-slate-200 hover:border-white/20 hover:bg-white/10"
                }`}
              >
                {clearLabel}
              </button>
            ) : null}

            {errorMessage ? (
              <p className="rounded-2xl border border-rose-300/20 bg-rose-500/10 px-4 py-3 text-sm text-rose-200">
                {errorMessage}
              </p>
            ) : null}

            {!errorMessage && isLoading && options.length === 0 ? (
              <p className="rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-sm text-slate-400">
                Загружаем варианты...
              </p>
            ) : null}

            {!errorMessage && !isLoading && options.length === 0 ? (
              <p className="rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-sm text-slate-400">
                {emptyMessage}
              </p>
            ) : null}

            {options.map((option) => {
              const optionLabel = getOptionLabel(option);
              const optionDescription = getOptionDescription?.(option);
              const isSelected = option.id === value;

              return (
                <button
                  key={option.id}
                  type="button"
                  onClick={() => handleSelect(option)}
                  className={`block w-full rounded-2xl border px-4 py-3 text-left transition ${
                    isSelected
                      ? "border-cyan-200/35 bg-cyan-400/10 text-cyan-50"
                      : "border-white/10 bg-white/5 text-slate-100 hover:border-white/20 hover:bg-white/10"
                  }`}
                >
                  <span className="block text-sm">{optionLabel}</span>
                  {optionDescription ? <span className="mt-1 block text-xs text-slate-400">{optionDescription}</span> : null}
                </button>
              );
            })}
          </div>
        </div>
      ) : null}
    </div>
  );
}
