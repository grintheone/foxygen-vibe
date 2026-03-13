import { useNavigate, useParams } from "react-router";
import {
  useGetClientByIdQuery,
  useGetClientContactsQuery,
} from "../../../shared/api/tickets-api";
import { routePaths } from "../../../shared/config/routes";
import { ContactCard } from "../../../shared/ui/contact-card";
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

function BackButton({ onClick }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="inline-flex h-11 w-11 items-center justify-center rounded-2xl bg-[#6A3BF2] text-white transition hover:bg-[#7C52F5]"
      aria-label="Назад"
    >
      <svg
        xmlns="http://www.w3.org/2000/svg"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="2.5"
        strokeLinecap="round"
        strokeLinejoin="round"
        className="h-5 w-5"
        aria-hidden="true"
      >
        <path d="M15 18l-6-6 6-6" />
      </svg>
    </button>
  );
}

function ClientContactsHeader({ title, onBack }) {
  return (
    <header className="rounded-3xl border border-white/10 bg-slate-950/35 p-6 shadow-xl shadow-black/20 backdrop-blur">
      <BackButton onClick={onBack} />
      <p className="mt-4 text-xs font-semibold uppercase tracking-[0.3em] text-slate-400">Контакты</p>
      <h1 className="mt-2 text-3xl font-bold tracking-tight sm:text-4xl">{title}</h1>
    </header>
  );
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
        <ClientContactsHeader
          title={pageTitle}
          onBack={() => navigate(routePaths.clientById(clientId))}
        />

        {isClientLoading || isClientFetching ? (
          <div className="rounded-3xl border border-white/10 bg-white/5 p-6">
            <p className="text-sm text-slate-300">Загрузка клиента...</p>
          </div>
        ) : null}

        {isClientError ? (
          <div className="rounded-3xl border border-rose-300/30 bg-rose-500/10 p-6">
            <p className="text-sm text-rose-100">Не удалось загрузить клиента.</p>
          </div>
        ) : null}

        {isContactsLoading || isContactsFetching ? (
          <div className="rounded-3xl border border-white/10 bg-white/5 p-6">
            <p className="text-sm text-slate-300">Загрузка контактов...</p>
          </div>
        ) : null}

        {isContactsError ? (
          <div className="rounded-3xl border border-rose-300/30 bg-rose-500/10 p-6">
            <p className="text-sm text-rose-100">Не удалось загрузить контакты.</p>
          </div>
        ) : null}

        {!isContactsLoading && !isContactsFetching && !isContactsError && contacts.length > 0 ? (
          <div className="grid gap-3">
            {contacts.map((contact) => (
              <ContactCard
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
          <div className="rounded-3xl border border-white/10 bg-white/5 p-6">
            <p className="text-sm text-slate-300">У этого клиента пока нет контактов.</p>
          </div>
        ) : null}
      </section>
    </PageShell>
  );
}
