import { lazy, Suspense } from "react";
import { useNavigate } from "react-router";
import { useAuth } from "../../../features/auth";
import { routePaths } from "../../../shared/config/routes";
import { PageShell } from "../../../shared/ui/page-shell";
import { DashboardHeader } from "./dashboard-header";

const CoordinatorDashboard = lazy(() =>
  import("./coordinator-dashboard").then((module) => ({
    default: module.CoordinatorDashboard,
  })),
);
const EngineerDashboard = lazy(() =>
  import("./engineer-dashboard").then((module) => ({
    default: module.EngineerDashboard,
  })),
);

const dashboardByRole = {
  admin: CoordinatorDashboard,
  coordinator: CoordinatorDashboard,
};

export function DashboardPage() {
  const navigate = useNavigate();
  const { session } = useAuth();
  const role = session?.role || "user";
  const DashboardView = dashboardByRole[role] || EngineerDashboard;

  return (
    <PageShell>
      <section className="w-full space-y-6">
        <DashboardHeader onOpenProfile={() => navigate(routePaths.profile)} />
        <Suspense
          fallback={
            <section className="app-subtle-notice">
              <p className="text-sm text-slate-300">Загружаем панель...</p>
            </section>
          }
        >
          <DashboardView executorId={session?.user_id} department={session?.department} />
        </Suspense>
      </section>
    </PageShell>
  );
}
