import { useEffect, useRef, useState } from "react";
import fireIcon from "../../../assets/icons/fire-icon.svg";
import ticketClosedIcon from "../../../assets/icons/ticket-closed.svg";
import ticketDoneIcon from "../../../assets/icons/ticket-done.svg";
import ticketAssignedIcon from "../../../assets/icons/ticket-assigned.svg";
import { useNavigate, useParams } from "react-router";
import { useAuth } from "../../../features/auth";
import {
    useGetMyProfileQuery,
    useGetProfileByIdQuery,
    useSetProfileDisabledMutation,
    useUploadProfileAvatarMutation,
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

function CameraIcon({ className }) {
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
            <path d="M4.5 8.7a2.2 2.2 0 0 1 2.2-2.2H9l1.1-1.8c.4-.6 1-.9 1.7-.9h.4c.7 0 1.3.3 1.7.9L15 6.5h2.3a2.2 2.2 0 0 1 2.2 2.2v7.8a2.2 2.2 0 0 1-2.2 2.2H6.7a2.2 2.2 0 0 1-2.2-2.2V8.7Z" />
            <circle cx="12" cy="12.6" r="3.2" />
        </svg>
    );
}

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

function ProfileAvatar({ canUpload = false, isUploading = false, logo, name, onUploadClick }) {
    const initials = (name || "")
        .trim()
        .split(/\s+/)
        .slice(0, 2)
        .map((part) => part.charAt(0).toUpperCase())
        .join("");

    const avatarContent = logo?.trim() ? (
        <img
            src={logo}
            alt={name || "Фото сотрудника"}
            className="h-28 w-28 rounded-lg border border-white/10 object-cover shadow-lg shadow-black/20 sm:h-32 sm:w-32"
        />
    ) : (
        <div className="flex h-28 w-28 items-center justify-center rounded-lg border border-white/10 bg-gradient-to-br from-cyan-400/25 via-[#6A3BF2]/30 to-emerald-400/25 text-white shadow-lg shadow-black/20 sm:h-32 sm:w-32">
            {initials ? (
                <span className="text-3xl font-semibold tracking-[0.08em]">{initials}</span>
            ) : (
                <PersonIcon className="h-10 w-10" />
            )}
        </div>
    );

    if (!canUpload) {
        return avatarContent;
    }

    return (
        <button
            type="button"
            onClick={onUploadClick}
            disabled={isUploading}
            aria-label={isUploading ? "Загрузка фото профиля" : "Загрузить фото профиля"}
            className="group relative inline-flex rounded-lg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-200/80 focus-visible:ring-offset-2 focus-visible:ring-offset-slate-950 disabled:cursor-wait"
        >
            {avatarContent}
            <span className="absolute inset-x-2 bottom-2 inline-flex items-center justify-center gap-2 rounded-2xl bg-slate-950/75 px-3 py-2 text-[11px] font-semibold uppercase tracking-[0.2em] text-white opacity-100 shadow-lg shadow-black/30 transition sm:opacity-0 sm:group-hover:opacity-100">
                <CameraIcon className="h-3.5 w-3.5" />
                <span>{isUploading ? "Загрузка" : "Сменить фото"}</span>
            </span>
        </button>
    );
}

