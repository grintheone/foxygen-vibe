import { useNavigate, useParams } from "react-router";
import {
  useGetClientAgreementsQuery,
  useGetClientByIdQuery,
} from "../../../shared/api/tickets-api";
import { routePaths } from "../../../shared/config/routes";
import { NavigationCard } from "../../../shared/ui/navigation-card";
import { PageShell } from "../../../shared/ui/page-shell";

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

function ClientAgreementsHeader({ title, onBack }) {
  return (
    <header className="rounded-3xl border border-white/10 bg-slate-950/35 p-6 shadow-xl shadow-black/20 backdrop-blur">
      <BackButton onClick={onBack} />
      <p className="mt-4 text-xs font-semibold uppercase tracking-[0.3em] text-slate-400">Сервисные услуги</p>
      <h1 className="mt-2 text-3xl font-bold tracking-tight sm:text-4xl">{title}</h1>
    </header>
  );
}

export function ClientAgreementsPage() {
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
    data: agreements = [],
    isError: isAgreementsError,
    isFetching: isAgreementsFetching,
    isLoading: isAgreementsLoading,
  } = useGetClientAgreementsQuery(
    { clientId },
    {
      skip: !clientId,
    },
  );

  const pageTitle = client?.title?.trim() || "Сервисные услуги клиента";

  return (
    <PageShell>
      <section className="w-full space-y-6">
        <ClientAgreementsHeader
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

        {isAgreementsLoading || isAgreementsFetching ? (
          <div className="rounded-3xl border border-white/10 bg-white/5 p-6">
            <p className="text-sm text-slate-300">Загрузка оборудования...</p>
          </div>
        ) : null}

        {isAgreementsError ? (
          <div className="rounded-3xl border border-rose-300/30 bg-rose-500/10 p-6">
            <p className="text-sm text-rose-100">Не удалось загрузить оборудование.</p>
          </div>
        ) : null}

        {!isAgreementsLoading && !isAgreementsFetching && !isAgreementsError && agreements.length > 0 ? (
          <div className="grid gap-3">
            {agreements.map((agreement) => (
              <NavigationCard
                key={agreement.id}
                value={agreement.deviceName}
                subtitle={`С/Н: ${agreement.deviceSerialNumber || "Не указано"}`}
                disabled={!agreement.device}
                onClick={() => {
                  if (!agreement.device) {
                    return;
                  }

                  navigate(routePaths.deviceById(agreement.device));
                }}
              />
            ))}
          </div>
        ) : null}

        {!isAgreementsLoading && !isAgreementsFetching && !isAgreementsError && agreements.length === 0 ? (
          <div className="rounded-3xl border border-white/10 bg-white/5 p-6">
            <p className="text-sm text-slate-300">У этого клиента пока нет оборудования по договорам.</p>
          </div>
        ) : null}
      </section>
    </PageShell>
  );
}
