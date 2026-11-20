import { SlideOverSheet } from "../../../../shared/ui/slide-over-sheet";

export function TicketCancellationSheet({ isSubmitting, onClose, onConfirm, submitError, ticketNumber }) {
  return (
    <SlideOverSheet
      isOpen
      onClose={isSubmitting ? undefined : onClose}
      eyebrow={`Тикет №${ticketNumber}`}
      title="Отменить тикет?"
    >
      <p className="mt-6 text-base text-slate-300">
        Тикет будет отменён и исчезнет из списка активных тикетов.
      </p>
      {submitError ? <p role="alert" className="mt-4 text-sm text-rose-200">{submitError}</p> : null}
      <div className="mt-8 flex flex-col gap-3">
        <button
          type="button"
          disabled={isSubmitting}
          onClick={onConfirm}
          className="min-h-12 rounded-xl bg-rose-600 px-5 font-semibold text-white transition hover:bg-rose-500 disabled:cursor-not-allowed disabled:opacity-50"
        >
          {isSubmitting ? "Отменяем..." : "Отменить тикет"}
        </button>
        <button
          type="button"
          disabled={isSubmitting}
          onClick={onClose}
          className="min-h-12 rounded-xl bg-white/10 px-5 font-semibold text-slate-100 transition hover:bg-white/20 disabled:opacity-50"
        >
          Оставить тикет
        </button>
      </div>
    </SlideOverSheet>
  );
}
