import { useEffect, useState } from "react";
import { PageShell } from "../../../shared/ui/page-shell";
import { StatusMessage } from "../../../shared/ui/status-message";

export const editorFieldClassName =
    "mt-3 w-full rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3 text-sm text-white outline-none transition placeholder:text-slate-500 focus:border-cyan-200/40 focus:bg-slate-950/60";

export const editorTextareaClassName = `${editorFieldClassName} resize-y`;

export const editorSelectClassName = "min-h-[3.25rem] bg-slate-950/40 px-4 py-3 text-sm";

export function useSyncedSidebarHeight(targetRef) {
    const [height, setHeight] = useState(null);

    useEffect(() => {
        if (typeof window === "undefined") {
            return undefined;
        }

        const mediaQuery = window.matchMedia("(min-width: 1280px)");

        function updateHeight() {
            if (!mediaQuery.matches) {
                setHeight(null);
                return;
            }

            const nextHeight = targetRef.current?.offsetHeight;
            setHeight(Number.isFinite(nextHeight) ? nextHeight : null);
        }

        updateHeight();

        const observer = new ResizeObserver(() => {
            updateHeight();
        });

        if (targetRef.current) {
            observer.observe(targetRef.current);
        }

        function handleMediaChange() {
            updateHeight();
        }

        mediaQuery.addEventListener("change", handleMediaChange);
        window.addEventListener("resize", updateHeight);

        return () => {
            observer.disconnect();
            mediaQuery.removeEventListener("change", handleMediaChange);
            window.removeEventListener("resize", updateHeight);
        };
    }, [targetRef]);

    return height;
}

