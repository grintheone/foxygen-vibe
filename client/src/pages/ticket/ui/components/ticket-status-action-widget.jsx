export function TicketStatusActionWidget({
  actionState,
  errorMessage,
  isLoading,
  onSubmit,
}) {
  if (!actionState?.isVisible) {
    return null;
  }

  return (
    <div className="fixed bottom-4 left-1/2 z-40 w-[min(56rem,calc(100%-2rem))] -translate-x-1/2 sm:bottom-6">
      <div className="rounded-3xl border border-white/10 bg-slate-950/35 p-3 shadow-xl shadow-black/20 backdrop-blur">
        <button
          type="button"
          disabled={!actionState.isEnabled || isLoading}
          onClick={onSubmit}
          className={`flex min-h-14 w-full items-center justify-center rounded-2xl px-5 text-base font-semibold text-white transition sm:text-lg ${
            actionState.isEnabled
              ? actionState.colorClassName
              : "cursor-not-allowed bg-slate-600/70 text-slate-300"
          }`}
        >
          {isLoading ? (
            "Сохраняем..."
          ) : (
            <>
              {actionState.hasSuccessIcon ? (
                <span className="mr-2 inline-flex h-6 w-6 items-center justify-center rounded-full bg-white/20">
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    strokeWidth="2.5"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    className="h-4 w-4"
                    aria-hidden="true"
                  >
                    <path d="M20 6 9 17l-5-5" />
                  </svg>
                </span>
              ) : null}
              <span>{actionState.title}</span>
            </>
          )}
        </button>

        {errorMessage ? (
          <p className="mt-2 px-1 text-center text-xs text-rose-200">{errorMessage}</p>
        ) : null}
      </div>
    </div>
  );
}
