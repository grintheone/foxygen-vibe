import fireIcon from "../../../assets/icons/fire-icon.svg";
import ticketClosedIcon from "../../../assets/icons/ticket-closed.svg";
import ticketDoneIcon from "../../../assets/icons/ticket-done.svg";
import ticketAssignedIcon from "../../../assets/icons/ticket-assigned.svg";
import { useNavigate, useParams } from "react-router";
import { useAuth } from "../../../features/auth";
import {
  useGetMyProfileQuery,
  useGetProfileByIdQuery,
} from "../../../shared/api/tickets-api";
import { routePaths } from "../../../shared/config/routes";
import { ProfileTicketCard } from "../../../shared/ui/profile-ticket-card";
import { PageShell } from "../../../shared/ui/page-shell";
import { StatusMessage } from "../../../shared/ui/status-message";

function resolveErrorMessage(error, fallbackMessage) {
  if (typeof error?.data === "string" && error.data.trim()) {
    return error.data;
  }

  if (typeof error?.error === "string" && error.error.trim()) {
    return error.error;
  }

  return fallbackMessage;
}

function PersonIcon({ className }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      className={className}
      aria-hidden="true"
    >
      <circle cx="12" cy="8" r="3.6" />
      <path d="M4.5 19.2C5.9 15.9 8.6 14.4 12 14.4s6.1 1.5 7.5 4.8" />
    </svg>
  );
}

function BackButton({ onClick }) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-label="Назад на дашборд"
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

function ProfileAvatar({ logo, name }) {
  const initials = (name || "")
    .trim()
    .split(/\s+/)
    .slice(0, 2)
    .map((part) => part.charAt(0).toUpperCase())
    .join("");

  if (logo?.trim()) {
    return (
      <img
        src={logo}
        alt={name || "Фото сотрудника"}
        className="h-28 w-28 rounded-[2rem] border border-white/10 object-cover shadow-lg shadow-black/20 sm:h-32 sm:w-32"
      />
    );
  }

  return (
    <div className="flex h-28 w-28 items-center justify-center rounded-[2rem] border border-white/10 bg-gradient-to-br from-cyan-400/25 via-[#6A3BF2]/30 to-emerald-400/25 text-white shadow-lg shadow-black/20 sm:h-32 sm:w-32">
      {initials ? (
        <span className="text-3xl font-semibold tracking-[0.08em]">{initials}</span>
      ) : (
        <PersonIcon className="h-10 w-10" />
      )}
    </div>
  );
}

function ProfileActivityIndicator({ latestTicketStatus }) {
  if (latestTicketStatus === "inWork") {
    return (
      <div className="inline-flex items-center gap-3 rounded-full border border-violet-200/30 bg-violet-500/15 px-4 py-2 text-sm font-semibold text-violet-50">
        <span className="relative flex h-2.5 w-2.5">
          <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-violet-100 opacity-90" />
          <span className="relative inline-flex h-2.5 w-2.5 rounded-full bg-violet-50" />
        </span>
        <span>На выезде</span>
      </div>
    );
  }

  if (latestTicketStatus === "worksDone" || latestTicketStatus === "closed") {
    return (
      <div className="inline-flex items-center gap-3 rounded-full border border-emerald-200/30 bg-emerald-500/15 px-4 py-2 text-sm font-semibold text-emerald-50">
        <span className="relative flex h-2.5 w-2.5">
          <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-100 opacity-90" />
          <span className="relative inline-flex h-2.5 w-2.5 rounded-full bg-emerald-50" />
        </span>
        <span>Работы завершены</span>
      </div>
    );
  }

  return null;
}

function ProfileInfoRow({ label, value }) {
  return (
    <div className="rounded-2xl border border-white/10 bg-white/5 px-4 py-3">
      <p className="text-xs font-semibold uppercase tracking-[0.24em] text-slate-400">{label}</p>
      <p className="mt-2 break-all text-base text-slate-100">{value}</p>
    </div>
  );
}

function ProfileStatCard({ icon, iconAlt = "", label, value, toneClassName }) {
  return (
    <article className="rounded-3xl border border-white/10 bg-slate-950/35 p-5 shadow-lg shadow-black/10">
      <div className="flex items-center justify-between gap-3">
        <p className="text-sm font-semibold text-slate-300">{label}</p>
        <span className={`inline-flex h-10 w-10 items-center justify-center rounded-2xl ${toneClassName}`}>
          <img src={icon} alt={iconAlt} className="h-5 w-5" />
        </span>
      </div>
      <p className="mt-4 text-3xl font-bold tracking-tight text-white">{value}</p>
    </article>
  );
}