export function BackButton({ onClick }) {
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

export function DashboardButton({ onClick }) {
    return (
        <button
            type="button"
            onClick={onClick}
            aria-label="Перейти в дэшборд"
            className="inline-flex h-11 w-11 items-center justify-center rounded-2xl bg-[#6A3BF2] text-white shadow-lg shadow-[#6A3BF2]/35 transition hover:bg-[#7C52F5]"
        >
            <svg
                xmlns="http://www.w3.org/2000/svg"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
                className="h-5 w-5"
                aria-hidden="true"
            >
                <rect x="3.5" y="3.5" width="7" height="7" rx="1.5" />
                <rect x="13.5" y="3.5" width="7" height="5" rx="1.5" />
                <rect x="13.5" y="11.5" width="7" height="9" rx="1.5" />
                <rect x="3.5" y="13.5" width="7" height="7" rx="1.5" />
            </svg>
        </button>
    );
}

export function SummaryCard({ label, value }) {
    return (
        <div className="rounded-3xl border border-white/10 bg-slate-950/35 p-4 shadow-lg shadow-black/15">
            <p className="text-xs font-semibold uppercase tracking-[0.24em] text-slate-500">{label}</p>
            <p className="mt-3 text-xl font-semibold text-slate-100">{value}</p>
        </div>
    );
}

export function EditorContextPanel({ children, footer = null, height = null, title }) {
    return (
        <aside
            style={height ? { height: `${height}px` } : undefined}
            className="self-start rounded-3xl border border-white/10 bg-slate-950/35 p-5"
        >
            <div className="space-y-5 lg:sticky lg:top-6">
                <div>
                    <p className="text-xs font-semibold uppercase tracking-[0.22em] text-slate-400">{title}</p>
                </div>
                {children}
                {footer}
            </div>
        </aside>
    );
}

export function EditorContextSection({ children, title }) {
    return (
        <div>
            <p className="text-xs font-semibold uppercase tracking-[0.22em] text-slate-400">{title}</p>
            <div className="mt-4 space-y-3 text-sm text-slate-300">{children}</div>
        </div>
    );
}

export function EditorContextItem({ label, value }) {
    return (
        <p>
            <span className="text-slate-500">{label}:</span> {value}
        </p>
    );
}

export function EditorFormField({ label, children, hint }) {
    return (
        <label className="block">
            <span className="text-xs font-semibold uppercase tracking-[0.22em] text-slate-400">{label}</span>
            {children}
            {hint ? <span className="mt-2 block text-sm text-slate-500">{hint}</span> : null}
        </label>
    );
}

export function EditorPageHeader({ action, description, leadingAction, textAlign = "center", title }) {
    const isLeftAligned = textAlign === "left";
    const hasLeadingAction = Boolean(leadingAction);
    const hasAction = Boolean(action);
    let containerClassName = "grid items-center gap-4";

    if (hasLeadingAction && hasAction) {
        containerClassName += " grid-cols-[2.75rem_minmax(0,1fr)_2.75rem]";
    } else if (hasLeadingAction) {
        containerClassName += " grid-cols-[2.75rem_minmax(0,1fr)_2.75rem]";
    } else if (hasAction) {
        containerClassName += " grid-cols-[minmax(0,1fr)_2.75rem]";
    }

    return (
        <header className="rounded-3xl border border-white/10 bg-slate-950/35 p-6 shadow-xl shadow-black/20 backdrop-blur">
            <div className={containerClassName}>
                {hasLeadingAction ? (
                    <div className="flex min-h-11 w-11 items-center justify-start">{leadingAction}</div>
                ) : null}

                <div
                    className={`flex flex-col ${isLeftAligned ? "items-start text-left" : "items-center text-center"}`}
                >
                    <p className="text-xs font-semibold uppercase tracking-[0.3em] text-slate-400">Редактор</p>
                    <h1 className="mt-3 text-3xl font-bold tracking-tight text-white sm:text-4xl">{title}</h1>
                    {description ? <p className="mt-3 max-w-2xl text-base text-slate-300">{description}</p> : null}
                </div>

                {hasLeadingAction && !hasAction ? <div className="min-h-11 w-11" aria-hidden="true" /> : null}

                {hasAction ? <div className="flex min-h-11 w-11 items-center justify-end">{action}</div> : null}
            </div>
        </header>
    );
}

export function EditorWorkspace({ sidebar, children }) {
    return (
        <section className="grid gap-6 xl:grid-cols-[360px_minmax(0,1fr)]">
            {sidebar}
            {children}
        </section>
    );
}

export function EditorSidebar({ footer, height, children }) {
    return (
        <aside
            style={height ? { height: `${height}px` } : undefined}
            className="grid min-h-0 grid-rows-[auto_minmax(0,1fr)_auto] gap-4 overflow-hidden rounded-[2rem] border border-white/10 bg-white/10 p-5 shadow-2xl shadow-[#6A3BF2]/15 backdrop-blur-xl"
        >
            {children}
            <p className="self-end text-xs text-slate-500">{footer}</p>
        </aside>
    );
}

export function EditorListHeader({ title }) {
    return (
        <div>
            <p className="text-xs font-semibold uppercase tracking-[0.32em] text-cyan-200">Список</p>
            <h2 className="mt-3 text-2xl font-bold tracking-tight text-white">{title}</h2>
        </div>
    );
}

export function EditorSearchField({ onChange, placeholder, value }) {
    return (
        <label className="block">
            <span className="text-xs font-semibold uppercase tracking-[0.22em] text-slate-400">Поиск</span>
            <input
                type="search"
                value={value}
                onChange={onChange}
                placeholder={placeholder}
                className={editorFieldClassName}
            />
        </label>
    );
}

export function EditorPane({ editorPaneRef, children }) {
    return (
        <section
            ref={editorPaneRef}
            className="space-y-6 rounded-[2rem] border border-white/10 bg-slate-950/30 p-6 shadow-2xl shadow-black/20 backdrop-blur-xl"
        >
            {children}
        </section>
    );
}

export function EditorNoticeCard({ dashed = false, message }) {
    return (
        <div
            className={`rounded-3xl p-8 text-slate-300 ${
                dashed ? "border border-dashed border-white/15 bg-white/5" : "border border-white/10 bg-white/5"
            }`}
        >
            {message}
        </div>
    );
}

export function EditorListError({ error, fallbackMessage }) {
    if (!error) {
        return null;
    }

    return (
        <StatusMessage
            feedback={{
                message: typeof error?.data === "string" ? error.data : fallbackMessage,
                tone: "error",
            }}
        />
    );
}

export function EditorRecordHeader({ id, isDirty, isSaving, onSave, title, titleLabel }) {
    return (
        <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
            <div>
                <p className="text-xs font-semibold uppercase tracking-[0.32em] text-cyan-200">{titleLabel}</p>
                <h2 className="mt-3 text-3xl font-bold tracking-tight text-white">{title}</h2>
                <p className="mt-3 text-sm text-slate-400">ID: {id}</p>
            </div>

            <div className="flex flex-wrap items-center justify-end gap-3">
                <EditorSaveStateBadge isDirty={isDirty} />
                <button
                    type="button"
                    onClick={onSave}
                    disabled={isSaving || !isDirty}
                    className={`rounded-2xl px-5 py-3 text-sm font-semibold transition ${
                        isSaving || !isDirty
                            ? "cursor-not-allowed border border-white/10 bg-white/5 text-slate-500"
                            : "border border-cyan-200/30 bg-cyan-400/15 text-cyan-50 hover:border-cyan-100/45 hover:bg-cyan-400/20"
                    }`}
                >
                    {isSaving ? "Сохраняем..." : "Сохранить"}
                </button>
            </div>
        </div>
    );
}

export function EditorSaveStateBadge({ isDirty }) {
    if (isDirty) {
        return (
            <span className="rounded-full border border-amber-300/25 bg-amber-400/10 px-4 py-2 text-sm font-semibold text-amber-100">
                Есть несохраненные изменения
            </span>
        );
    }

    return (
        <span className="rounded-full border border-emerald-300/20 bg-emerald-400/10 px-4 py-2 text-sm font-semibold text-emerald-100">
            Все изменения сохранены
        </span>
    );
}

export function EditorNoAccess({ onBack }) {
    return (
        <PageShell>
            <section className="w-full space-y-6">
                <EditorPageHeader title="Нет доступа" leadingAction={<BackButton onClick={onBack} />} />

                <section className="rounded-3xl border border-rose-300/20 bg-rose-500/10 p-6 shadow-xl shadow-black/20 backdrop-blur">
                    <p className="text-base text-rose-50">
                        Редактор пока доступен только координаторам и администраторам.
                    </p>
                </section>
            </section>
        </PageShell>
    );
}

export function EditorEntityCard({ badge, description, disabled = false, onClick, title }) {
    const classes = disabled
        ? "cursor-not-allowed border-white/10 bg-white/5 text-slate-500 opacity-80"
        : "border-cyan-200/20 bg-cyan-400/10 text-slate-100 hover:border-cyan-100/40 hover:bg-cyan-400/15";

    return (
        <button
            type="button"
            onClick={onClick}
            disabled={disabled}
            className={`rounded-[2rem] border p-6 text-left shadow-xl shadow-black/15 transition ${classes}`}
        >
            <div className="flex items-start justify-between gap-4">
                <div>
                    <p className="text-2xl font-bold tracking-tight">{title}</p>
                    <p className="mt-3 max-w-md text-sm text-slate-300">{description}</p>
                </div>
                {badge ? (
                    <span className="rounded-full border border-white/10 bg-black/15 px-3 py-1 text-xs font-semibold uppercase tracking-[0.2em] text-slate-200">
                        {badge}
                    </span>
                ) : null}
            </div>
        </button>
    );
}
