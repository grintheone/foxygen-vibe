import { useNavigate } from "react-router";
import { useAuth } from "../../../features/auth";
import { routePaths } from "../../../shared/config/routes";
import { PageShell } from "../../../shared/ui/page-shell";

function BackButton({ onClick }) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-label="Назад"
      className="inline-flex h-11 w-11 items-center justify-center rounded-2xl bg-[#6A3BF2] text-white shadow-lg shadow-[#6A3BF2]/35 transition hover:bg-[#7C52F5]"
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
  );
}

export function EditorPage() {
  const navigate = useNavigate();
  const { session } = useAuth();
  const canOpenEditor = session?.role === "coordinator" || session?.role === "admin";

  function handleBack() {
    navigate(routePaths.profile);
  }

  if (!canOpenEditor) {
    return (
      <PageShell>
        <section className="w-full space-y-6">
          <header className="rounded-3xl border border-white/10 bg-slate-950/35 p-6 shadow-xl shadow-black/20 backdrop-blur">
            <div className="flex items-center justify-between gap-4">
              <div>
                <p className="text-xs font-semibold uppercase tracking-[0.3em] text-slate-400">Редактор</p>
                <h1 className="mt-3 text-3xl font-bold tracking-tight text-white sm:text-4xl">Нет доступа</h1>
              </div>
              <BackButton onClick={handleBack} />
            </div>
          </header>

          <section className="rounded-3xl border border-rose-300/20 bg-rose-500/10 p-6 shadow-xl shadow-black/20 backdrop-blur">
            <p className="text-base text-rose-50">
              Редактор пока доступен только координаторам и администраторам.
            </p>
          </section>
        </section>
      </PageShell>
    );
  }

  return (
    <PageShell>
      <section className="w-full space-y-6">
        <header className="rounded-3xl border border-white/10 bg-slate-950/35 p-6 shadow-xl shadow-black/20 backdrop-blur">
          <div className="flex items-center justify-between gap-4">
            <div>
              <p className="text-xs font-semibold uppercase tracking-[0.3em] text-slate-400">Редактор</p>
              <h1 className="mt-3 text-3xl font-bold tracking-tight text-white sm:text-4xl">База данных</h1>
            </div>
            <BackButton onClick={handleBack} />
          </div>
        </header>

        <section className="rounded-[2rem] border border-white/10 bg-white/10 p-8 shadow-2xl shadow-[#6A3BF2]/20 backdrop-blur-xl">
          <p className="text-xs font-semibold uppercase tracking-[0.32em] text-cyan-200">Скоро здесь</p>
          <h2 className="mt-4 text-3xl font-bold tracking-tight text-white sm:text-4xl">
            Редактирование всех сущностей
          </h2>
          <p className="mt-4 max-w-2xl text-base text-slate-300">
            Это заглушка будущего редактора. Здесь появится управление всеми сущностями базы данных:
            пользователями, клиентами, устройствами, тикетами, отделами и связанными справочниками.
          </p>
        </section>
      </section>
    </PageShell>
  );
}
