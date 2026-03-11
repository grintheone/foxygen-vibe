export function BottomPageAction({
  buttonClassName = "bg-[#6A3BF2] hover:bg-[#7C52F5]",
  children,
  disabled = false,
  errorMessage = "",
  onClick,
  type = "button",
}) {
  return (
    <div className="fixed bottom-4 left-1/2 z-40 w-[min(56rem,calc(100%-2rem))] -translate-x-1/2 sm:bottom-6">
      <div className="rounded-3xl border border-white/10 bg-slate-950/35 p-3 shadow-xl shadow-black/20 backdrop-blur">
        <button
          type={type}
          disabled={disabled}
          onClick={onClick}
          className={`flex min-h-14 w-full items-center justify-center rounded-2xl px-5 text-base font-semibold text-white transition sm:text-lg ${
            disabled ? "cursor-not-allowed bg-slate-600/70 text-slate-300" : buttonClassName
          }`}
        >
          {children}
        </button>

        {errorMessage ? (
          <p className="mt-2 px-1 text-center text-xs text-rose-200">{errorMessage}</p>
        ) : null}
      </div>
    </div>
  );
}
