import { ContactCard } from "../../../../shared/ui/contact-card";

export function TicketContactCard({ contactName, contactPosition, phoneHref, emailHref }) {
  return (
    <ContactCard
      contactName={contactName}
      contactPosition={contactPosition}
      phoneHref={phoneHref}
      emailHref={emailHref}
    />
  );
}
