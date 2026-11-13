import { useEffect, useRef, useState } from "react";
import { Link, useLocation } from "react-router";
import { useSearchArchivesQuery } from "../api/tickets-api";
import { routePaths } from "../config/routes";

function SearchIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="h-5 w-5" aria-hidden="true">
      <circle cx="10.5" cy="10.5" r="6.5" />
      <path d="m16 16 4.5 4.5" />
    </svg>
  );
}

export function ArchiveSearch() {
  const dialogRef = useRef(null);
  const triggerRef = useRef(null);
  const inputRef = useRef(null);
  const [isOpen, setIsOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [debouncedQuery, setDebouncedQuery] = useState("");
  const location = useLocation();
  const normalizedQuery = query.trim();
  const isValid = Array.from(normalizedQuery).length >= 2;
  const { currentData, isFetching, isError, refetch } = useSearchArchivesQuery(debouncedQuery, {
    skip: !isOpen || Array.from(debouncedQuery).length < 2,
  });
  const isPending = isValid && (normalizedQuery !== debouncedQuery || isFetching);

  useEffect(() => {
    const timer = window.setTimeout(() => setDebouncedQuery(normalizedQuery), 300);
    return () => window.clearTimeout(timer);
  }, [normalizedQuery]);

  useEffect(() => {
    setIsOpen(false);
  }, [location]);

  useEffect(() => {
    const dialog = dialogRef.current;
    if (!isOpen) {
      dialog.close();
      return undefined;
    }
    dialog.showModal();
    inputRef.current?.focus();
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    return () => { document.body.style.overflow = previousOverflow; };
  }, [isOpen]);

  function close() {
    setIsOpen(false);
    triggerRef.current?.focus();
  }

  return (
    <>
      <button ref={triggerRef} type="button" onClick={() => setIsOpen(true)} aria-label="Поиск" title="Поиск по архивам" aria-haspopup="dialog" aria-expanded={isOpen}
        className="fixed bottom-[calc(6rem+env(safe-area-inset-bottom))] right-[calc(1rem+env(safe-area-inset-right))] z-30 inline-flex h-12 w-12 items-center justify-center rounded-full border border-white/20 bg-slate-900/90 text-slate-100 shadow-lg shadow-black/30 backdrop-blur-md transition hover:bg-violet-600 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-violet-400 sm:right-[calc(1.5rem+env(safe-area-inset-right))]">
        <SearchIcon />
      </button>
      <dialog ref={dialogRef} aria-labelledby="archive-search-title" className="archive-search-dialog text-slate-100"
        onCancel={(event) => { event.preventDefault(); close(); }}
        onClick={(event) => { if (event.target === event.currentTarget) close(); }}>
        <section className="archive-search-panel rounded-t-3xl border border-white/10 bg-slate-950 p-5 shadow-2xl sm:rounded-3xl sm:p-6">
          <div className="flex items-center justify-between gap-4">
            <h2 id="archive-search-title" className="text-xl font-semibold">Поиск по архивам</h2>
            <button type="button" onClick={close} aria-label="Закрыть поиск" className="flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-white/5 text-2xl hover:bg-white/10">×</button>
          </div>
          <label htmlFor="archive-search-input" className="mt-4 block text-sm text-slate-300">Клиент или оборудование</label>
          <div className="mt-2 flex items-center gap-3 rounded-2xl border border-white/20 bg-white/5 px-4 focus-within:border-violet-400">
            <SearchIcon />
            <input ref={inputRef} id="archive-search-input" type="search" autoFocus maxLength={100} autoComplete="off" value={query}
              onKeyDown={(event) => {
                if (event.key === "Escape") {
                  event.preventDefault();
                  event.stopPropagation();
                  close();
                }
              }}
              onChange={(event) => setQuery(event.target.value)} placeholder="Название, адрес, серийный номер…"
              className="min-w-0 flex-1 bg-transparent py-4 text-base outline-none placeholder:text-slate-500" />
          </div>
          <p className="mt-3 text-xs leading-relaxed text-slate-400">Клиенты: название, адрес, регион. Оборудование: название, серийный номер, производитель, свойства.</p>
          <div className="mt-5" aria-live="polite" aria-busy={isPending}>
            {!isValid ? <p className="py-6 text-center text-sm text-slate-400">Введите минимум 2 символа для поиска.</p>
              : isPending ? <p className="py-6 text-center text-sm text-slate-400">Ищем…</p>
              : isError ? <div className="py-6 text-center text-sm text-rose-200"><p>Не удалось выполнить поиск.</p><button type="button" onClick={refetch} className="mt-3 rounded-lg bg-white/10 px-4 py-2 text-white">Повторить</button></div>
              : !currentData?.length ? <p className="py-6 text-center text-sm text-slate-400">Ничего не найдено. Попробуйте другое название или номер.</p>
              : [
                { kind: "client", title: "Клиенты", path: routePaths.clientArchiveById },
                { kind: "device", title: "Оборудование", path: routePaths.deviceArchiveById },
              ].map((group) => {
                const items = currentData.filter((item) => item.kind === group.kind);
                if (!items.length) return null;
                return <section key={group.kind} className="mb-5">
                  <h3 className="mb-2 text-xs font-semibold uppercase tracking-wider text-slate-400">{group.title}</h3>
                  <ul className="space-y-2">
                    {items.slice(0, 20).map((item) => <li key={item.id}>
                      <Link to={group.path(item.id)} onClick={close} className="group block rounded-xl border border-white/10 px-4 py-3 transition hover:border-violet-400/50 hover:bg-violet-400/10 focus-visible:outline focus-visible:outline-2 focus-visible:outline-violet-400">
                        <span className="block break-words font-medium text-slate-100">{item.title || "Без названия"}</span>
                        {item.subtitle ? <span className="mt-1 block break-words text-sm text-slate-400">{item.subtitle}</span> : null}
                        <span className="mt-2 block text-xs text-violet-300">Открыть архив →</span>
                      </Link>
                    </li>)}
                  </ul>
                  {items.length > 20 ? <p className="mt-3 text-xs text-slate-400">Показаны первые 20 результатов. Уточните запрос.</p> : null}
                </section>;
              })}
          </div>
        </section>
      </dialog>
    </>
  );
}
