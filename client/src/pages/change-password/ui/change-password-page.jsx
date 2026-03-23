import { useState } from "react";
import { useNavigate } from "react-router";
import { routePaths } from "../../../shared/config/routes";
import { useChangePasswordMutation } from "../../../shared/api/tickets-api";
import { PageShell } from "../../../shared/ui/page-shell";
import { StatusMessage } from "../../../shared/ui/status-message";

const initialForm = {
  currentPassword: "",
  newPassword: "",
  confirmPassword: "",
};

function resolveErrorMessage(error, fallbackMessage) {
  if (typeof error?.data === "string" && error.data.trim()) {
    return error.data;
  }

  if (typeof error?.error === "string" && error.error.trim()) {
    return error.error;
  }

  return fallbackMessage;
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

export function ChangePasswordPage() {
  const navigate = useNavigate();
  const [form, setForm] = useState(initialForm);
  const [feedback, setFeedback] = useState(null);
  const [changePassword, { isLoading }] = useChangePasswordMutation();

  function handleChange(event) {
    const { name, value } = event.target;

    setForm((current) => ({
      ...current,
      [name]: value,
    }));
  }

  function handleBack() {
    navigate(routePaths.profile);
  }

  async function handleSubmit(event) {
    event.preventDefault();

    const currentPassword = form.currentPassword.trim();
    const newPassword = form.newPassword.trim();
    const confirmPassword = form.confirmPassword.trim();

    if (!currentPassword || !newPassword || !confirmPassword) {
      setFeedback({
        tone: "error",
        message: "Заполните текущий пароль, новый пароль и подтверждение.",
      });
      return;
    }

    if (newPassword.length < 8) {
      setFeedback({
        tone: "error",
        message: "Новый пароль должен содержать минимум 8 символов.",
      });
      return;
    }

    if (newPassword !== confirmPassword) {
      setFeedback({
        tone: "error",
        message: "Подтверждение пароля не совпадает.",
      });
      return;
    }

    if (currentPassword === newPassword) {
      setFeedback({
        tone: "error",
        message: "Новый пароль должен отличаться от текущего.",
      });
      return;
    }

    setFeedback(null);

    try {
      await changePassword({
        currentPassword,
        newPassword,
      }).unwrap();

      setForm(initialForm);
      setFeedback({
        tone: "success",
        message: "Пароль обновлен. Теперь можно вернуться в профиль.",
      });
    } catch (error) {
      setFeedback({
        tone: "error",
        message: resolveErrorMessage(error, "Не удалось сменить пароль."),
      });
    }
  }

  return (
    <PageShell>
      <section className="w-full max-w-3xl space-y-6">
        <header className="rounded-3xl border border-white/10 bg-slate-950/35 p-6 shadow-xl shadow-black/20 backdrop-blur">
          <div className="flex items-center justify-between gap-4">
            <div>
              <p className="text-xs font-semibold uppercase tracking-[0.3em] text-slate-400">Безопасность</p>
              <h1 className="mt-3 text-3xl font-bold tracking-tight text-white sm:text-4xl">Смена пароля</h1>
              <p className="mt-3 max-w-2xl text-sm text-slate-300">
                Введите текущий пароль и задайте новый. После сохранения входить заново не нужно.
              </p>
            </div>
            <BackButton onClick={handleBack} />
          </div>
        </header>

        <section className="rounded-[2rem] border border-white/10 bg-white/10 p-8 shadow-2xl shadow-[#6A3BF2]/20 backdrop-blur-xl">
          <form className="space-y-5" onSubmit={handleSubmit}>
            <label className="block">
              <span className="mb-2 block text-sm font-medium text-slate-200">Текущий пароль</span>
              <input
                type="password"
                name="currentPassword"
                value={form.currentPassword}
                onChange={handleChange}
                autoComplete="current-password"
                placeholder="Введите текущий пароль"
                className="w-full rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3 text-base text-slate-100 outline-none transition focus:border-[#6A3BF2] focus:ring-2 focus:ring-[#6A3BF2]/30"
              />
            </label>

            <label className="block">
              <span className="mb-2 block text-sm font-medium text-slate-200">Новый пароль</span>
              <input
                type="password"
                name="newPassword"
                value={form.newPassword}
                onChange={handleChange}
                autoComplete="new-password"
                placeholder="Минимум 8 символов"
                className="w-full rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3 text-base text-slate-100 outline-none transition focus:border-[#6A3BF2] focus:ring-2 focus:ring-[#6A3BF2]/30"
              />
            </label>

            <label className="block">
              <span className="mb-2 block text-sm font-medium text-slate-200">Подтвердите новый пароль</span>
              <input
                type="password"
                name="confirmPassword"
                value={form.confirmPassword}
                onChange={handleChange}
                autoComplete="new-password"
                placeholder="Повторите новый пароль"
                className="w-full rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3 text-base text-slate-100 outline-none transition focus:border-[#6A3BF2] focus:ring-2 focus:ring-[#6A3BF2]/30"
              />
            </label>

            {feedback ? <StatusMessage feedback={feedback} /> : null}

            <div className="flex flex-col gap-3 sm:flex-row">
              <button
                type="submit"
                disabled={isLoading}
                className="flex-1 rounded-2xl bg-[#6A3BF2] px-4 py-3 text-sm font-semibold uppercase tracking-[0.25em] text-white transition hover:bg-[#7C52F5] disabled:cursor-not-allowed disabled:bg-[#6A3BF2]/60"
              >
                {isLoading ? "Сохраняем..." : "Сохранить пароль"}
              </button>
              <button
                type="button"
                onClick={handleBack}
                className="rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-sm font-semibold text-slate-200 transition hover:bg-white/10"
              >
                Вернуться в профиль
              </button>
            </div>
          </form>
        </section>
      </section>
    </PageShell>
  );
}
