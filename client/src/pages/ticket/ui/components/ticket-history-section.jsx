import { MOCK_TICKET_HISTORY } from "../../model/ticket-page-model";

export function TicketHistorySection({ historyActorName }) {
    return (
        <section className="space-y-5 rounded-3xl border border-white/15 bg-white/10 p-5 sm:p-6">
            <h2 className="text-4xl font-semibold tracking-tight text-slate-300">История тикета</h2>
            <div className="space-y-6">
                {MOCK_TICKET_HISTORY.map((entry, index) => {
                    const isLast = index === MOCK_TICKET_HISTORY.length - 1;

                    return (
                        <article key={entry.id} className="grid grid-cols-[1fr_auto] gap-4">
                            <div className="flex min-w-0 items-start gap-4">
                                <span className="mt-1 inline-flex h-12 w-12 shrink-0 rounded-full bg-slate-950" />
                                <div className="min-w-0">
                                    <p className="text-2xl text-slate-400">{entry.title}</p>
                                    <p className="text-3xl font-semibold leading-tight text-slate-100">
                                        {historyActorName}
                                    </p>
                                </div>
                            </div>

                            <div className="flex items-start gap-3">
                                <div className="text-right">
                                    <p className="text-2xl font-semibold text-slate-50">{entry.date}</p>
                                    <p className="text-2xl text-slate-400">{entry.time}</p>
                                </div>
                                <div className="relative flex min-h-[4.25rem] w-6 justify-center">
                                    {!isLast ? (
                                        <span className="absolute left-1/2 top-7 h-[calc(100%-1.25rem)] w-0.5 -translate-x-1/2 bg-slate-400/60" />
                                    ) : null}
                                    <img src={entry.icon} alt="" className="relative z-[1] mt-1 h-6 w-6" />
                                </div>
                            </div>
                        </article>
                    );
                })}
            </div>
        </section>
    );
}