function ProfileHeader({ isMemberProfile, onBack }) {
  return (
    <header className="rounded-3xl border border-white/10 bg-slate-950/35 p-6 shadow-xl shadow-black/20 backdrop-blur">
      <div className="flex items-center justify-between gap-4">
        <div>
          <p className="text-xs font-semibold uppercase tracking-[0.3em] text-slate-400">
            {isMemberProfile ? "Профиль сотрудника" : "Мой профиль"}
          </p>
          <h1 className="mt-3 text-3xl font-bold tracking-tight text-white sm:text-4xl">Профиль</h1>
        </div>
        <BackButton onClick={onBack} />
      </div>
    </header>
  );
}

function ActiveTicketsSection({ archiveHref, isMemberProfile, onOpenArchive, onOpenTicket, tickets }) {
  return (
    <section className="space-y-4">
      <h2 className="text-2xl font-bold tracking-tight text-slate-50 sm:text-3xl">Активные выезды</h2>

      {tickets.length > 0 ? (
        <div className="grid gap-3">
          {tickets.map((ticket) => (
            <ProfileTicketCard key={ticket.id} ticket={ticket} onOpenTicket={onOpenTicket} />
          ))}
        </div>
      ) : (
        <div className="rounded-3xl border border-white/10 bg-white/5 p-6">
          <p className="text-sm text-slate-300">
            {isMemberProfile
              ? "У этого сотрудника сейчас нет активных выездов."
              : "У вас сейчас нет активных выездов."}
          </p>
        </div>
      )}

      {archiveHref ? (
        <button
          type="button"
          onClick={onOpenArchive}
          className="inline-flex items-center gap-3 rounded-2xl px-2 py-1 text-lg font-semibold text-[#9B7BFF] transition hover:text-[#B49CFF]"
        >
          <span>Архив выездов</span>
          <svg
            xmlns="http://www.w3.org/2000/svg"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2.2"
            strokeLinecap="round"
            strokeLinejoin="round"
            className="h-5 w-5"
            aria-hidden="true"
          >
            <path d="M5 12h14" />
            <path d="m13 6 6 6-6 6" />
          </svg>
        </button>
      ) : null}
    </section>
  );
}

function ProfileActionsSection({ canOpenEditor, isOwnProfile, onEditorOpen, onSignOut }) {
  if (!isOwnProfile) {
    return null;
  }

  return (
    <section className="rounded-3xl border border-white/10 bg-slate-950/35 p-6 shadow-xl shadow-black/20 backdrop-blur">
      <p className="text-xs font-semibold uppercase tracking-[0.3em] text-slate-400">Действия</p>
      <div className="mt-5 grid gap-3 sm:grid-cols-2">
        <button
          type="button"
          onClick={onEditorOpen}
          disabled={!canOpenEditor}
          className={`rounded-2xl px-4 py-3 text-sm font-semibold transition ${
            canOpenEditor
              ? "border border-cyan-200/25 bg-cyan-400/15 text-cyan-50 hover:border-cyan-100/40 hover:bg-cyan-400/20"
              : "cursor-not-allowed border border-white/10 bg-white/5 text-slate-500 opacity-70"
          }`}
        >
          Редактор
        </button>
        <button
          type="button"
          onClick={onSignOut}
          className="rounded-2xl border border-rose-300/20 bg-rose-500/10 px-4 py-3 text-sm font-semibold text-rose-50 transition hover:border-rose-200/35 hover:bg-rose-500/15"
        >
          Выйти из аккаунта
        </button>
      </div>
    </section>
  );
}

