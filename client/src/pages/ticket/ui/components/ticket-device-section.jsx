export function TicketDeviceCard({ deviceName, serialNumber, disabled, onOpenDevice }) {
    return (
        <button
            type="button"
            onClick={onOpenDevice}
            disabled={disabled}
            className="flex w-full items-center justify-between gap-4 rounded-lg border border-slate-400/20 bg-[#2f3748] px-4 py-4 text-left shadow-xl shadow-black/20 transition hover:border-slate-300/35 hover:bg-[#333c4f] disabled:cursor-not-allowed disabled:opacity-70"
        >
            <div className="min-w-0 flex-1">
                <p className="text-[16px] font-semibold leading-snug tracking-tight text-slate-50">
                    {deviceName || "Не указано"}
                </p>
                <p className="mt-2 text-[16px] leading-snug text-slate-200/85">
                    {serialNumber ? `С/Н: ${serialNumber}` : "Серийный номер не указан"}
                </p>
            </div>
            <svg
                xmlns="http://www.w3.org/2000/svg"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2.2"
                strokeLinecap="round"
                strokeLinejoin="round"
                className="h-5 w-5 shrink-0 self-center text-slate-100"
                aria-hidden="true"
            >
                <path d="M9 6l6 6-6 6" />
            </svg>
        </button>
    );
}

export function TicketDeviceSection({ deviceName, serialNumber, disabled, onOpenDevice }) {
    return (
        <section className="space-y-4">
            <h2 className="text-[16px] font-semibold tracking-tight text-[#BCC2CA] sm:text-[18px] lg:text-[20px]">
                Оборудование
            </h2>

            <TicketDeviceCard
                deviceName={deviceName}
                serialNumber={serialNumber}
                disabled={disabled}
                onOpenDevice={onOpenDevice}
            />
        </section>
    );
}
