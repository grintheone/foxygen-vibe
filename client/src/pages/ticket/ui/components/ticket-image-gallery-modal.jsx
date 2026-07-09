import { useEffect, useRef } from "react";

function wrapIndex(index, total) {
    if (total <= 0) {
        return -1;
    }

    if (index < 0) {
        return total - 1;
    }

    if (index >= total) {
        return 0;
    }

    return index;
}

export function TicketImageGalleryModal({ activeIndex, images, onClose, onSelectIndex }) {
    const touchStartRef = useRef(null);
    const totalImages = Array.isArray(images) ? images.length : 0;
    const safeIndex = activeIndex >= 0 && activeIndex < totalImages ? activeIndex : -1;
    const activeImage = safeIndex >= 0 ? images[safeIndex] : null;
    const hasMultipleImages = totalImages > 1;

    useEffect(() => {
        if (!activeImage) {
            return undefined;
        }

        const previousOverflow = document.body.style.overflow;
        document.body.style.overflow = "hidden";

        function handleKeyDown(event) {
            if (event.key === "Escape") {
                onClose();
                return;
            }

            if (!hasMultipleImages) {
                return;
            }

            if (event.key === "ArrowLeft") {
                onSelectIndex(wrapIndex(safeIndex - 1, totalImages));
            }

            if (event.key === "ArrowRight") {
                onSelectIndex(wrapIndex(safeIndex + 1, totalImages));
            }
        }

        document.addEventListener("keydown", handleKeyDown);

        return () => {
            document.body.style.overflow = previousOverflow;
            document.removeEventListener("keydown", handleKeyDown);
        };
    }, [activeImage, hasMultipleImages, onClose, onSelectIndex, safeIndex, totalImages]);

    if (!activeImage) {
        return null;
    }

    function handleTouchStart(event) {
        const touch = event.changedTouches?.[0];
        if (!touch) {
            return;
        }

        touchStartRef.current = {
            x: touch.clientX,
            y: touch.clientY,
        };
    }

    function handleTouchEnd(event) {
        if (!hasMultipleImages || !touchStartRef.current) {
            touchStartRef.current = null;
            return;
        }

        const touch = event.changedTouches?.[0];
        if (!touch) {
            touchStartRef.current = null;
            return;
        }

        const deltaX = touch.clientX - touchStartRef.current.x;
        const deltaY = touch.clientY - touchStartRef.current.y;
        touchStartRef.current = null;

        if (Math.abs(deltaX) < 40 || Math.abs(deltaX) < Math.abs(deltaY)) {
            return;
        }

        onSelectIndex(wrapIndex(safeIndex + (deltaX > 0 ? -1 : 1), totalImages));
    }

    function showPreviousImage() {
        onSelectIndex(wrapIndex(safeIndex - 1, totalImages));
    }

    function showNextImage() {
        onSelectIndex(wrapIndex(safeIndex + 1, totalImages));
    }

    return (
        <div className="fixed inset-0 z-[70]">
            <button
                type="button"
                aria-label="Закрыть просмотр изображений"
                className="absolute inset-0 bg-slate-950/92 backdrop-blur-sm"
                onClick={onClose}
            />

            <div
                role="dialog"
                aria-modal="true"
                aria-label={activeImage.name || "Просмотр изображения"}
                className="absolute inset-0 flex items-center justify-center pl-[calc(0.75rem+env(safe-area-inset-left))] pr-[calc(0.75rem+env(safe-area-inset-right))] pt-[calc(1.5rem+env(safe-area-inset-top))] pb-[calc(1.5rem+env(safe-area-inset-bottom))] sm:pl-[calc(1.5rem+env(safe-area-inset-left))] sm:pr-[calc(1.5rem+env(safe-area-inset-right))]"
            >
                <div
                    className="relative flex h-full w-full max-w-6xl flex-col overflow-hidden rounded-[2rem] border border-white/10 bg-slate-950/85 shadow-2xl shadow-black/50"
                    onTouchStart={handleTouchStart}
                    onTouchEnd={handleTouchEnd}
                >
                    <div className="flex items-start justify-between gap-4 border-b border-white/10 px-4 py-4 sm:px-6">
                        <div className="min-w-0">
                            <p className="truncate text-sm font-semibold uppercase tracking-[0.24em] text-[#AADB1E]">
                                Изображения отчета
                            </p>
                            <p className="mt-2 truncate text-sm text-slate-200 sm:text-base">{activeImage.name}</p>
                        </div>

                        <div className="flex items-center gap-3">
                            <span className="rounded-full border border-white/10 bg-white/5 px-3 py-1 text-xs font-semibold text-slate-200">
                                {safeIndex + 1} / {totalImages}
                            </span>
                            <button
                                type="button"
                                onClick={onClose}
                                className="inline-flex h-11 w-11 items-center justify-center rounded-full border border-white/10 bg-white/5 text-slate-100 transition hover:bg-white/10"
                                aria-label="Закрыть"
                            >
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
                                    <path d="M18 6 6 18" />
                                    <path d="m6 6 12 12" />
                                </svg>
                            </button>
                        </div>
                    </div>

                    <div className="relative flex min-h-0 flex-1 items-center justify-center overflow-hidden px-3 py-4 sm:px-6 sm:py-6">
                        {hasMultipleImages ? (
                            <button
                                type="button"
                                onClick={showPreviousImage}
                                className="absolute left-3 top-1/2 z-10 inline-flex h-12 w-12 -translate-y-1/2 items-center justify-center rounded-full border border-white/10 bg-black/45 text-white transition hover:bg-black/65 sm:left-6"
                                aria-label="Предыдущее изображение"
                            >
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
                                    <path d="m15 18-6-6 6-6" />
                                </svg>
                            </button>
                        ) : null}

                        <div className="flex h-full w-full items-center justify-center rounded-[1.75rem] bg-black/25">
                            {activeImage.previewUrl ? (
                                <img
                                    src={activeImage.previewUrl}
                                    alt={activeImage.name}
                                    className="max-h-full max-w-full object-contain select-none"
                                    draggable="false"
                                />
                            ) : (
                                <div className="flex h-full w-full items-center justify-center px-6 text-center text-sm text-slate-300">
                                    Изображение загружается...
                                </div>
                            )}
                        </div>

                        {hasMultipleImages ? (
                            <button
                                type="button"
                                onClick={showNextImage}
                                className="absolute right-3 top-1/2 z-10 inline-flex h-12 w-12 -translate-y-1/2 items-center justify-center rounded-full border border-white/10 bg-black/45 text-white transition hover:bg-black/65 sm:right-6"
                                aria-label="Следующее изображение"
                            >
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
                                    <path d="m9 18 6-6-6-6" />
                                </svg>
                            </button>
                        ) : null}
                    </div>
                </div>
            </div>
        </div>
    );
}
