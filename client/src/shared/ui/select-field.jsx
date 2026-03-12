export function SelectField({
  children,
  className = "",
  disabled = false,
  id,
  name,
  onChange,
  value,
}) {
  return (
    <div className="relative">
      <select
        id={id}
        name={name}
        value={value}
        onChange={onChange}
        disabled={disabled}
        className={`min-h-16 w-full appearance-none rounded-2xl border border-white/10 bg-slate-950/35 px-4 py-3 pr-12 text-lg text-slate-100 outline-none transition focus:border-[#9B7BFF]/70 focus:ring-2 focus:ring-[#9B7BFF]/20 disabled:cursor-not-allowed disabled:opacity-80 ${className}`}
      >
        {children}
      </select>

      <span
        className="pointer-events-none absolute inset-y-0 right-4 flex items-center text-slate-400"
        aria-hidden="true"
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
          className="h-5 w-5"
        >
          <path d="m6 9 6 6 6-6" />
        </svg>
      </span>
    </div>
  );
}
