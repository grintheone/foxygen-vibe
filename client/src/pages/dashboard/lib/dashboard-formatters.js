import { MOCK_TICKET_REASONS } from "../model/mock-dashboard-data";

export function resolveTicketReason(ticket) {
  const reason = MOCK_TICKET_REASONS.find((item) => item.id === ticket.reason);
  if (!reason) {
    return "Не указано";
  }

  if (ticket.status === "assigned") {
    return reason.future || reason.title || "Не указано";
  }

  if (ticket.status === "worksDone") {
    return reason.past || reason.title || "Не указано";
  }

  return reason.present || reason.title || "Не указано";
}

export function formatWorkDuration(startedAt, finishedAt) {
  if (!startedAt || !finishedAt) {
    return "Не указано";
  }

  const start = new Date(startedAt).getTime();
  const finish = new Date(finishedAt).getTime();
  if (Number.isNaN(start) || Number.isNaN(finish) || finish < start) {
    return "Не указано";
  }

  const minutes = Math.floor((finish - start) / 60000);
  const hoursPart = Math.floor(minutes / 60);
  const minutesPart = minutes % 60;

  if (hoursPart === 0) {
    return `${minutesPart} мин`;
  }

  return `${hoursPart} ч. ${minutesPart} мин`;
}

export function formatDateDayMonth(value) {
  if (!value) {
    return "--.--";
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "--.--";
  }

  const day = String(date.getDate()).padStart(2, "0");
  const month = String(date.getMonth() + 1).padStart(2, "0");
  return `${day}.${month}`;
}

export function isTodayOrPast(value) {
  if (!value) {
    return false;
  }

  const target = new Date(value);
  if (Number.isNaN(target.getTime())) {
    return false;
  }

  const now = new Date();
  const todayStart = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime();
  const targetStart = new Date(target.getFullYear(), target.getMonth(), target.getDate()).getTime();

  return todayStart >= targetStart;
}

export function resolveTicketDeadlineDisplay(ticket) {
  if (ticket?.workfinished_at) {
    return {
      dateValue: formatDateDayMonth(ticket.workfinished_at),
      isOverdue: false,
      shouldUseFireIcon: false,
      isPlaceholder: false,
      isFinishedDate: true,
    };
  }

  if (!ticket?.assigned_end) {
    return {
      dateValue: formatDateDayMonth(ticket?.assigned_end),
      isOverdue: false,
      shouldUseFireIcon: false,
      isPlaceholder: true,
      isFinishedDate: false,
    };
  }

  const isOverdue = isTodayOrPast(ticket.assigned_end);
  return {
    dateValue: formatDateDayMonth(ticket.assigned_end),
    isOverdue,
    shouldUseFireIcon: isOverdue,
    isPlaceholder: false,
    isFinishedDate: false,
  };
}
