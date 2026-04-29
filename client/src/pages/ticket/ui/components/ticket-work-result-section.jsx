import { useEffect, useState } from "react";
import { loadTicketAttachmentPreviewUrl } from "../../../../shared/api/tickets-api";
import { UserAvatar } from "../../../../shared/ui/user-avatar";

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
        <section
            className="space-y-4 rounded-lg border border-[#AADB1E]/35 p-4 shadow-xl shadow-black/20 sm:p-5"
            style={{
                background: "linear-gradient(180deg, rgba(170, 219, 30, 0.22) 0%, rgba(16, 185, 129, 0.12) 100%)",
            }}
        >
            <div className="flex items-center justify-between gap-4">
                <h2 className="text-[16px] font-semibold tracking-tight text-[#AADB1E] sm:text-[18px] lg:text-[20px]">
                    Результат работы
                </h2>
                <p className="inline-flex items-center gap-2 text-[16px] font-semibold text-[#AADB1E] sm:text-[18px] lg:text-[20px]">
                    <svg
                        xmlns="http://www.w3.org/2000/svg"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        strokeWidth="1.9"
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        className="h-5 w-5 sm:h-6 sm:w-6"
                        aria-hidden="true"
                    >
                        <circle cx="12" cy="12" r="8" />
                        <path d="M12 8v5l3 2" />
                    </svg>
                    {workDuration}
                </p>
            </div>

            {attachments.length > 0 ? (
                <div className="space-y-3">
                    <div className="-mx-4 overflow-x-auto pb-2 sm:-mx-5">
                        <div className="flex min-w-full gap-3 px-4 sm:px-5">
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
                                        className="flex h-60 w-44 shrink-0 flex-col overflow-hidden rounded-lg border border-white/15 bg-slate-900/25 text-left transition hover:bg-slate-900/40 disabled:cursor-not-allowed disabled:opacity-60"
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
                </div>
            ) : (
                <div className="rounded-lg border border-dashed border-white/15 bg-slate-900/15 px-4 py-5 text-sm text-slate-200">
                    Вложения появятся после загрузки отчета.
                </div>
            )}

            <div className="border-t border-white/10 pt-4">
                <div className="flex items-start gap-4">
                    <UserAvatar
                        avatarUrl={ticket.executorAvatarUrl}
                        userId={ticket.executor}
                        name={ticket.executorName}
                        className="h-12 w-12"
                        iconClassName="h-6 w-6"
                    />
                    <div className="min-w-0">
                        <p className="text-[16px] font-semibold leading-snug tracking-tight text-slate-50 sm:text-[18px]">
                            {ticket.executorName || "Исполнитель не назначен"}
                        </p>
                        <p className="mt-1 text-[16px] leading-snug text-slate-200/80">
                            {ticket.executorDepartment || "Отдел не указан"}
                        </p>
                    </div>
                </div>

                <p className="mt-4 text-[16px] leading-relaxed text-slate-100">{ticket.result}</p>
            </div>

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
