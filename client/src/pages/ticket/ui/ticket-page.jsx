import { useNavigate, useParams } from "react-router";
import { useEffect, useState } from "react";
import { useAuth } from "../../../features/auth";
import { routePaths } from "../../../shared/config/routes";
import { useGetTicketByIdQuery, usePatchTicketMutation } from "../../../shared/api/tickets-api";
import { PageShell } from "../../../shared/ui/page-shell";
import { useTicketViewModel } from "../lib/use-ticket-view-model";
import { buildTicketPatchPayload, resolveTicketActionState } from "../model/ticket-action-widget-model";
import { TicketContactCard } from "./components/ticket-contact-card";
import { TicketHeader } from "./components/ticket-header";
import { TicketHistorySection } from "./components/ticket-history-section";
import { TicketNavigationCard } from "./components/ticket-navigation-card";
import { TicketReportSheet } from "./components/ticket-report-sheet";
import { TicketStatusActionWidget } from "./components/ticket-status-action-widget";
import { TicketSummaryCard } from "./components/ticket-summary-card";
import { TicketWorkResultSection } from "./components/ticket-work-result-section";

export function TicketPage() {
    const navigate = useNavigate();
    const { ticketId } = useParams();
    const { session } = useAuth();
    const [actionError, setActionError] = useState("");
    const [reportSubmitError, setReportSubmitError] = useState("");
    const [isReportSheetOpen, setIsReportSheetOpen] = useState(false);
    const {
        data: ticket,
        isError,
        isFetching,
        isLoading,
    } = useGetTicketByIdQuery(ticketId, {
        skip: !ticketId,
    });

    const {
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
        historyActorName,
    } = useTicketViewModel(ticket);
    const [patchTicket, { isLoading: isPatching }] = usePatchTicketMutation();

    const actionState = resolveTicketActionState({
        currentUserId: session?.id || session?.user_id || "",
        ticket,
    });

    useEffect(() => {
        if (!isReportSheetOpen) {
            return undefined;
        }

        const previousOverflow = document.body.style.overflow;
        document.body.style.overflow = "hidden";

        return () => {
            document.body.style.overflow = previousOverflow;
        };
    }, [isReportSheetOpen]);

    function handleOpenDevice() {
        if (!ticket?.device) {
            return;
        }

        navigate(routePaths.deviceById(ticket.device));
    }

    function handleOpenClient() {
        if (!ticket?.client) {
            return;
        }

        navigate(routePaths.clientById(ticket.client));
    }

    function resolveActionErrorMessage(error) {
        if (!error) {
            return "Не удалось обновить тикет.";
        }

        if (typeof error.data === "string") {
            return error.data;
        }

        if (typeof error.error === "string") {
            return error.error;
        }

        return "Не удалось обновить тикет.";
    }

    async function handleTicketAction() {
        if (!ticket?.id || !actionState?.isEnabled) {
            return;
        }

        if (actionState.actionType === "openReportSheet") {
            setReportSubmitError("");
            setIsReportSheetOpen(true);
            return;
        }

        const patch = buildTicketPatchPayload({
            actionState,
            ticket,
        });
        if (!patch) {
            return;
        }

        setActionError("");

        try {
            await patchTicket({
                patch,
                ticketId: ticket.id,
            }).unwrap();
        } catch (error) {
            setActionError(resolveActionErrorMessage(error));
        }
    }

    async function handleCloseTicketReport(patch) {
        if (!ticket?.id) {
            return null;
        }

        setReportSubmitError("");

        try {
            const response = await patchTicket({
                patch,
                ticketId: ticket.id,
            }).unwrap();
            return response;
        } catch (error) {
            setReportSubmitError(resolveActionErrorMessage(error));
            throw error;
        }
    }

    return (
        <PageShell>
            <section className={`w-full space-y-6 pb-28 transition ${isReportSheetOpen ? "brightness-75" : ""}`}>
                <TicketHeader
                    ticketNumber={ticketNumber}
                    isInWork={isInWork}
                    statusIcon={statusIcon}
                    statusAlt={statusAlt}
                    finishedDate={finishedDate}
                    onBack={() => navigate(routePaths.dashboard)}
                />

                {isLoading || isFetching ? (
                    <div className="rounded-3xl border border-white/10 bg-white/5 p-6">
                        <p className="text-sm text-slate-300">Загрузка тикета...</p>
                    </div>
                ) : null}

                {isError ? (
                    <div className="rounded-3xl border border-rose-300/30 bg-rose-500/10 p-6">
                        <p className="text-sm text-rose-100">Не удалось загрузить тикет.</p>
                    </div>
                ) : null}

                {!isLoading && !isFetching && !isError && ticket ? (
                    <>
                        <TicketSummaryCard
                            reasonValue={reasonValue}
                            deadlineDisplay={deadlineDisplay}
                            description={ticket.description}
                        />

                        <section className="space-y-3">
                            <h2 className="text-3xl font-semibold tracking-tight text-slate-300">Оборудование</h2>
                            <TicketNavigationCard
                                onClick={handleOpenDevice}
                                disabled={!canOpenDevice}
                                value={ticket.deviceName}
                                subtitle={`С/Н: ${ticket.deviceSerialNumber || "Не указано"}`}
                            />
                        </section>

                        <section className="space-y-3">
                            <h2 className="text-3xl font-semibold tracking-tight text-slate-300">Клиент</h2>
                            <TicketNavigationCard
                                onClick={handleOpenClient}
                                disabled={!canOpenClient}
                                value={ticket.clientName}
                                subtitle={ticket.clientAddress}
                            />
                            <TicketContactCard
                                contactName={ticket.contactName}
                                contactPosition={ticket.contactPosition}
                                phoneHref={phoneHref}
                                emailHref={emailHref}
                            />
                        </section>

                        <TicketWorkResultSection ticket={ticket} workDuration={workDuration} />
                        <TicketHistorySection historyActorName={historyActorName} />
                    </>
                ) : null}
            </section>

            <TicketStatusActionWidget
                actionState={actionState}
                errorMessage={actionError}
                isLoading={isPatching}
                onSubmit={handleTicketAction}
            />

            <TicketReportSheet
                isOpen={isReportSheetOpen}
                isSubmitting={isPatching}
                onClose={() => {
                    setReportSubmitError("");
                    setIsReportSheetOpen(false);
                }}
                onSubmitClose={handleCloseTicketReport}
                resolvedReason={ticket?.resolvedReason}
                deviceName={ticket?.deviceName}
                clientName={ticket?.clientName}
                submitError={reportSubmitError}
                ticketNumber={ticketNumber}
            />
        </PageShell>
    );
}
