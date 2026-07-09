import { useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router";
import { routePaths } from "../../../shared/config/routes";
import { ProfileTicketCard } from "../../../shared/ui/profile-ticket-card";
import { useDashboardTickets } from "../lib/use-dashboard-tickets";
import { TicketSliderCard } from "./ticket-slider-card";

export function EngineerDashboard({ executorId }) {
  const navigate = useNavigate();
  const [activeSlide, setActiveSlide] = useState(0);
  const pointerStartXRef = useRef(null);
  const suppressClickRef = useRef(false);
  const touchStartRef = useRef(null);
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

  function applySwipe(deltaX, deltaY = 0) {
    const swipeThreshold = 45;
    const isHorizontalSwipe = Math.abs(deltaX) >= swipeThreshold && Math.abs(deltaX) > Math.abs(deltaY);

    if (!isHorizontalSwipe) {
      return;
    }

    suppressClickRef.current = true;
    if (deltaX > 0) {
      goToPreviousSlide();
    } else {
      goToNextSlide();
    }
  }

  function handlePointerDown(event) {
    if (event.pointerType === "touch") {
      return;
    }

    pointerStartXRef.current = event.clientX;
    suppressClickRef.current = false;
  }

  function handlePointerUp(event) {
    if (event.pointerType === "touch") {
      return;
    }

    if (pointerStartXRef.current === null) {
      return;
    }

    const deltaX = event.clientX - pointerStartXRef.current;
    applySwipe(deltaX);
    pointerStartXRef.current = null;
  }

  function handleTouchStart(event) {
    const [touch] = event.changedTouches;
    if (!touch) {
      return;
    }

    touchStartRef.current = {
      x: touch.clientX,
      y: touch.clientY,
    };
    suppressClickRef.current = false;
  }

  function handleTouchEnd(event) {
    if (!touchStartRef.current) {
      return;
    }

    const [touch] = event.changedTouches;
    if (!touch) {
      touchStartRef.current = null;
      return;
    }

    applySwipe(touch.clientX - touchStartRef.current.x, touch.clientY - touchStartRef.current.y);
    touchStartRef.current = null;
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
            className="touch-pan-y overflow-hidden rounded-3xl"
            onPointerDown={handlePointerDown}
            onPointerUp={handlePointerUp}
            onPointerCancel={() => {
              pointerStartXRef.current = null;
            }}
            onTouchStart={handleTouchStart}
            onTouchEnd={handleTouchEnd}
            onTouchCancel={() => {
              touchStartRef.current = null;
            }}
          >
            <div
              className="flex transition-transform duration-300 ease-out"
              style={{ transform: `translateX(-${activeSlide * 100}%)` }}
            >
              {tickets.map((ticket) => (
                <article key={ticket.id} className="min-w-full px-1">
                  <TicketSliderCard ticket={ticket} onOpenTicket={handleOpenTicket} />
                </article>
              ))}
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
