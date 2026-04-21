import { ContactCard } from "../../../../shared/ui/contact-card";

export function TicketContactCard({ contactName, contactPosition, phoneHref, emailHref }) {
  const hasContactData = Boolean(contactName?.trim() || contactPosition?.trim() || phoneHref || emailHref);

  if (!hasContactData) {
    return (
      <div className="rounded-3xl border border-dashed border-white/10 bg-white/5 p-5">
        <p className="text-base font-semibold text-slate-100">Контакт не указан</p>
        <p className="mt-2 text-sm text-slate-400">Для этого тикета не выбран контакт клиента.</p>
      </div>
    );
  }

  return (
    <ContactCard
      contactName={contactName}
      contactPosition={contactPosition}
      phoneHref={phoneHref}
      emailHref={emailHref}
    />
  );
}
