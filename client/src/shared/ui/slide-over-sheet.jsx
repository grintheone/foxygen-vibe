import { useEffect } from "react";

export function SlideOverSheet({
  children,
  closeLabel = "Закрыть",
  eyebrow,
  isOpen,
  onClose,
  panelClassName = "",
  title,
}) {
  useEffect(() => {
    if (!isOpen) {
      return undefined;
    }

    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";

    return () => {
      document.body.style.overflow = previousOverflow;
    };
  }, [isOpen]);

  if (!isOpen) {
    return null;
  }

  return (
    <div className="fixed inset-0 z-50">
      <button
        type="button"
        aria-label={closeLabel}
        onClick={onClose}
        className="absolute inset-0 bg-black/55"
      />

      <aside
        className={`absolute right-0 top-0 h-full w-full overflow-y-auto border-l border-white/10 bg-slate-950/95 p-6 shadow-2xl shadow-black/50 backdrop-blur md:w-[78%] lg:w-[33.333%] ${panelClassName}`.trim()}
      >
        {(eyebrow || title) ? (
          <div className="flex items-start justify-between gap-4">
            <div>
              {eyebrow ? (
                <p className="text-xs font-semibold uppercase tracking-[0.25em] text-slate-400">{eyebrow}</p>
              ) : null}
              {title ? <h2 className="mt-2 text-2xl font-semibold text-slate-100">{title}</h2> : null}
            </div>

            <button
              type="button"
              onClick={onClose}
              className="inline-flex h-10 w-10 items-center justify-center rounded-2xl bg-white/10 text-slate-100 transition hover:bg-white/20"
              aria-label={closeLabel}
            >
              <svg
                xmlns="http://www.w3.org/2000/svg"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2.5"
                strokeLinecap="round"
                strokeLinejoin="round"
                className="h-5 w-5"
                aria-hidden="true"
              >
                <path d="M18 6 6 18" />
                <path d="m6 6 12 12" />
              </svg>
            </button>
          </div>
        ) : null}

        {children}
      </aside>
    </div>
  );
}
