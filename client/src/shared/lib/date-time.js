// API timestamps represent UTC instants; all app calendar values use Moscow time.
export const APP_TIME_ZONE = "Europe/Moscow";

const partsFormatter = new Intl.DateTimeFormat("en-GB", {
  timeZone: APP_TIME_ZONE,
  year: "numeric", month: "2-digit", day: "2-digit",
  hour: "2-digit", minute: "2-digit", hourCycle: "h23",
});

export function parseTimestamp(value) {
  if (!value) return new Date(NaN);
  if (typeof value === "string" && /^\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}(?::\d{2}(?:\.\d+)?)?$/.test(value)) {
    return new Date(`${value.replace(" ", "T")}Z`);
  }
  return new Date(value);
}

function moscowParts(value) {
  const date = parseTimestamp(value);
  if (Number.isNaN(date.getTime())) return null;
  return Object.fromEntries(partsFormatter.formatToParts(date).map(({ type, value }) => [type, value]));
}

export function formatAppDate(value, { shortYear = false, includeYear = true } = {}) {
  const p = moscowParts(value);
  if (!p) return null;
  return `${p.day}.${p.month}${includeYear ? `.${shortYear ? p.year.slice(-2) : p.year}` : ""}`;
}

export function formatAppTime(value) {
  const p = moscowParts(value);
  return p ? `${p.hour}:${p.minute}` : null;
}

export function formatAppDateTime(value) {
  return moscowParts(value) ? `${formatAppDate(value)} ${formatAppTime(value)}` : null;
}

export function toAppDateValue(value) {
  const p = moscowParts(value);
  return p ? `${p.year}-${p.month}-${p.day}` : "";
}

export function toAppDateTimeValue(value) {
  const p = moscowParts(value);
  return p ? `${p.year}-${p.month}-${p.day}T${p.hour}:${p.minute}` : "";
}

// datetime-local inputs have no offset. Interpret user input as Moscow (UTC+03:00).
export function appDateTimeToISO(value) {
  const normalized = value.trim();
  if (!normalized) return "";
  if (!/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}(?::\d{2}(?:\.\d+)?)?$/.test(normalized)) return null;
  const date = new Date(`${normalized}+03:00`);
  return Number.isNaN(date.getTime()) ? null : date.toISOString();
}
