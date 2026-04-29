function PhoneIcon() {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      className="h-8 w-8"
      aria-hidden="true"
    >
      <path d="M22 16.92v2a2 2 0 0 1-2.18 2 19.8 19.8 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.8 19.8 0 0 1-3.07-8.67A2 2 0 0 1 4.11 1h2a2 2 0 0 1 2 1.72c.12.9.32 1.79.59 2.64a2 2 0 0 1-.45 2.11L7.09 8.91a16 16 0 0 0 8 8l1.44-1.16a2 2 0 0 1 2.11-.45c.85.27 1.74.47 2.64.59A2 2 0 0 1 22 16.92z" />
      <path d="M15.5 5.5a5 5 0 0 1 3 3" />
      <path d="M15.5 1.5a9 9 0 0 1 7 7" />
    </svg>
  );
}

function MailIcon() {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      className="h-8 w-8"
      aria-hidden="true"
    >
      <rect x="3" y="5" width="18" height="14" rx="2" />
      <path d="M3 7l9 6 9-6" />
    </svg>
  );
}

export function ContactCard({
  actionButtonClassName = "",
  className = "",
  contactName,
  contactPosition,
  phoneHref,
  emailHref,
}) {
  const hasContactData = Boolean(contactName?.trim() || contactPosition?.trim() || phoneHref || emailHref);

  if (!hasContactData) {
    return null;
  }

  return (
    <div
      className={`flex items-center gap-4 rounded-3xl border border-white/10 bg-slate-950/35 p-5 text-left shadow-xl shadow-black/20 backdrop-blur ${className}`.trim()}
    >
      <div className="min-w-0 flex-1">
        <p className="text-lg font-semibold leading-tight text-slate-100 sm:text-2xl">{contactName || "Не указано"}</p>
        <p className="mt-2 text-sm text-slate-400 sm:text-xl">{contactPosition || "Не указано"}</p>
      </div>
      <div className="flex items-center gap-3">
        {phoneHref ? (
          <a
            href={phoneHref}
            aria-label={`Позвонить ${contactName || "контакту"}`}
            className={`inline-flex h-12 w-12 items-center justify-center rounded-full border border-white/15 bg-white/5 text-white transition hover:border-white/30 hover:bg-white/10 sm:h-16 sm:w-16 ${actionButtonClassName}`.trim()}
          >
            <PhoneIcon />
          </a>
        ) : null}
        {emailHref ? (
          <a
            href={emailHref}
            aria-label={`Написать ${contactName || "контакту"}`}
            className={`inline-flex h-12 w-12 items-center justify-center rounded-full border border-white/15 bg-white/5 text-white transition hover:border-white/30 hover:bg-white/10 sm:h-16 sm:w-16 ${actionButtonClassName}`.trim()}
          >
            <MailIcon />
          </a>
        ) : null}
      </div>
    </div>
  );
}
