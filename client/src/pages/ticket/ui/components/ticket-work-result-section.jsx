import { useEffect, useState } from "react";
import { loadTicketAttachmentPreviewUrl } from "../../../../shared/api/tickets-api";

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
    const [previewUrls, setPreviewUrls] = useState({});
    const attachments = Array.isArray(ticket?.attachments) ? ticket.attachments : [];
    const previewLoadKey = attachments
        .map((attachment) => [attachment.id, attachment.downloadUrl, attachment.mediaType, attachment.ext].join(":"))
        .join("|");

    useEffect(() => {
        let isCancelled = false;
        const objectUrls = [];

        setPreviewUrls({});

        async function loadPreviews() {
            const previewableAttachments = attachments.filter((attachment) =>
                isPreviewableAttachment(attachment),
            );

            if (previewableAttachments.length === 0) {
                return;
            }

            const nextPreviewUrls = {};
            await Promise.all(
                previewableAttachments.map(async (attachment) => {
                    try {
                        const previewUrl = await loadTicketAttachmentPreviewUrl(attachment.downloadUrl);
                        if (!previewUrl) {
                            return;
                        }

                        if (isCancelled) {
                            window.URL.revokeObjectURL(previewUrl);
                            return;
                        }

                        objectUrls.push(previewUrl);
                        nextPreviewUrls[attachment.id] = previewUrl;
                    } catch {}
                }),
            );

            if (!isCancelled) {
                setPreviewUrls(nextPreviewUrls);
            }
        }

        void loadPreviews();

        return () => {
            isCancelled = true;
            objectUrls.forEach((objectUrl) => {
                window.URL.revokeObjectURL(objectUrl);
            });
        };
    }, [previewLoadKey]);

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

                <p className="mt-4 text-2xl leading-relaxed text-slate-200">{ticket.result}</p>
            </div>

            {attachments.length > 0 ? (
                <div className="space-y-3">
                    <div className="-mx-5 overflow-x-auto pb-2 sm:-mx-6">
                        <div className="flex min-w-full gap-3 px-5 sm:px-6">
                            {attachments.map((attachment) => {
                                const previewUrl = previewUrls[attachment.id];

                                return (
                                    <button
                                        key={attachment.id}
                                        type="button"
                                        onClick={() => {
                                            void handleDownload(attachment);
                                        }}
                                        disabled={!attachment.downloadUrl}
                                        className="flex h-60 w-44 shrink-0 flex-col overflow-hidden rounded-2xl border border-white/15 bg-slate-900/25 text-left transition hover:bg-slate-900/40 disabled:cursor-not-allowed disabled:opacity-60"
                                    >
                                        <div className="flex h-44 w-44 items-center justify-center bg-slate-950/40 p-3">
                                            {previewUrl ? (
                                                <img
                                                    src={previewUrl}
                                                    alt={attachment.name}
                                                    className="h-full w-full object-contain"
                                                    loading="lazy"
                                                />
                                            ) : (
                                                <AttachmentFallback attachment={attachment} />
                                            )}
                                        </div>
                                        <div className="flex min-h-0 flex-1 flex-col justify-center space-y-1 border-t border-white/10 px-3 py-2">
                                            <p className="truncate text-sm font-semibold text-slate-100">{attachment.name}</p>
                                            <p className="truncate text-xs text-slate-300">
                                                {attachment.mediaType || "Файл"} · {formatAttachmentSize(attachment.sizeBytes)}
                                            </p>
                                        </div>
                                    </button>
                                );
                            })}
                        </div>
                    </div>

                    <p className="text-center text-xs uppercase tracking-[0.2em] text-emerald-100/80">
                        Нажмите на вложение, чтобы скачать
                    </p>
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

function isPreviewableAttachment(attachment) {
    if (!attachment?.downloadUrl) {
        return false;
    }

    const mediaType = attachment?.mediaType?.toLowerCase() || "";
    if (mediaType.startsWith("image/")) {
        return true;
    }

    const ext = attachment?.ext?.toLowerCase() || "";
    return ["avif", "gif", "heic", "jpeg", "jpg", "png", "svg", "webp"].includes(ext);
}

function AttachmentFallback({ attachment }) {
    const ext = attachment?.ext?.toUpperCase() || "FILE";

    return (
        <div className="flex h-full w-full flex-col items-center justify-center rounded-xl border border-dashed border-white/15 bg-white/5 px-3 text-center">
            <svg
                xmlns="http://www.w3.org/2000/svg"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="1.7"
                strokeLinecap="round"
                strokeLinejoin="round"
                className="h-9 w-9 text-slate-300"
                aria-hidden="true"
            >
                <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
                <path d="M14 2v6h6" />
                <path d="M9 15h6" />
                <path d="M9 11h2" />
            </svg>
            <span className="mt-3 text-xs font-semibold uppercase tracking-[0.24em] text-slate-200">{ext}</span>
        </div>
    );
}