function ProfileActivityIndicator({ isDisabled = false, latestTicketStatus }) {
    if (isDisabled) {
        return (
            <div className="inline-flex items-center gap-3 rounded-full border border-rose-200/30 bg-rose-500/15 px-4 py-2 text-sm font-semibold text-rose-50">
                <span className="relative flex h-2.5 w-2.5">
                    <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-rose-100 opacity-90" />
                    <span className="relative inline-flex h-2.5 w-2.5 rounded-full bg-rose-50" />
                </span>
                <span>Профиль отключен</span>
            </div>
        );
    }

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

function ProfileDisabledNotice({ isMemberProfile }) {
    return (
        <div className="mt-6 rounded-3xl border border-rose-300/25 bg-rose-500/10 px-5 py-4 text-sm text-rose-50">
            {isMemberProfile
                ? "Этот сотрудник сейчас отключен и не доступен для выбора"
                : "Ваш профиль сейчас отключен."}
        </div>
    );
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
                <div className="grid gap-2">
                    {tickets.map((ticket) => (
                        <ProfileTicketCard key={ticket.id} ticket={ticket} onOpenTicket={onOpenTicket} />
                    ))}
                </div>
            ) : (
                <div className="app-subtle-notice">
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

function resolveProfileActionClassName(tone, disabled, isLoading = false) {
    if (disabled && !isLoading) {
        return "w-full cursor-not-allowed rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-sm font-semibold text-slate-500 opacity-70";
    }

    if (tone === "warning") {
        return "w-full rounded-2xl border border-amber-200/25 bg-amber-400/10 px-4 py-3 text-sm font-semibold text-amber-50 transition hover:border-amber-100/40 hover:bg-amber-400/15 disabled:cursor-wait disabled:opacity-70";
    }

    if (tone === "danger") {
        return "w-full rounded-2xl border border-rose-300/20 bg-rose-500/10 px-4 py-3 text-sm font-semibold text-rose-50 transition hover:border-rose-200/35 hover:bg-rose-500/15 disabled:cursor-wait disabled:opacity-70";
    }

    return "w-full rounded-2xl border border-cyan-200/25 bg-cyan-400/15 px-4 py-3 text-sm font-semibold text-cyan-50 transition hover:border-cyan-100/40 hover:bg-cyan-400/20 disabled:cursor-wait disabled:opacity-70";
}

function ProfileActionButton({ action }) {
    const isDisabled = Boolean(action.disabled);
    const isLoading = isDisabled && Boolean(action.loadingLabel);

    return (
        <button
            type="button"
            onClick={action.onClick}
            disabled={isDisabled}
            className={resolveProfileActionClassName(action.tone, isDisabled, isLoading)}
        >
            {isLoading ? action.loadingLabel : action.label}
        </button>
    );
}

function ProfileActionsSection({ actions, feedback = null }) {
    if (!actions.length) {
        return null;
    }

    const columnsClassName =
        actions.length === 1 ? "" : actions.length === 2 ? "sm:grid-cols-2" : "sm:grid-cols-2 lg:grid-cols-3";

    return (
        <section className="rounded-3xl border border-white/10 bg-slate-950/35 p-6 shadow-xl shadow-black/20 backdrop-blur">
            <p className="text-xs font-semibold uppercase tracking-[0.3em] text-slate-400">Действия</p>
            <div className={`mt-5 grid gap-3 ${columnsClassName}`}>
                {actions.map((action) => (
                    <ProfileActionButton key={action.key} action={action} />
                ))}
            </div>

            {feedback ? (
                <div className="mt-4">
                    <StatusMessage feedback={feedback} />
                </div>
            ) : null}
        </section>
    );
}

export function ProfilePage() {
    const navigate = useNavigate();
    const { session, signOut } = useAuth();
    const { userId } = useParams();
    const avatarInputRef = useRef(null);
    const [avatarFeedback, setAvatarFeedback] = useState(null);
    const [disabledFeedback, setDisabledFeedback] = useState(null);
    const [uploadProfileAvatar, { isLoading: isAvatarUploading }] = useUploadProfileAvatarMutation();
    const [setProfileDisabled, { isLoading: isProfileDisabledUpdating }] = useSetProfileDisabledMutation();
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
    const isCoordinatorOrAdmin = session?.role === "coordinator" || session?.role === "admin";
    const canUploadAvatar = !isMemberProfile;
    const isOwnProfile = !isMemberProfile || session?.user_id === profile?.user_id;
    const isProfileDisabled = Boolean(profile?.disabled);
    const canManageDisabled = Boolean(
        isMemberProfile && isCoordinatorOrAdmin && profile?.user_id && session?.user_id !== profile.user_id,
    );

    useEffect(() => {
        setAvatarFeedback(null);
        setDisabledFeedback(null);
    }, [userId]);

    function handleBack() {
        navigate(-1);
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

    function handleOpenChangePassword() {
        navigate(routePaths.changePassword);
    }

    function handleAvatarUploadClick() {
        if (!canUploadAvatar || isAvatarUploading) {
            return;
        }

        avatarInputRef.current?.click();
    }

    async function handleAvatarFileChange(event) {
        const selectedFile = event.target.files?.[0];
        event.target.value = "";

        if (!selectedFile) {
            return;
        }

        if (!selectedFile.type.startsWith("image/")) {
            setAvatarFeedback({
                tone: "error",
                message: "Можно загружать только изображения.",
            });
            return;
        }

        if (selectedFile.size > 10 * 1024 * 1024) {
            setAvatarFeedback({
                tone: "error",
                message: "Фото профиля должно быть меньше 10 МБ.",
            });
            return;
        }

        setAvatarFeedback(null);

        try {
            await uploadProfileAvatar({ file: selectedFile }).unwrap();
            setAvatarFeedback({
                tone: "success",
                message: "Фото профиля обновлено.",
            });
        } catch (error) {
            setAvatarFeedback({
                tone: "error",
                message: resolveErrorMessage(error, "Не удалось загрузить фото профиля."),
            });
        }
    }

    async function handleToggleProfileDisabled() {
        if (!canManageDisabled || !profile?.user_id || isProfileDisabledUpdating) {
            return;
        }

        const nextDisabledValue = !isProfileDisabled;
        setDisabledFeedback(null);

        try {
            await setProfileDisabled({
                disabled: nextDisabledValue,
                userId: profile.user_id,
            }).unwrap();
            setDisabledFeedback({
                tone: "success",
                message: nextDisabledValue ? "Профиль сотрудника отключен." : "Профиль сотрудника снова активен.",
            });
        } catch (error) {
            setDisabledFeedback({
                tone: "error",
                message: resolveErrorMessage(
                    error,
                    nextDisabledValue ? "Не удалось отключить сотрудника." : "Не удалось включить сотрудника.",
                ),
            });
        }
    }

    const profileActions = [];

    if (isOwnProfile) {
        profileActions.push(
            {
                disabled: !canOpenEditor,
                key: "editor",
                label: "Редактор",
                onClick: handleOpenEditor,
                tone: "primary",
            },
            {
                key: "change-password",
                label: "Сменить пароль",
                onClick: handleOpenChangePassword,
                tone: "warning",
            },
            {
                key: "sign-out",
                label: "Выйти из аккаунта",
                onClick: handleSignOut,
                tone: "danger",
            },
        );
    }

    if (canManageDisabled) {
        profileActions.push({
            disabled: isProfileDisabledUpdating,
            key: "disable-user",
            label: isProfileDisabled ? "Включить сотрудника" : "Отключить сотрудника",
            loadingLabel: "Сохраняем статус...",
            onClick: handleToggleProfileDisabled,
            tone: "danger",
        });
    }

    return (
        <PageShell>
            <section className="w-full space-y-6">
                <ProfileHeader isMemberProfile={isMemberProfile} onBack={handleBack} />

                {isLoading ? (
                    <section className="app-subtle-notice">
                        <p className="text-sm text-slate-300">
                            {isMemberProfile ? "Загружаем профиль сотрудника..." : "Загружаем профиль..."}
                        </p>
                    </section>
                ) : null}

                {hasError ? <StatusMessage feedback={{ message: errorMessage, tone: "error" }} /> : null}

                {!isLoading && !hasError && profile ? (
                    <>
                        <input
                            ref={avatarInputRef}
                            type="file"
                            accept="image/png,image/jpeg,image/gif,image/webp"
                            className="hidden"
                            onChange={handleAvatarFileChange}
                        />

                        <section className="grid gap-6 xl:grid-cols-[1.1fr_0.9fr]">
                            <article className="rounded-lg border border-white/10 bg-white/10 p-6 shadow-2xl shadow-[#6A3BF2]/20 backdrop-blur-xl sm:p-8">
                                <div className="flex flex-col gap-6 sm:flex-row sm:items-start sm:justify-between">
                                    <div className="flex items-start gap-4 sm:gap-5">
                                        <div className="space-y-3">
                                            <ProfileAvatar
                                                canUpload={canUploadAvatar}
                                                isUploading={isAvatarUploading}
                                                logo={profile.logo}
                                                name={displayName}
                                                onUploadClick={handleAvatarUploadClick}
                                            />

                                            {canUploadAvatar ? (
                                                <p className="max-w-32 text-center text-xs font-medium text-slate-300">
                                                    {isAvatarUploading
                                                        ? "Загружаем фото..."
                                                        : "Нажмите на фото, чтобы обновить аватар"}
                                                </p>
                                            ) : null}
                                        </div>

                                        <div className="min-w-0">
                                            <p className="text-xs font-semibold uppercase tracking-[0.32em] text-cyan-200">
                                                {isMemberProfile ? "Карточка сотрудника" : "Личный кабинет"}
                                            </p>
                                            <h2 className="mt-3 text-3xl font-bold tracking-tight text-white sm:text-4xl">
                                                {displayName}
                                            </h2>
                                            <p className="mt-2 text-lg font-semibold text-slate-200">
                                                {departmentTitle}
                                            </p>
                                        </div>
                                    </div>

                                    <div className="shrink-0">
                                        <ProfileActivityIndicator
                                            isDisabled={isProfileDisabled}
                                            latestTicketStatus={profile.latestTicketStatus}
                                        />
                                    </div>
                                </div>

                                <div className="mt-8 grid gap-3 sm:grid-cols-2">
                                    <ProfileInfoRow label="Email" value={emailValue} />
                                    <ProfileInfoRow label="Телефон" value={phoneValue} />
                                </div>

                                {isProfileDisabled ? <ProfileDisabledNotice isMemberProfile={isMemberProfile} /> : null}

                                {avatarFeedback ? (
                                    <div className="mt-6">
                                        <StatusMessage feedback={avatarFeedback} />
                                    </div>
                                ) : null}
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

                        <ProfileActionsSection actions={profileActions} feedback={disabledFeedback} />
                    </>
                ) : null}
            </section>
        </PageShell>
    );
}
