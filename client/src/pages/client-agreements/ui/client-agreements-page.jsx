import { useNavigate, useParams } from "react-router";
import { useGetClientAgreementsQuery, useGetClientByIdQuery } from "../../../shared/api/tickets-api";
import { routePaths } from "../../../shared/config/routes";
import { PageShell } from "../../../shared/ui/page-shell";
import { TicketDeviceCard } from "../../ticket/ui/components/ticket-device-section";
import { TicketHeader } from "../../ticket/ui/components/ticket-header";

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

    const pageTitle = client?.title?.trim() || "Сервисные условия клиента";

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

                {isAgreementsLoading || isAgreementsFetching ? (
                    <div className="app-subtle-notice px-1">
                        <p className="text-sm text-slate-300">Загрузка оборудования...</p>
                    </div>
                ) : null}

                {isAgreementsError ? (
                    <div className="rounded-3xl border border-rose-300/30 bg-rose-500/10 p-6">
                        <p className="text-sm text-rose-100">Не удалось загрузить оборудование.</p>
                    </div>
                ) : null}

                {!isAgreementsLoading && !isAgreementsFetching && !isAgreementsError && agreements.length > 0 ? (
                    <div className="grid gap-3 px-1">
                        {agreements.map((agreement) => (
                            <TicketDeviceCard
                                key={agreement.id}
                                deviceName={agreement.deviceName}
                                serialNumber={agreement.deviceSerialNumber}
                                disabled={!agreement.device}
                                onOpenDevice={() => {
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
                    <div className="rounded-lg border border-white/20 bg-transparent px-5 py-4 text-left">
                        <p className="text-[16px] font-semibold leading-snug tracking-tight text-slate-50">
                            Оборудования пока нет
                        </p>
                        <p className="mt-2 text-[16px] leading-snug text-slate-200/85">
                            У этого клиента пока нет оборудования по договорам.
                        </p>
                    </div>
                ) : null}
            </section>
        </PageShell>
    );
}
