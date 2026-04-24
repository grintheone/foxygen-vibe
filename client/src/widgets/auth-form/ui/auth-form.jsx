import { StatusMessage } from "../../../shared/ui/status-message";

export function AuthForm({ feedback, form, isSubmitting, onChange, onSubmit }) {
    return (
        <div className="rounded-lg border border-white/10 bg-white/10 p-8 shadow-2xl shadow-[#6A3BF2]/25 backdrop-blur-xl">
            <p className="text-xs font-semibold uppercase tracking-[0.45em] text-[#6A3BF2]">Мобильный инженер</p>
            <h1 className="mt-4 text-4xl font-bold tracking-tight sm:text-5xl">Вход</h1>
            <form className="mt-8 space-y-5" onSubmit={onSubmit}>
                <label className="block">
                    <span className="mb-2 block text-sm font-medium text-slate-200">Имя пользователя</span>
                    <input
                        type="text"
                        name="username"
                        value={form.username}
                        onChange={onChange}
                        autoComplete="username"
                        placeholder="mobile.engineer"
                        className="w-full rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3 text-base text-slate-100 outline-none transition focus:border-[#6A3BF2] focus:ring-2 focus:ring-[#6A3BF2]/30"
                    />
                </label>

                <label className="block">
                    <span className="mb-2 block text-sm font-medium text-slate-200">Пароль</span>
                    <input
                        type="password"
                        name="password"
                        value={form.password}
                        onChange={onChange}
                        autoComplete="current-password"
                        placeholder="Введите пароль"
                        className="w-full rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3 text-base text-slate-100 outline-none transition focus:border-[#6A3BF2] focus:ring-2 focus:ring-[#6A3BF2]/30"
                    />
                </label>

                {feedback.message ? <StatusMessage feedback={feedback} /> : null}

                <button
                    type="submit"
                    disabled={isSubmitting}
                    className="w-full rounded-2xl bg-[#6A3BF2] px-4 py-3 text-sm font-semibold uppercase tracking-[0.25em] text-white transition hover:bg-[#7C52F5] disabled:cursor-not-allowed disabled:bg-[#6A3BF2]/60"
                >
                    {isSubmitting ? "Выполняется..." : "Войти"}
                </button>
            </form>
        </div>
    );
}
