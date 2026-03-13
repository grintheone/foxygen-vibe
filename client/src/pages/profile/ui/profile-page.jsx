import { useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router";
import { useAuth } from "../../../features/auth";
import { fetchWithAuth } from "../../../shared/api/authenticated-fetch";
import { routePaths } from "../../../shared/config/routes";
import { PageShell } from "../../../shared/ui/page-shell";
import { StatusMessage } from "../../../shared/ui/status-message";
import { DashboardOverview } from "../../../widgets/dashboard-overview";
import { SessionPanel } from "../../../widgets/session-panel";

export function ProfilePage() {
  const navigate = useNavigate();
  const { userId } = useParams();
  const { feedback, isRefreshing, rotateSession, session, signOut } = useAuth();
  const [memberProfile, setMemberProfile] = useState(null);
  const [isLoadingMember, setIsLoadingMember] = useState(false);
  const [memberError, setMemberError] = useState("");
  const isMemberProfile = Boolean(userId);

  useEffect(() => {
    if (!isMemberProfile) {
      setMemberProfile(null);
      setMemberError("");
      setIsLoadingMember(false);
      return;
    }

    let isActive = true;

    async function loadMemberProfile() {
      setIsLoadingMember(true);
      setMemberError("");

      try {
        const response = await fetchWithAuth(`/api/profile/${userId}`);
        if (!response.ok) {
          const message = (await response.text()) || "Не удалось загрузить профиль сотрудника.";
          throw new Error(message);
        }

        const data = await response.json();
        if (isActive) {
          setMemberProfile(data);
        }
      } catch (error) {
        if (isActive) {
          setMemberProfile(null);
          setMemberError(error.message || "Не удалось загрузить профиль сотрудника.");
        }
      } finally {
        if (isActive) {
          setIsLoadingMember(false);
        }
      }
    }

    loadMemberProfile();

    return () => {
      isActive = false;
    };
  }, [isMemberProfile, userId]);

  function handleRotate() {
    rotateSession().catch(() => {});
  }

  function handleSignOut() {
    signOut();
    navigate(routePaths.signIn);
  }

  function handleBackToDashboard() {
    navigate(routePaths.dashboard);
  }

  const profileData = isMemberProfile ? memberProfile : session;

  return (
    <PageShell>
      <section className="grid w-full gap-6 lg:grid-cols-[1.35fr_0.85fr]">
        {isMemberProfile && isLoadingMember ? (
          <div className="rounded-[2rem] border border-white/10 bg-white/10 p-8 shadow-2xl shadow-[#6A3BF2]/25 backdrop-blur-xl">
            <p className="text-sm text-slate-300">Загружаем профиль сотрудника...</p>
          </div>
        ) : isMemberProfile && memberError ? (
          <StatusMessage feedback={{ message: memberError, tone: "error" }} />
        ) : (
          <DashboardOverview isMemberProfile={isMemberProfile} session={profileData} />
        )}
        <aside className="space-y-6">
          {isMemberProfile ? (
            <section className="rounded-[2rem] border border-cyan-300/20 bg-cyan-400/10 p-6 shadow-2xl shadow-cyan-950/30 backdrop-blur-xl">
              <p className="text-xs font-semibold uppercase tracking-[0.35em] text-cyan-200">
                Профиль сотрудника
              </p>
              <p className="mt-4 text-xl font-semibold text-slate-50">
                {profileData?.username || "Сотрудник"}
              </p>
              <p className="mt-2 text-sm text-slate-300">
                Просмотр карточки сотрудника из вашего отдела.
              </p>

              <div className="mt-5 grid gap-3">
                <button
                  type="button"
                  onClick={handleBackToDashboard}
                  className="rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-xs font-semibold uppercase tracking-[0.2em] text-slate-100 transition hover:bg-white/10"
                >
                  На дашборд
                </button>
              </div>
            </section>
          ) : (
            <>
              <SessionPanel
                session={session}
                isRefreshing={isRefreshing}
                onRotate={handleRotate}
                onSignOut={handleSignOut}
              />
              {feedback.message ? <StatusMessage feedback={feedback} /> : null}
            </>
          )}
        </aside>
      </section>
    </PageShell>
  );
}
