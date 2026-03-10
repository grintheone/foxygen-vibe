import { useEffect, useRef, useState } from "react";

function createPreviewItems(files) {
    return files.map((file) => ({
        downloadUrl: "",
        file,
        id: `${file.name}-${file.lastModified}-${file.size}-${Math.random().toString(36).slice(2)}`,
        previewUrl: URL.createObjectURL(file),
        serverAttachmentId: "",
        uploadStatus: "pending",
    }));
}

function resolveUploadStatusClassName(status) {
    if (status === "submitting") {
        return "bg-sky-500/80";
    }

    if (status === "uploaded") {
        return "bg-emerald-500/80";
    }

    if (status === "failed") {
        return "bg-rose-500/80";
    }

    return "bg-slate-500/80";
}

function resolveUploadStatusLabel(status) {
    if (status === "submitting") {
        return "Сохраняем";
    }

    if (status === "uploaded") {
        return "Загружено";
    }

    if (status === "failed") {
        return "Ошибка";
    }

    return "Ожидает";
}

export function TicketReportSheet({
    clientName,
    deviceName,
    isOpen,
    isSubmitting,
    onClose,
    onDownloadAttachment,
    onSubmitClose,
    onUploadAttachment,
    resolvedReason,
    submitError,
    ticketNumber,
}) {
    const [isDoubleSigned, setIsDoubleSigned] = useState(false);
    const [isSubmitted, setIsSubmitted] = useState(false);
    const [localError, setLocalError] = useState("");
    const [mediaPreviews, setMediaPreviews] = useState([]);
    const [resultValue, setResultValue] = useState("");
    const mediaPreviewsRef = useRef(mediaPreviews);

    useEffect(() => {
        mediaPreviewsRef.current = mediaPreviews;
    }, [mediaPreviews]);

    useEffect(() => {
        return () => {
            mediaPreviewsRef.current.forEach((item) => URL.revokeObjectURL(item.previewUrl));
        };
    }, []);

    useEffect(() => {
        if (!isOpen) {
            setLocalError("");
            setIsDoubleSigned(false);
            setIsSubmitted(false);
            setMediaPreviews((previous) => {
                previous.forEach((item) => URL.revokeObjectURL(item.previewUrl));
                return [];
            });
            setResultValue("");
        }
    }, [isOpen]);

    if (!isOpen) {
        return null;
    }

    const reportSummary = [resolvedReason, deviceName, "в", clientName]
        .filter((part) => Boolean(part && String(part).trim()))
        .join(" ");

    function handleMediaSelect(event) {
        const selectedFiles = Array.from(event.target.files || []);
        if (selectedFiles.length === 0) {
            return;
        }

        const previewItems = createPreviewItems(selectedFiles);
        setMediaPreviews((previous) => [...previous, ...previewItems]);
        setIsSubmitted(false);
        setLocalError("");
        event.target.value = "";
    }

    function handleRemovePreview(previewId) {
        setMediaPreviews((previous) => {
            const previewToRemove = previous.find((item) => item.id === previewId);
            if (previewToRemove) {
                URL.revokeObjectURL(previewToRemove.previewUrl);
            }

            return previous.filter((item) => item.id !== previewId);
        });
    }

    async function handleDownloadPreview(preview) {
        if (!preview.serverAttachmentId || !onDownloadAttachment) {
            return;
        }

        setLocalError("");

        try {
            await onDownloadAttachment({
                downloadUrl: preview.downloadUrl,
                id: preview.serverAttachmentId,
                name: preview.file.name,
            });
        } catch (error) {
            setLocalError(error?.message || "Не удалось скачать вложение.");
        }
    }

    async function handleSubmit(event) {
        event.preventDefault();

        if (isSubmitting || isSubmitted) {
            return;
        }

        const trimmedResult = resultValue.trim();
        if (!trimmedResult) {
            setLocalError("Заполните поле результата работы.");
            return;
        }

        if (mediaPreviews.length === 0) {
            setLocalError("Добавьте хотя бы одну фотографию.");
            return;
        }

        setLocalError("");
        if (!onUploadAttachment) {
            setLocalError("Загрузка вложений не настроена.");
            return;
        }

        const previewsToUpload = mediaPreviewsRef.current.filter((item) => !item.serverAttachmentId);
        if (previewsToUpload.length > 0) {
            setMediaPreviews((previous) =>
                previous.map((item) =>
                    item.serverAttachmentId
                        ? item
                        : {
                              ...item,
                              uploadStatus: "submitting",
                          },
                ),
            );
        }

        let uploadFailed = false;
        for (const item of previewsToUpload) {
            try {
                const uploadedAttachment = await onUploadAttachment(item.file);
                setMediaPreviews((previous) =>
                    previous.map((current) =>
                        current.id !== item.id
                            ? current
                            : {
                                  ...current,
                                  downloadUrl: uploadedAttachment?.downloadUrl || "",
                                  serverAttachmentId: uploadedAttachment?.id || "",
                                  uploadStatus: "uploaded",
                              },
                    ),
                );
            } catch {
                uploadFailed = true;
                setMediaPreviews((previous) =>
                    previous.map((current) =>
                        current.id !== item.id
                            ? current
                            : {
                                  ...current,
                                  uploadStatus: "failed",
                              },
                    ),
                );
            }
        }

        if (uploadFailed) {
            setLocalError("Не удалось загрузить все вложения.");
            return;
        }

        try {
            await onSubmitClose({
                closed_at: new Date().toISOString(),
                double_signed: isDoubleSigned,
                result: trimmedResult,
                status: "closed",
            });
            setIsSubmitted(true);
        } catch {
            return;
        }
    }

    return (
        <div className="fixed inset-0 z-50">
            <button
                type="button"
                aria-label="Закрыть панель отчета"
                onClick={onClose}
                className="absolute inset-0 bg-black/55"
            />

            <aside className="absolute right-0 top-0 h-full w-full overflow-y-auto border-l border-white/10 bg-slate-950/95 p-6 shadow-2xl shadow-black/50 backdrop-blur md:w-[78%] lg:w-[33.333%]">
                <div className="flex items-start justify-between gap-4">
                    <div>
                        <p className="text-xs font-semibold uppercase tracking-[0.25em] text-slate-400">
                            Закрытие тикета
                        </p>
                        <h2 className="mt-2 text-2xl font-semibold text-slate-100">
                            Отчет по заявке #{ticketNumber || "—"}
                        </h2>
                    </div>

                    <button
                        type="button"
                        onClick={onClose}
                        className="inline-flex h-10 w-10 items-center justify-center rounded-2xl bg-white/10 text-slate-100 transition hover:bg-white/20"
                        aria-label="Закрыть"
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
                            <path d="M18 6 6 18" />
                            <path d="m6 6 12 12" />
                        </svg>
                    </button>
                </div>

                <div className="mt-8 rounded-2xl border border-white/10 bg-white/5 p-5">
                    <p className="text-sm text-slate-300">{reportSummary || "—"}</p>
                </div>

                <form className="mt-8 space-y-8" onSubmit={handleSubmit}>
                    <div className="space-y-3">
                        <label htmlFor="ticket-report-result" className="block text-3xl font-semibold text-slate-100">
                            Результат работы
                        </label>
                        <textarea
                            id="ticket-report-result"
                            name="result"
                            required
                            value={resultValue}
                            onChange={(event) => setResultValue(event.target.value)}
                            disabled={isSubmitted}
                            placeholder="Опишите какие работы были проведены"
                            className="min-h-56 w-full resize-y rounded-2xl border border-slate-400/35 bg-transparent px-4 py-4 text-xl text-slate-100 outline-none transition placeholder:text-slate-400 focus:border-[#9fb5d6] focus:ring-2 focus:ring-[#9fb5d6]/30 disabled:opacity-80"
                        />
                    </div>

                    <div className="space-y-3">
                        <label htmlFor="ticket-report-media" className="block text-3xl font-semibold text-slate-100">
                            Фотографии
                        </label>

                        <div className="flex flex-wrap items-start gap-3">
                            <label
                                htmlFor="ticket-report-media"
                                className="group flex h-40 w-40 cursor-pointer items-center justify-center rounded-2xl border border-slate-400/35 bg-transparent transition hover:border-slate-300/60 hover:bg-white/5"
                            >
                                <span className="inline-flex h-10 w-10 items-center justify-center rounded-lg bg-slate-500/30 text-slate-300 transition group-hover:bg-slate-400/40 group-hover:text-slate-100">
                                    <svg
                                        xmlns="http://www.w3.org/2000/svg"
                                        viewBox="0 0 24 24"
                                        fill="none"
                                        stroke="currentColor"
                                        strokeWidth="2"
                                        strokeLinecap="round"
                                        strokeLinejoin="round"
                                        className="h-6 w-6"
                                        aria-hidden="true"
                                    >
                                        <path d="M3 7a2 2 0 0 1 2-2h2l1.2-1.5A2 2 0 0 1 9.8 3h4.4a2 2 0 0 1 1.6.8L17 5h2a2 2 0 0 1 2 2v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z" />
                                        <circle cx="12" cy="13" r="3.5" />
                                    </svg>
                                </span>
                            </label>

                            {mediaPreviews.map((item) => (
                                <div
                                    key={item.id}
                                    className="relative h-40 w-40 overflow-hidden rounded-2xl border border-slate-400/35 bg-slate-900/60"
                                >
                                    <img
                                        src={item.previewUrl}
                                        alt={item.file.name}
                                        className="h-full w-full object-cover"
                                    />
                                    <button
                                        type="button"
                                        onClick={() => handleRemovePreview(item.id)}
                                        disabled={isSubmitting || isSubmitted || Boolean(item.serverAttachmentId)}
                                        className="absolute right-2 top-2 inline-flex h-7 w-7 items-center justify-center rounded-full bg-black/60 text-white transition hover:bg-black/80 disabled:opacity-60"
                                        aria-label={`Удалить ${item.file.name}`}
                                    >
                                        <svg
                                            xmlns="http://www.w3.org/2000/svg"
                                            viewBox="0 0 24 24"
                                            fill="none"
                                            stroke="currentColor"
                                            strokeWidth="2.5"
                                            strokeLinecap="round"
                                            strokeLinejoin="round"
                                            className="h-4 w-4"
                                            aria-hidden="true"
                                        >
                                            <path d="M18 6 6 18" />
                                            <path d="m6 6 12 12" />
                                        </svg>
                                    </button>

                                    <span
                                        className={`absolute bottom-2 left-2 rounded-full px-2 py-1 text-[11px] font-semibold text-white ${resolveUploadStatusClassName(item.uploadStatus)}`}
                                    >
                                        {resolveUploadStatusLabel(item.uploadStatus)}
                                    </span>

                                    {item.serverAttachmentId ? (
                                        <button
                                            type="button"
                                            onClick={() => handleDownloadPreview(item)}
                                            disabled={isSubmitting}
                                            className="absolute bottom-2 right-2 rounded-full bg-black/60 px-3 py-1 text-[11px] font-semibold text-white transition hover:bg-black/80 disabled:opacity-60"
                                        >
                                            Скачать
                                        </button>
                                    ) : null}
                                </div>
                            ))}
                        </div>

                        <input
                            id="ticket-report-media"
                            name="media"
                            type="file"
                            accept="image/*"
                            multiple
                            disabled={isSubmitting || isSubmitted}
                            className="sr-only"
                            onChange={handleMediaSelect}
                        />

                        <label className="mt-4 inline-flex cursor-pointer items-center gap-3 select-none">
                            <input
                                type="checkbox"
                                name="double_signed"
                                className="h-5 w-5 rounded border border-slate-400/60 bg-transparent text-[#6A3BF2] accent-[#6A3BF2]"
                                checked={isDoubleSigned}
                                disabled={isSubmitting || isSubmitted}
                                onChange={(event) => setIsDoubleSigned(event.target.checked)}
                            />
                            <span className="text-lg text-slate-100">Акт подписан с двух сторон</span>
                        </label>
                    </div>

                    <div className="pt-3">
                        <div className="h-px w-full bg-white/15" />
                    </div>

                    <div className="sticky bottom-0 -mx-6 border-t border-white/15 bg-slate-950/95 px-6 pb-1 pt-5 backdrop-blur">
                        <button
                            type="submit"
                            disabled={isSubmitting || isSubmitted}
                            className="flex min-h-14 w-full items-center justify-center rounded-2xl bg-emerald-500 px-5 text-base font-semibold text-white transition hover:bg-emerald-400 disabled:cursor-not-allowed disabled:bg-emerald-600/70 sm:text-lg"
                        >
                            <span className="mr-2 inline-flex h-6 w-6 items-center justify-center rounded-full bg-white/20">
                                <svg
                                    xmlns="http://www.w3.org/2000/svg"
                                    viewBox="0 0 24 24"
                                    fill="none"
                                    stroke="currentColor"
                                    strokeWidth="2.5"
                                    strokeLinecap="round"
                                    strokeLinejoin="round"
                                    className="h-4 w-4"
                                    aria-hidden="true"
                                >
                                    <path d="M20 6 9 17l-5-5" />
                                </svg>
                            </span>
                            <span>
                                {isSubmitting ? "Сохраняем..." : isSubmitted ? "Тикет закрыт" : "Закрыть тикет"}
                            </span>
                        </button>

                        {localError ? <p className="mt-2 text-center text-xs text-rose-200">{localError}</p> : null}
                        {submitError ? <p className="mt-2 text-center text-xs text-rose-200">{submitError}</p> : null}
                        {isSubmitted ? (
                            <p className="mt-2 text-center text-xs text-emerald-200">
                                Тикет закрыт, вложения загружены.
                            </p>
                        ) : null}
                    </div>
                </form>
            </aside>
        </div>
    );
}
