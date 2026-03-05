import { useNavigate, useParams } from "react-router";
import { routePaths } from "../../../shared/config/routes";
import { PageShell } from "../../../shared/ui/page-shell";

export function TicketPage() {
  const navigate = useNavigate();
  const { ticketId } = useParams();

  return (
    <PageShell>
      <section className="w-full space-y-6">
        <header className="rounded-3xl border border-white/10 bg-slate-950/35 p-6 shadow-xl shadow-black/20 backdrop-blur">
          <p className="text-xs font-semibold uppercase tracking-[0.3em] text-slate-400">Тикет</p>
          <h1 className="mt-3 text-3xl font-bold tracking-tight sm:text-4xl">Заявка #{ticketId}</h1>
        </header>

        <div className="rounded-3xl border border-white/10 bg-white/5 p-6">
          <p className="text-sm text-slate-200">Детальная страница тикета {ticketId} (временный экран).</p>
        </div>

        <div className="flex justify-end">
          <button
            type="button"
            onClick={() => navigate(routePaths.dashboard)}
            className="rounded-2xl bg-[#6A3BF2] px-5 py-3 text-xs font-semibold uppercase tracking-[0.2em] text-white transition hover:bg-[#7C52F5]"
          >
            Назад в дэшборд
          </button>
        </div>
      </section>
    </PageShell>
  );
}
