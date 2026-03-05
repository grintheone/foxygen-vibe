import { useRef, useState } from "react";
import { useNavigate } from "react-router";
import { routePaths } from "../../../shared/config/routes";
import { formatWorkDuration, resolveTicketReason } from "../lib/dashboard-formatters";
import { useDashboardTickets } from "../lib/use-dashboard-tickets";
import { TicketCardWithStatus } from "./ticket-card-with-status";

export function EngineerDashboard({ executorId }) {
  const navigate = useNavigate();
  const [activeSlide, setActiveSlide] = useState(0);
  const pointerStartXRef = useRef(null);
  const suppressClickRef = useRef(false);
  const { tickets, assignedTickets } = useDashboardTickets(executorId);

  function goToPreviousSlide() {
    setActiveSlide((prev) => (prev - 1 + tickets.length) % tickets.length);
  }

  function goToNextSlide() {
    setActiveSlide((prev) => (prev + 1) % tickets.length);
  }

  function handlePointerDown(event) {
    pointerStartXRef.current = event.clientX;
    suppressClickRef.current = false;
  }

  function handlePointerUp(event) {
    if (pointerStartXRef.current === null) {
      return;
    }

    const deltaX = event.clientX - pointerStartXRef.current;
    const swipeThreshold = 45;

    if (Math.abs(deltaX) >= swipeThreshold) {
      suppressClickRef.current = true;
      if (deltaX > 0) {
        goToPreviousSlide();
      } else {
        goToNextSlide();
      }
    }

    pointerStartXRef.current = null;
  }

  function handleOpenTicket(ticketId) {
    if (suppressClickRef.current) {
      suppressClickRef.current = false;
      return;
    }

    navigate(routePaths.ticketById(ticketId));
  }

  return (
    <section className="space-y-6">
      {tickets.length > 0 ? (
        <>
          <div
            className="overflow-hidden rounded-3xl"
            onPointerDown={handlePointerDown}
            onPointerUp={handlePointerUp}
            onPointerCancel={() => {
              pointerStartXRef.current = null;
            }}
          >
            <div
              className="flex transition-transform duration-300 ease-out"
              style={{ transform: `translateX(-${activeSlide * 100}%)` }}
            >
              {tickets.map((ticket) => {
                const isInWork = ticket.status === "inWork";
                const reasonValue = resolveTicketReason(ticket);
                const isInWorkValue = isInWork
                  ? "В процессе"
                  : formatWorkDuration(ticket.workstarted_at, ticket.workfinished_at);
                const cardClassName = isInWork
                  ? "border-emerald-300/30 bg-emerald-500/20"
                  : "border-fuchsia-300/30 bg-fuchsia-500/20";
                const toneBlockClass = isInWork
                  ? "border-emerald-200/30 bg-emerald-200/20"
                  : "border-fuchsia-200/30 bg-fuchsia-200/20";

                return (
                  <article key={ticket.id} className="min-w-full px-1">
                    <button
                      type="button"
                      onClick={() => handleOpenTicket(ticket.id)}
                      className={`w-full rounded-3xl border p-6 text-left shadow-xl backdrop-blur transition hover:border-white/50 ${cardClassName}`}
                    >
                      <div className="flex items-center gap-2">
                        <p className={`rounded-xl border px-3 py-1.5 text-sm font-semibold text-white ${toneBlockClass}`}>
                          {reasonValue || "Не указано"}
                        </p>
                        {isInWork ? (
                          <p className="inline-flex items-center gap-2 text-sm font-semibold text-white">
                            <span className="relative flex h-2.5 w-2.5">
                              <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-100 opacity-80" />
                              <span className="relative inline-flex h-2.5 w-2.5 rounded-full bg-emerald-50" />
                            </span>
                            {isInWorkValue}
                          </p>
                        ) : (
                          <p className="text-sm font-semibold text-white">{isInWorkValue}</p>
                        )}
                        <p className="ml-auto text-sm font-semibold text-white">#{ticket.number}</p>
                      </div>

                      <div className="mt-4 flex flex-col gap-1.5 text-white">
                        <p className="text-base font-semibold">{ticket.deviceName}</p>
                        <p className="text-sm text-slate-100/90">С/Н {ticket.deviceSerialNumber}</p>
                      </div>

                      <div className={`mt-4 rounded-2xl border p-4 ${toneBlockClass}`}>
                        <div className="flex flex-col gap-1 text-white">
                          <p className="text-sm font-semibold">{ticket.clientName}</p>
                          <p className="text-sm text-slate-100/90">{ticket.clientAddress}</p>
                        </div>
                      </div>
                    </button>
                  </article>
                );
              })}
            </div>
          </div>

          <div className="flex items-center justify-center gap-2">
            {tickets.map((ticket, index) => {
              const isActive = index === activeSlide;

              return (
                <button
                  key={ticket.id}
                  type="button"
                  onClick={() => setActiveSlide(index)}
                  aria-label={`Перейти к слайду ${index + 1}`}
                  className={`h-2.5 rounded-full transition ${
                    isActive ? "w-8 bg-white" : "w-2.5 bg-white/35 hover:bg-white/60"
                  }`}
                />
              );
            })}
          </div>
        </>
      ) : (
        <section className="rounded-3xl border border-white/10 bg-white/5 p-6">
          <p className="text-sm text-slate-300">Для текущего инженера нет тикетов в процессе.</p>
        </section>
      )}

      <section className="space-y-3">
        <h2 className="text-sm font-semibold uppercase tracking-[0.18em] text-slate-300">Назначенные тикеты</h2>
        {assignedTickets.length > 0 ? (
          <div className="grid gap-3">
            {assignedTickets.map((ticket) => (
              <TicketCardWithStatus key={ticket.id} ticket={ticket} onOpenTicket={handleOpenTicket} />
            ))}
          </div>
        ) : (
          <div className="rounded-2xl border border-white/10 bg-white/5 p-4 text-sm text-slate-300">
            Нет назначенных тикетов.
          </div>
        )}
      </section>
    </section>
  );
}
