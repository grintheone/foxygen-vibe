import { formatAppDate } from "../../../shared/lib/date-time";
import { formatWorkDuration, resolveTicketDeadlineDisplay } from "../../dashboard/lib/dashboard-formatters";
import { FALLBACK_STATUS_ICON, statusIconByType } from "../model/ticket-page-model";

function formatMonthDay(value) {
  return formatAppDate(value, { includeYear: false });
}

function normalizePhoneValue(phone) {
    if (!phone) {
        return "";
    }

    return phone.replace(/[^\d+]/g, "");
}

export function useTicketViewModel(ticket) {
    const ticketNumber = ticket?.number ?? "—";
    const statusIcon = statusIconByType[ticket?.status] || FALLBACK_STATUS_ICON;
    const statusAlt = ticket?.status || "status";
    const finishedDate = formatMonthDay(ticket?.workfinished_at);
    const isInWork = ticket?.status === "inWork";
    const deadlineDisplay = resolveTicketDeadlineDisplay(ticket);
    const reasonValue = ticket?.resolvedReason || "Не указано";
    const canOpenDevice = Boolean(ticket?.device);
    const canOpenClient = Boolean(ticket?.client);
    const phoneHrefValue = normalizePhoneValue(ticket?.contactPhone);
    const phoneHref = phoneHrefValue ? `tel:${phoneHrefValue}` : null;
    const emailHref = ticket?.contactEmail ? `mailto:${ticket.contactEmail}` : null;
    const workDuration = formatWorkDuration(ticket?.workstarted_at, ticket?.workfinished_at);

    return {
        ticketNumber,
        statusIcon,
        statusAlt,
        finishedDate,
        isInWork,
        deadlineDisplay,
        reasonValue,
        canOpenDevice,
        canOpenClient,
        phoneHref,
        emailHref,
        workDuration,
    };
}
