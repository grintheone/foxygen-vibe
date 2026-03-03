import { useNavigate } from "react-router";
import { useAuth } from "../../../features/auth";
import { routePaths } from "../../../shared/config/routes";
import { demoAccounts } from "../../../shared/model/demo-accounts";
import { PageShell } from "../../../shared/ui/page-shell";
import { StatusMessage } from "../../../shared/ui/status-message";
import { DashboardOverview } from "../../../widgets/dashboard-overview";
import { SessionPanel } from "../../../widgets/session-panel";

export function DashboardPage() {
  const navigate = useNavigate();
  const { feedback, isRefreshing, rotateSession, session, signOut } = useAuth();

  const activeDemo = demoAccounts.find(
    (account) => account.username === session?.username,
  );

  function handleRotate() {
    rotateSession().catch(() => {});
  }

  function handleSignOut() {
    signOut();
    navigate(routePaths.signIn);
  }

  return (
    <PageShell>
      <section className="grid w-full gap-6 lg:grid-cols-[1.35fr_0.85fr]">
        <DashboardOverview session={session} />
        <aside className="space-y-6">
          <SessionPanel
            activeDemo={activeDemo}
            isRefreshing={isRefreshing}
            onRotate={handleRotate}
            onSignOut={handleSignOut}
          />
          {feedback.message ? <StatusMessage feedback={feedback} /> : null}
        </aside>
      </section>
    </PageShell>
  );
}
