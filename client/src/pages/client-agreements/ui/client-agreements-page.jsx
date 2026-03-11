import { useNavigate, useParams } from "react-router";
import { routePaths } from "../../../shared/config/routes";
import { PageShell } from "../../../shared/ui/page-shell";

export function ClientAgreementsPage() {
  const navigate = useNavigate();
  const { clientId } = useParams();

  return (
    <PageShell>
      <section className="w-full space-y-6">
        <header className="rounded-3xl border border-white/10 bg-slate-950/35 p-6 shadow-xl shadow-black/20 backdrop-blur">
          <button
            type="button"
            onClick={() => navigate(routePaths.clientById(clientId))}
            className="inline-flex h-11 w-11 items-center justify-center rounded-2xl bg-[#6A3BF2] text-white transition hover:bg-[#7C52F5]"
            aria-label="Назад"
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
              <path d="M15 18l-6-6 6-6" />
            </svg>
          </button>
          <p className="mt-4 text-xs font-semibold uppercase tracking-[0.3em] text-slate-400">Оборудование</p>
          <h1 className="mt-2 text-3xl font-bold tracking-tight sm:text-4xl">Все оборудование</h1>
        </header>

        <div className="rounded-3xl border border-white/10 bg-white/5 p-6">
          <p className="text-sm text-slate-200">
            Страница оборудования клиента {clientId || "—"} пока не реализована.
          </p>
        </div>
      </section>
    </PageShell>
  );
}
