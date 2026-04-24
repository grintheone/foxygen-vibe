import { useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router";
import ticketDoneIcon from "../../../assets/icons/ticket-done.svg";
import ticketInWorkIcon from "../../../assets/icons/ticket-inwork.svg";
import { routePaths } from "../../../shared/config/routes";
import { ProfileTicketCard } from "../../../shared/ui/profile-ticket-card";
import { formatWorkDuration, resolveTicketReason } from "../lib/dashboard-formatters";
import { useDashboardTickets } from "../lib/use-dashboard-tickets";

export function EngineerDashboard({ executorId }) {
  const navigate = useNavigate();
  const [activeSlide, setActiveSlide] = useState(0);
  const pointerStartXRef = useRef(null);
  const suppressClickRef = useRef(false);
  const { tickets, assignedTickets, isLoading, isError } = useDashboardTickets(executorId);

  useEffect(() => {
    if (activeSlide >= tickets.length) {
      setActiveSlide(0);
    }
  }, [activeSlide, tickets.length]);

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
      {isLoading ? (
        <section className="app-subtle-notice">
          <p className="text-sm text-slate-300">Загружаем тикеты...</p>
        </section>
      ) : null}
      {isError ? (
        <section className="rounded-3xl border border-rose-300/30 bg-rose-500/10 p-6">
          <p className="text-sm text-rose-100">Не удалось загрузить тикеты. Попробуйте обновить страницу.</p>
        </section>
      ) : null}
      {!isLoading && !isError && tickets.length > 0 ? (
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
                const statusIcon = isInWork ? ticketInWorkIcon : ticketDoneIcon;

                return (
                  <article key={ticket.id} className="min-w-full px-1">
                    <button
                      type="button"
                      onClick={() => handleOpenTicket(ticket.id)}
                      className="w-full overflow-hidden rounded-lg border border-slate-400/20 bg-[#2f3748] text-left shadow-xl shadow-black/20 transition hover:border-slate-300/35 hover:bg-[#333c4f]"
                    >
                      <div className="grid grid-cols-[1fr_auto] gap-3 px-4 py-3.5">
                        <div className="min-w-0 space-y-1.5">
                          <div className="flex min-w-0 items-center gap-2">
                            <img src={statusIcon} alt="" className="h-5 w-5 shrink-0" />
                            <p className="truncate text-sm font-semibold text-slate-100">{reasonValue || "Не указано"}</p>
                          </div>
                          <p className="text-base font-semibold text-white">{ticket.deviceName}</p>
                          <p className="text-sm text-slate-300">С/Н {ticket.deviceSerialNumber}</p>
                        </div>

                        <div className="flex flex-col items-end">
                          <p className="text-sm font-semibold text-slate-200">{isInWorkValue}</p>
                          <p className="text-sm font-semibold text-slate-200/80">#{ticket.number}</p>
                        </div>
                      </div>

                      <div className="border-t border-slate-400/10 bg-[#3f485a] px-4 py-3">
                        <div className="flex flex-col gap-1 text-slate-300">
                          <p className="text-sm font-semibold text-slate-100">{ticket.clientName}</p>
                          <p className="text-sm text-slate-200/80">{ticket.clientAddress}</p>
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
      ) : !isLoading && !isError ? (
        <section className="app-subtle-notice">
          <p className="text-sm text-slate-300">Для текущего инженера нет тикетов в процессе.</p>
        </section>
      ) : null}

      <section className="space-y-3">
        <h2 className="text-sm font-semibold uppercase tracking-[0.18em] text-slate-300">Назначенные тикеты</h2>
        {!isLoading && !isError && assignedTickets.length > 0 ? (
          <div className="grid gap-2">
            {assignedTickets.map((ticket) => (
              <ProfileTicketCard key={ticket.id} ticket={ticket} onOpenTicket={handleOpenTicket} />
            ))}
          </div>
        ) : !isLoading && !isError ? (
          <div className="app-subtle-notice">
            Нет назначенных тикетов.
          </div>
        ) : null}
      </section>
    </section>
  );
}
