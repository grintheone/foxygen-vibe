import { useNavigate } from "react-router";
import { useAuth } from "../../../features/auth";
import { routePaths } from "../../../shared/config/routes";
import { PageShell } from "../../../shared/ui/page-shell";
import { CoordinatorDashboard } from "./coordinator-dashboard";
import { DashboardHeader } from "./dashboard-header";
import { EngineerDashboard } from "./engineer-dashboard";

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
        <DashboardView executorId={session?.user_id} department={session?.department} />
      </section>
    </PageShell>
  );
}
