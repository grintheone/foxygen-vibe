export function BottomPageAction({
  buttonClassName = "bg-[#6A3BF2] hover:bg-[#7C52F5]",
  children,
  disabled = false,
  errorMessage = "",
  onClick,
  type = "button",
}) {
  return (
    <div className="fixed bottom-4 left-1/2 z-40 flex w-[min(56rem,calc(100%-2rem))] -translate-x-1/2 flex-col gap-2 sm:bottom-6">
      <button
        type={type}
        disabled={disabled}
        onClick={onClick}
        className={`flex min-h-12 w-full items-center justify-center rounded-xl px-5 text-base font-semibold text-white shadow-xl shadow-black/20 transition sm:text-lg ${
          disabled ? "cursor-not-allowed bg-slate-600/70 text-slate-300" : buttonClassName
        }`}
      >
        {children}
      </button>

      {errorMessage ? (
        <p className="px-1 text-center text-xs text-rose-200">{errorMessage}</p>
      ) : null}
    </div>
  );
}
