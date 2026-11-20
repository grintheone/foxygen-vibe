export function BottomPageAction({
  buttonClassName = "bg-[#6A3BF2] hover:bg-[#7C52F5]",
  children,
  disabled = false,
  errorMessage = "",
  onClick,
  secondaryAction,
  type = "button",
}) {
  return (
    <div data-bottom-page-action className="fixed bottom-[calc(1rem+env(safe-area-inset-bottom))] left-1/2 z-40 flex w-[min(56rem,calc(100%-2rem))] -translate-x-1/2 flex-col gap-2 sm:bottom-[calc(1.5rem+env(safe-area-inset-bottom))]">
      <div className="flex items-stretch justify-between gap-3">
        <button
          type={type}
          disabled={disabled}
          onClick={onClick}
          className={`flex min-h-12 min-w-0 flex-1 items-center justify-center rounded-xl px-5 text-base font-semibold text-white shadow-xl shadow-black/20 transition sm:text-lg ${
            disabled ? "cursor-not-allowed bg-slate-600/70 text-slate-300" : buttonClassName
          }`}
        >
          {children}
        </button>

        {secondaryAction ? (
          <button
            type="button"
            disabled={disabled}
            onClick={secondaryAction.onClick}
            className="min-h-12 min-w-0 flex-1 rounded-xl bg-rose-600 px-5 font-semibold text-white transition hover:bg-rose-500 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {secondaryAction.title}
          </button>
        ) : null}
      </div>

      {errorMessage ? (
        <p className="px-1 text-center text-xs text-rose-200">{errorMessage}</p>
      ) : null}
    </div>
  );
}
