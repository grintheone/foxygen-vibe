import { useNavigate, useParams } from "react-router";
import {
  useGetClientByIdQuery,
  useGetClientContactsQuery,
} from "../../../shared/api/tickets-api";
import { TicketContactCard } from "../../ticket/ui/components/ticket-contact-card";
import { TicketHeader } from "../../ticket/ui/components/ticket-header";
import { PageShell } from "../../../shared/ui/page-shell";

function normalizePhoneHref(phone) {
  const value = (phone || "").trim();
  if (!value) {
    return "";
  }

  const normalized = value.replace(/[^\d+]/g, "");
  return normalized ? `tel:${normalized}` : "";
}

function normalizeEmailHref(email) {
  const value = (email || "").trim();
  return value ? `mailto:${value}` : "";
}

export function ClientContactsPage() {
  const navigate = useNavigate();
  const { clientId } = useParams();
  const {
    data: client,
    isError: isClientError,
    isFetching: isClientFetching,
    isLoading: isClientLoading,
  } = useGetClientByIdQuery(clientId, {
    skip: !clientId,
  });
  const {
    data: contacts = [],
    isError: isContactsError,
    isFetching: isContactsFetching,
    isLoading: isContactsLoading,
  } = useGetClientContactsQuery(
    { clientId },
    {
      skip: !clientId,
    },
  );

  const pageTitle = client?.title?.trim() || "Контакты клиента";

  return (
    <PageShell>
      <section className="w-full space-y-6">
        <TicketHeader title={pageTitle} onBack={() => navigate(-1)} />

        {isClientLoading || isClientFetching ? (
          <div className="app-subtle-notice px-1">
            <p className="text-sm text-slate-300">Загрузка клиента...</p>
          </div>
        ) : null}

        {isClientError ? (
          <div className="rounded-3xl border border-rose-300/30 bg-rose-500/10 p-6">
            <p className="text-sm text-rose-100">Не удалось загрузить клиента.</p>
          </div>
        ) : null}

        {isContactsLoading || isContactsFetching ? (
          <div className="app-subtle-notice">
            <p className="text-sm text-slate-300">Загрузка контактов...</p>
          </div>
        ) : null}

        {isContactsError ? (
          <div className="rounded-3xl border border-rose-300/30 bg-rose-500/10 p-6">
            <p className="text-sm text-rose-100">Не удалось загрузить контакты.</p>
          </div>
        ) : null}

        {!isContactsLoading && !isContactsFetching && !isContactsError && contacts.length > 0 ? (
          <div className="grid gap-3 px-1">
            {contacts.map((contact) => (
              <TicketContactCard
                key={contact.id}
                contactName={contact.name}
                contactPosition={contact.position}
                phoneHref={normalizePhoneHref(contact.phone)}
                emailHref={normalizeEmailHref(contact.email)}
              />
            ))}
          </div>
        ) : null}

        {!isContactsLoading && !isContactsFetching && !isContactsError && contacts.length === 0 ? (
          <div className="rounded-lg border border-white/20 bg-transparent px-5 py-4 text-left">
            <p className="text-[16px] font-semibold leading-snug tracking-tight text-slate-50">Контактов пока нет</p>
            <p className="mt-2 text-[16px] leading-snug text-slate-200/85">
              У этого клиента пока нет контактов.
            </p>
          </div>
        ) : null}
      </section>
    </PageShell>
  );
}
