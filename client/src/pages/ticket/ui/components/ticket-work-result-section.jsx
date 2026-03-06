import { MOCK_WORK_RESULT } from "../../model/ticket-page-model";

export function TicketWorkResultSection({ ticket, workDuration }) {
    return (
        <section className="space-y-3 rounded-3xl border border-emerald-300/25 bg-emerald-500/10 p-5 sm:p-6">
            <div className="flex items-center justify-between gap-4">
                <h2 className="text-3xl font-semibold tracking-tight text-emerald-100">Результат работы</h2>
                <p className="inline-flex items-center gap-2 text-2xl font-semibold text-emerald-100">
                    <svg
                        xmlns="http://www.w3.org/2000/svg"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        strokeWidth="2.2"
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        className="h-6 w-6"
                        aria-hidden="true"
                    >
                        <circle cx="12" cy="12" r="8" />
                        <path d="M12 8v5l3 2" />
                    </svg>
                    {workDuration}
                </p>
            </div>

            <div className="rounded-2xl border border-white/15 bg-white/10 p-5 shadow-lg shadow-black/15">
                <div className="flex items-start gap-4">
                    <span className="inline-flex h-12 w-12 shrink-0 items-center justify-center rounded-full bg-slate-950 text-sm font-semibold text-slate-100">
                        {ticket.executorName ? ticket.executorName.trim().charAt(0).toUpperCase() : "?"}
                    </span>
                    <div className="min-w-0">
                        <p className="text-2xl font-semibold leading-tight text-slate-100">
                            {ticket.executorName || "Исполнитель не назначен"}
                        </p>
                        <p className="text-2xl text-slate-400">{ticket.executorDepartment || "Отдел не указан"}</p>
                    </div>
                </div>

                <p className="mt-4 text-2xl leading-relaxed text-slate-200">{ticket.result || MOCK_WORK_RESULT}</p>
            </div>

            <div className="grid grid-cols-3 gap-2 sm:gap-3">
                <div className="flex aspect-[4/3] items-center justify-center rounded-xl border border-white/15 bg-slate-900/25 text-sm font-semibold text-slate-200">
                    Фото 1
                </div>
                <div className="flex aspect-[4/3] items-center justify-center rounded-xl border border-white/15 bg-slate-900/25 text-sm font-semibold text-slate-200">
                    Фото 2
                </div>
                <div className="flex aspect-[4/3] items-center justify-center rounded-xl border border-white/15 bg-slate-900/25 text-sm font-semibold text-slate-200">
                    Фото 3
                </div>
            </div>
        </section>
    );
}