export function ProfilePage() {
  const navigate = useNavigate();
  const { signOut } = useAuth();
  const { userId } = useParams();
  const isMemberProfile = Boolean(userId);
  const {
    data: ownProfile,
    error: ownProfileError,
    isFetching: isOwnProfileFetching,
    isLoading: isOwnProfileLoading,
  } = useGetMyProfileQuery(undefined, {
    skip: isMemberProfile,
  });
  const {
    data: memberProfile,
    error: memberProfileError,
    isFetching: isMemberProfileFetching,
    isLoading: isMemberProfileLoading,
  } = useGetProfileByIdQuery(userId, {
    skip: !isMemberProfile,
  });

  const profile = isMemberProfile ? memberProfile : ownProfile;
  const isLoading = isMemberProfile
    ? isMemberProfileLoading || isMemberProfileFetching
    : isOwnProfileLoading || isOwnProfileFetching;
  const errorMessage = isMemberProfile
    ? resolveErrorMessage(memberProfileError, "Не удалось загрузить профиль сотрудника.")
    : resolveErrorMessage(ownProfileError, "Не удалось загрузить профиль.");
  const hasError = isMemberProfile ? Boolean(memberProfileError) : Boolean(ownProfileError);
  const displayName = profile?.name?.trim() || profile?.username?.trim() || "Сотрудник";
  const departmentTitle = profile?.department?.trim() || "Отдел не указан";
  const emailValue = profile?.email?.trim() || "Не указан";
  const phoneValue = profile?.phone?.trim() || "Не указан";
  const archiveHref = profile?.user_id ? routePaths.profileArchiveById(profile.user_id) : "";
  const activeTickets = profile?.activeTickets || [];
  const canOpenEditor = profile?.role === "coordinator" || profile?.role === "admin";

  function handleBack() {
    navigate(routePaths.dashboard);
  }

  function handleOpenArchive() {
    if (!archiveHref) {
      return;
    }

    navigate(archiveHref);
  }

  function handleOpenTicket(ticketId) {
    navigate(routePaths.ticketById(ticketId));
  }

  function handleSignOut() {
    signOut();
    navigate(routePaths.signIn);
  }

  function handleOpenEditor() {
    if (!canOpenEditor) {
      return;
    }

    navigate(routePaths.editor);
  }

  return (
    <PageShell>
      <section className="w-full space-y-6">
        <ProfileHeader isMemberProfile={isMemberProfile} onBack={handleBack} />

        {isLoading ? (
          <section className="rounded-3xl border border-white/10 bg-white/5 p-6">
            <p className="text-sm text-slate-300">
              {isMemberProfile ? "Загружаем профиль сотрудника..." : "Загружаем профиль..."}
            </p>
          </section>
        ) : null}

        {hasError ? <StatusMessage feedback={{ message: errorMessage, tone: "error" }} /> : null}

        {!isLoading && !hasError && profile ? (
          <>
            <section className="grid gap-6 xl:grid-cols-[1.1fr_0.9fr]">
              <article className="rounded-[2rem] border border-white/10 bg-white/10 p-6 shadow-2xl shadow-[#6A3BF2]/20 backdrop-blur-xl sm:p-8">
                <div className="flex flex-col gap-6 sm:flex-row sm:items-start sm:justify-between">
                  <div className="flex items-start gap-4 sm:gap-5">
                    <ProfileAvatar logo={profile.logo} name={displayName} />

                    <div className="min-w-0">
                      <p className="text-xs font-semibold uppercase tracking-[0.32em] text-cyan-200">
                        {isMemberProfile ? "Карточка сотрудника" : "Личный кабинет"}
                      </p>
                      <h2 className="mt-3 text-3xl font-bold tracking-tight text-white sm:text-4xl">{displayName}</h2>
                      <p className="mt-2 text-lg font-semibold text-slate-200">{departmentTitle}</p>
                    </div>
                  </div>

                  <div className="shrink-0">
                    <ProfileActivityIndicator latestTicketStatus={profile.latestTicketStatus} />
                  </div>
                </div>

                <div className="mt-8 grid gap-3 sm:grid-cols-2">
                  <ProfileInfoRow label="Email" value={emailValue} />
                  <ProfileInfoRow label="Телефон" value={phoneValue} />
                </div>
              </article>

              <section className="grid gap-4 sm:grid-cols-2">
                <ProfileStatCard
                  icon={ticketAssignedIcon}
                  iconAlt=""
                  label="Всего тикетов"
                  value={profile.ticketStats?.total || 0}
                  toneClassName="border border-cyan-200/25 bg-cyan-400/15"
                />
                <ProfileStatCard
                  icon={ticketClosedIcon}
                  iconAlt=""
                  label="Закрыто"
                  value={profile.ticketStats?.closed || 0}
                  toneClassName="border border-emerald-200/25 bg-emerald-400/15"
                />
                <ProfileStatCard
                  icon={fireIcon}
                  iconAlt=""
                  label="Просрочено"
                  value={profile.ticketStats?.overdue || 0}
                  toneClassName="border border-rose-200/25 bg-rose-500/15"
                />
                <ProfileStatCard
                  icon={ticketDoneIcon}
                  iconAlt=""
                  label="Закрыто за месяц"
                  value={profile.ticketStats?.closedThisMonth || 0}
                  toneClassName="border border-violet-200/25 bg-violet-500/15"
                />
              </section>
            </section>

            <ActiveTicketsSection
              archiveHref={archiveHref}
              isMemberProfile={isMemberProfile}
              onOpenArchive={handleOpenArchive}
              onOpenTicket={handleOpenTicket}
              tickets={activeTickets}
            />

            <ProfileActionsSection
              canOpenEditor={canOpenEditor}
              isOwnProfile={!isMemberProfile}
              onEditorOpen={handleOpenEditor}
              onSignOut={handleSignOut}
            />
          </>
        ) : null}
      </section>
    </PageShell>
  );
}
