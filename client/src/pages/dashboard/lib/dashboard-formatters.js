import { parseTimestamp, formatAppDate, toAppDateValue } from "../../../shared/lib/date-time";
export function resolveTicketReason(ticket) {
  const resolvedReason = ticket?.resolvedReason?.trim();
  if (resolvedReason) {
    return resolvedReason;
  }

  const reasonTitle = ticket?.reasonTitle?.trim();
  if (reasonTitle) {
    return reasonTitle;
  }

  const reason = ticket?.reason?.trim();
  if (reason) {
    return reason;
  }

  return "Не указано";
}

export function formatWorkDuration(startedAt, finishedAt) {
  if (!startedAt || !finishedAt) {
    return "Не указано";
  }

  const start = parseTimestamp(startedAt).getTime();
  const finish = parseTimestamp(finishedAt).getTime();
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
  return formatAppDate(value, { includeYear: false }) ?? "--.--";
}

export function isTodayOrPast(value) {
  const target = toAppDateValue(value);
  return Boolean(target) && toAppDateValue(new Date()) >= target;
}

export function resolveTicketDeadlineDisplay(ticket) {
  if (ticket?.closed_at) {
    return {
      dateValue: formatDateDayMonth(ticket.closed_at),
      isOverdue: false,
      shouldUseFireIcon: false,
      isPlaceholder: false,
      isFinishedDate: true,
    };
  }

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
