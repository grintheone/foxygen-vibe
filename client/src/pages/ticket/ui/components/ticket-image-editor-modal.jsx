import { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { rotateImageFile } from "../../lib/rotate-image";

const buttonClass = "rounded-full border border-white/10 bg-white/5 px-4 py-3 text-sm font-semibold text-slate-100 transition hover:bg-white/10 disabled:opacity-50";

export function TicketImageEditorModal({ item, onClose, onSave }) {
    const [turns, setTurns] = useState(item.quarterTurns || 0);
    const [isSaving, setIsSaving] = useState(false);
    const [error, setError] = useState("");
    const [loaded, setLoaded] = useState(false);
    const [imageSize, setImageSize] = useState({ width: 0, height: 0 });
    const [viewport, setViewport] = useState({ width: 0, height: 0 });
    const [sourceUrl, setSourceUrl] = useState("");
    const viewportRef = useRef(null);
    const dialogRef = useRef(null);
    const busyRef = useRef(false);

    useEffect(() => {
        const url = URL.createObjectURL(item.originalFile);
        setSourceUrl(url);
        const previousFocus = document.activeElement;
        dialogRef.current?.focus();
        const observer = new ResizeObserver(([entry]) => setViewport(entry.contentRect));
        observer.observe(viewportRef.current);
        return () => {
            observer.disconnect();
            URL.revokeObjectURL(url);
            previousFocus?.focus();
        };
    }, [item.originalFile]);

    function close() {
        if (!busyRef.current) onClose();
    }

    async function save() {
        if (busyRef.current) return;
        busyRef.current = true;
        setIsSaving(true);
        setError("");
        try {
            const file = await rotateImageFile(item.originalFile, turns);
            onSave(file, turns);
        } catch (error) {
            setError(error?.message?.includes("25 МБ") ? error.message : "Не удалось повернуть изображение. Попробуйте другое фото или сохраните оригинал.");
        } finally {
            busyRef.current = false;
            setIsSaving(false);
        }
    }

    const rotatedWidth = turns % 2 ? imageSize.height : imageSize.width;
    const rotatedHeight = turns % 2 ? imageSize.width : imageSize.height;
    const scale = Math.min(1, viewport.width / (rotatedWidth || 1), viewport.height / (rotatedHeight || 1));

    return createPortal(
        <div className="fixed inset-0 z-[70] bg-slate-950/95 p-3 backdrop-blur-sm sm:p-6">
            <div
                ref={dialogRef}
                role="dialog"
                aria-modal="true"
                aria-label={`Поворот изображения ${item.file.name}`}
                tabIndex={-1}
                onKeyDown={(event) => {
                    if (event.key === "Escape") { event.stopPropagation(); close(); }
                    if (event.key === "Tab") {
                        const buttons = [...dialogRef.current.querySelectorAll("button:not(:disabled)")];
                        const first = buttons[0];
                        const last = buttons[buttons.length - 1];
                        if (event.shiftKey && (document.activeElement === first || document.activeElement === dialogRef.current)) {
                            event.preventDefault(); last?.focus();
                        } else if (!event.shiftKey && document.activeElement === last) {
                            event.preventDefault(); first?.focus();
                        }
                    }
                }}
                className="relative mx-auto flex h-full w-full max-w-6xl flex-col overflow-hidden rounded-[2rem] border border-white/10 bg-slate-950/85 shadow-2xl shadow-black/50 outline-none"
            >
                <div className="flex items-center justify-between gap-4 border-b border-white/10 px-4 py-4 sm:px-6">
                    <div className="min-w-0">
                        <p className="text-sm font-semibold uppercase tracking-[0.24em] text-[#AADB1E]">Изображение отчета</p>
                        <p className="mt-2 truncate text-sm text-slate-200">{item.file.name}</p>
                    </div>
                    <button type="button" className={buttonClass} onClick={close} disabled={isSaving} aria-label="Закрыть редактор">✕</button>
                </div>
                <div className="min-h-0 flex-1 p-3 sm:p-6">
                    <div ref={viewportRef} className="flex h-full w-full items-center justify-center overflow-hidden rounded-[1.75rem] bg-black/25">
                        <img
                            src={sourceUrl || undefined}
                            alt={item.file.name}
                            draggable={false}
                            onLoad={(event) => { setLoaded(true); setImageSize({ width: event.target.naturalWidth, height: event.target.naturalHeight }); }}
                            onError={() => setError("Браузер не может открыть это изображение для поворота. Оригинал можно загрузить без изменений.")}
                            className="max-w-none shrink-0 select-none"
                            style={{ width: imageSize.width * scale, height: imageSize.height * scale, transform: `rotate(${turns * 90}deg)` }}
                        />
                    </div>
                </div>
                <div className="space-y-3 border-t border-white/10 px-4 pt-4 pb-[calc(1rem+env(safe-area-inset-bottom))] sm:px-6">
                    {!/\.(jpe?g|png)$/i.test(item.originalFile.name) && <p className="text-sm text-slate-300">После поворота изображение будет сохранено в PNG. Анимация и дополнительные страницы не сохраняются.</p>}
                    {error && <p role="alert" className="text-sm text-rose-200">{error}</p>}
                    <div className="flex flex-wrap items-center justify-between gap-3">
                        <div className="flex gap-2">
                            <button type="button" className={buttonClass} disabled={!loaded || isSaving} onClick={() => setTurns((value) => (value + 3) % 4)} aria-label="Повернуть влево на 90 градусов">↶ Влево</button>
                            <button type="button" className={buttonClass} disabled={!loaded || isSaving} onClick={() => setTurns((value) => (value + 1) % 4)} aria-label="Повернуть вправо на 90 градусов">↷ Вправо</button>
                        </div>
                        <div className="flex gap-2">
                            <button type="button" className={buttonClass} onClick={close} disabled={isSaving}>Отмена</button>
                            <button type="button" className="rounded-full bg-emerald-500 px-5 py-3 text-sm font-semibold text-white hover:bg-emerald-400 disabled:opacity-50" onClick={save} disabled={!loaded || isSaving}>{isSaving ? "Сохраняем..." : "Сохранить"}</button>
                        </div>
                    </div>
                </div>
            </div>
        </div>, document.body,
    );
}
