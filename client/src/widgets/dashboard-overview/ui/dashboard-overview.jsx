export function DashboardOverview({ isMemberProfile = false, session }) {
  const eyebrow = isMemberProfile ? "Engineer Profile" : "Mobile Engineer V3";
  const description = isMemberProfile
    ? "Карточка сотрудника загружается из защищенного API и показывает основные данные инженера."
    : "Ваша сессия активна. Эта страница профиля получает защищенные данные учетной записи по access token, а ротация refresh token доступна без перехода на другие страницы.";

  return (
    <div className="rounded-[2rem] border border-white/10 bg-white/10 p-8 shadow-2xl shadow-[#6A3BF2]/25 backdrop-blur-xl">
      <p className="text-xs font-semibold uppercase tracking-[0.45em] text-[#6A3BF2]">
        {eyebrow}
      </p>
      <h1 className="mt-4 text-4xl font-bold tracking-tight sm:text-5xl">
        Профиль
      </h1>
      <p className="mt-3 max-w-xl text-sm text-slate-300 sm:text-base">
        {description}
      </p>

      <div className="mt-8 grid gap-4 sm:grid-cols-2">
        <article className="rounded-3xl border border-white/10 bg-slate-950/30 p-5">
          <p className="text-xs font-semibold uppercase tracking-[0.3em] text-slate-400">
            Имя
          </p>
          <p className="mt-3 text-2xl font-semibold text-slate-50">
            {session?.name || "Не указано"}
          </p>
        </article>

        <article className="rounded-3xl border border-white/10 bg-slate-950/30 p-5">
          <p className="text-xs font-semibold uppercase tracking-[0.3em] text-slate-400">
            Имя пользователя
          </p>
          <p className="mt-3 text-2xl font-semibold text-slate-50">
            {session?.username}
          </p>
        </article>

        <article className="rounded-3xl border border-white/10 bg-slate-950/30 p-5">
          <p className="text-xs font-semibold uppercase tracking-[0.3em] text-slate-400">
            Email
          </p>
          <p className="mt-3 break-all text-sm text-slate-200">
            {session?.email || "Не указано"}
          </p>
        </article>

        <article className="rounded-3xl border border-white/10 bg-slate-950/30 p-5">
          <p className="text-xs font-semibold uppercase tracking-[0.3em] text-slate-400">
            ID пользователя
          </p>
          <p className="mt-3 break-all text-sm text-slate-200">
            {session?.user_id}
          </p>
        </article>

        <article className="rounded-3xl border border-white/10 bg-slate-950/30 p-5 sm:col-span-2">
          <p className="text-xs font-semibold uppercase tracking-[0.3em] text-slate-400">
            Отдел
          </p>
          <p className="mt-3 text-lg font-semibold text-slate-100">
            {session?.department || "Не назначен"}
          </p>
        </article>

        <article className="rounded-3xl border border-white/10 bg-slate-950/30 p-5 sm:col-span-2">
          <p className="text-xs font-semibold uppercase tracking-[0.3em] text-slate-400">
            {isMemberProfile ? "Роль" : "О сессии"}
          </p>
          <p className="mt-3 text-sm leading-7 text-slate-300">
            {isMemberProfile
              ? session?.role || "Не указано"
              : "Access token проверяется по JWT-подписи. Refresh token ротируется на сервере и заменяется после каждого запроса обновления."}
          </p>
        </article>
      </div>
    </div>
  );
}
