import { useState } from "react";
import { MOCK_WORK_RESULT } from "../../model/ticket-page-model";

function formatAttachmentSize(sizeBytes) {
    if (!sizeBytes) {
        return "0 B";
    }

    if (sizeBytes < 1024) {
        return `${sizeBytes} B`;
    }

    if (sizeBytes < 1024 * 1024) {
        return `${Math.round(sizeBytes / 1024)} KB`;
    }

    return `${(sizeBytes / (1024 * 1024)).toFixed(1)} MB`;
}

export function TicketWorkResultSection({ ticket, workDuration, onDownloadAttachment }) {
    const [downloadError, setDownloadError] = useState("");
    const attachments = Array.isArray(ticket?.attachments) ? ticket.attachments : [];

    async function handleDownload(attachment) {
        if (!onDownloadAttachment || !attachment?.downloadUrl) {
            return;
        }

        setDownloadError("");

        try {
            await onDownloadAttachment(attachment);
        } catch (error) {
            setDownloadError(error?.message || "Не удалось скачать вложение.");
        }
    }

    return (
        <section className="space-y-3 rounded-3xl border border-emerald-300/25 bg-emerald-500/10 p-5 sm:p-6">
            <div className="flex items-center justify-between gap-4">
                <h2 className="text-3xl font-semibold tracking-tight text-emerald-100">Результат работы</h2>
                <p className="inline-flex items-center gap-2 text-2xl font-semibold text-emerald-100">
                    <svg
                        xmlns="http://www.w3.org/2000/svg"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        strokeWidth="2.2"
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        className="h-6 w-6"
                        aria-hidden="true"
                    >
                        <circle cx="12" cy="12" r="8" />
                        <path d="M12 8v5l3 2" />
                    </svg>
                    {workDuration}
                </p>
            </div>

            <div className="rounded-2xl border border-white/15 bg-white/10 p-5 shadow-lg shadow-black/15">
                <div className="flex items-start gap-4">
                    <span className="inline-flex h-12 w-12 shrink-0 items-center justify-center rounded-full bg-slate-950 text-sm font-semibold text-slate-100">
                        {ticket.executorName ? ticket.executorName.trim().charAt(0).toUpperCase() : "?"}
                    </span>
                    <div className="min-w-0">
                        <p className="text-2xl font-semibold leading-tight text-slate-100">
                            {ticket.executorName || "Исполнитель не назначен"}
                        </p>
                        <p className="text-2xl text-slate-400">{ticket.executorDepartment || "Отдел не указан"}</p>
                    </div>
                </div>

                <p className="mt-4 text-2xl leading-relaxed text-slate-200">{ticket.result || MOCK_WORK_RESULT}</p>
            </div>

            {attachments.length > 0 ? (
                <div className="grid gap-2 sm:grid-cols-2">
                    {attachments.map((attachment) => (
                        <button
                            key={attachment.id}
                            type="button"
                            onClick={() => {
                                void handleDownload(attachment);
                            }}
                            disabled={!attachment.downloadUrl}
                            className="flex items-center justify-between gap-4 rounded-2xl border border-white/15 bg-slate-900/25 px-4 py-3 text-left transition hover:bg-slate-900/40 disabled:cursor-not-allowed disabled:opacity-60"
                        >
                            <div className="min-w-0">
                                <p className="truncate text-sm font-semibold text-slate-100">{attachment.name}</p>
                                <p className="text-xs text-slate-300">
                                    {attachment.mediaType || "Файл"} · {formatAttachmentSize(attachment.sizeBytes)}
                                </p>
                            </div>
                            <span className="shrink-0 text-xs font-semibold uppercase tracking-[0.2em] text-emerald-100">
                                Скачать
                            </span>
                        </button>
                    ))}
                </div>
            ) : (
                <div className="rounded-2xl border border-dashed border-white/15 bg-slate-900/15 px-4 py-5 text-sm text-slate-200">
                    Вложения появятся после загрузки отчета.
                </div>
            )}

            {downloadError ? <p className="text-xs text-rose-100">{downloadError}</p> : null}
        </section>
    );
}
