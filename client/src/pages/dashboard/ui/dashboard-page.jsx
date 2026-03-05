import { useNavigate } from "react-router";
import { routePaths } from "../../../shared/config/routes";
import { PageShell } from "../../../shared/ui/page-shell";

export function DashboardPage() {
  const navigate = useNavigate();

  return (
    <PageShell>
      <section className="w-full">
        <div className="flex justify-end">
          <button
            type="button"
            onClick={() => navigate(routePaths.profile)}
            className="rounded-2xl bg-[#6A3BF2] px-5 py-3 text-xs font-semibold uppercase tracking-[0.2em] text-white transition hover:bg-[#7C52F5]"
          >
            Открыть профиль
          </button>
        </div>
      </section>
    </PageShell>
  );
}
