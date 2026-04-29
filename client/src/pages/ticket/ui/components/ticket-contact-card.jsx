function PhoneIcon() {
    return (
        <svg
            xmlns="http://www.w3.org/2000/svg"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="1.8"
            strokeLinecap="round"
            strokeLinejoin="round"
            className="h-5 w-5"
            aria-hidden="true"
        >
            <path d="M22 16.9v2a2 2 0 0 1-2.2 2 19.8 19.8 0 0 1-8.6-3.1 19.5 19.5 0 0 1-6-6 19.8 19.8 0 0 1-3.1-8.6A2 2 0 0 1 4.1 1h2a2 2 0 0 1 2 1.7l.4 2.2a2 2 0 0 1-.5 1.8L6.9 8a16 16 0 0 0 9.1 9.1l1.3-1.1a2 2 0 0 1 1.8-.5l2.2.4a2 2 0 0 1 1.7 2Z" />
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
            strokeWidth="1.8"
            strokeLinecap="round"
            strokeLinejoin="round"
            className="h-5 w-5"
            aria-hidden="true"
        >
            <path d="M4 6h16v12H4z" />
            <path d="m4 7 8 6 8-6" />
        </svg>
    );
}

function ContactActionButton({ ariaLabel, href, icon }) {
    if (!href) {
        return null;
    }

    return (
        <a
            href={href}
            aria-label={ariaLabel}
            className="inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl bg-[#2F3545] text-white transition hover:bg-[#394055] sm:h-12 sm:w-12"
        >
            {icon}
        </a>
    );
}

export function TicketContactCard({ contactName, contactPosition, phoneHref, emailHref }) {
    const hasContactData = Boolean(contactName?.trim() || contactPosition?.trim() || phoneHref || emailHref);
    const hasPhone = Boolean(phoneHref);
    const hasEmail = Boolean(emailHref);
    const actionCount = Number(hasPhone) + Number(hasEmail);

    if (!hasContactData) {
        return (
            <div className="rounded-lg border border-white/20 bg-transparent px-4 py-4">
                <p className="text-[16px] font-semibold leading-snug tracking-tight text-slate-50">Контакт не указан</p>
                <p className="mt-2 text-[16px] leading-snug text-slate-200/85">
                    Для этого тикета не выбран контакт клиента.
                </p>
            </div>
        );
    }

    return (
        <div className="flex items-start justify-between gap-4 rounded-lg border border-white/20 bg-transparent px-4 py-4 text-left">
            <div className="min-w-0 flex-1">
                <p className="text-[16px] font-semibold leading-snug tracking-tight text-slate-50">
                    {contactName || "Не указано"}
                </p>
                <p className="mt-2 text-[16px] leading-snug text-slate-200/85">
                    {contactPosition || "Не указано"}
                </p>
            </div>

            {actionCount > 0 ? (
                <div
                    className={`shrink-0 ${actionCount > 1 ? "grid grid-cols-2 gap-3" : "flex items-center gap-3"}`}
                >
                    <ContactActionButton
                        href={phoneHref}
                        ariaLabel={`Позвонить ${contactName || "контакту"}`}
                        icon={<PhoneIcon />}
                    />
                    <ContactActionButton
                        href={emailHref}
                        ariaLabel={`Написать ${contactName || "контакту"}`}
                        icon={<MailIcon />}
                    />
                </div>
            ) : null}
        </div>
    );
}
