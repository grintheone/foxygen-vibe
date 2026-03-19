import { lazy, Suspense, useState } from "react";
import { useNavigate, useParams } from "react-router";
import { useAuth } from "../../../features/auth";
import { routePaths } from "../../../shared/config/routes";
import {
    downloadTicketAttachmentFile,
    useGetTicketByIdQuery,
    usePatchTicketMutation,
    useUploadTicketAttachmentMutation,
} from "../../../shared/api/tickets-api";
import { PageShell } from "../../../shared/ui/page-shell";
import { useTicketViewModel } from "../lib/use-ticket-view-model";
import { buildTicketPatchPayload, resolveTicketActionState } from "../model/ticket-action-widget-model";
import { TicketContactCard } from "./components/ticket-contact-card";
import { TicketHeader } from "./components/ticket-header";
import { TicketHistorySection } from "./components/ticket-history-section";
import { TicketNavigationCard } from "./components/ticket-navigation-card";
import { TicketStatusActionWidget } from "./components/ticket-status-action-widget";
import { TicketSummaryCard } from "./components/ticket-summary-card";
import { TicketWorkResultSection } from "./components/ticket-work-result-section";

const TicketAssignmentSheet = lazy(() =>
    import("./components/ticket-assignment-sheet").then((module) => ({
        default: module.TicketAssignmentSheet,
    })),
);
const TicketReportSheet = lazy(() =>
    import("./components/ticket-report-sheet").then((module) => ({
        default: module.TicketReportSheet,
    })),
);

export function TicketPage() {
    const navigate = useNavigate();
    const { ticketId } = useParams();
    const { session } = useAuth();
    const [actionError, setActionError] = useState("");
    const [assignmentSubmitError, setAssignmentSubmitError] = useState("");
    const [isAssignmentSheetOpen, setIsAssignmentSheetOpen] = useState(false);
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
    } = useTicketViewModel(ticket);
    const [patchTicket, { isLoading: isPatching }] = usePatchTicketMutation();
    const [uploadTicketAttachment, { isLoading: isUploadingAttachment }] = useUploadTicketAttachmentMutation();
    const hasWorkResult = Boolean(ticket?.result?.trim());
    const hasContactData = Boolean(
        ticket?.contactName?.trim() || ticket?.contactPosition?.trim() || phoneHref || emailHref,
    );

    const actionState = resolveTicketActionState({
        currentUserDepartment: session?.department || "",
        currentUserId: session?.user_id || "",
        currentUserRole: session?.role || "",
        ticket,
    });
    const hasVisibleActionWidget = Boolean(actionState?.isVisible);
    const isInitialTicketLoad = !ticket && (isLoading || isFetching);
    const isReportSubmitting = isPatching || isUploadingAttachment;

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

        if (actionState.actionType === "openAssignmentSheet") {
            setActionError("");
            setAssignmentSubmitError("");
            setIsAssignmentSheetOpen(true);
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

    async function handleAssignEngineer(patch) {
        if (!ticket?.id) {
            return null;
        }

        setAssignmentSubmitError("");

        try {
            const response = await patchTicket({
                patch,
                ticketId: ticket.id,
            }).unwrap();
            setIsAssignmentSheetOpen(false);
            return response;
        } catch (error) {
            setAssignmentSubmitError(resolveActionErrorMessage(error));
            throw error;
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

    async function handleUploadAttachment(file) {
        if (!ticket?.id) {
            return null;
        }

        return uploadTicketAttachment({
            file,
            ticketId: ticket.id,
        }).unwrap();
    }

    async function handleDownloadAttachment(attachment) {
        if (!ticket?.id || !attachment?.id) {
            return;
        }

        await downloadTicketAttachmentFile({
            attachmentId: attachment.id,
            fileName: attachment.name,
            ticketId: ticket.id,
        });
    }

    return (
        <PageShell>
            <section
                className={`w-full space-y-6 transition ${hasVisibleActionWidget ? "pb-28" : ""} ${
                    isReportSheetOpen || isAssignmentSheetOpen ? "brightness-75" : ""
                }`}
            >
                <TicketHeader
                    ticketNumber={ticketNumber}
                    isInWork={isInWork}
                    statusIcon={statusIcon}
                    statusAlt={statusAlt}
                    finishedDate={finishedDate}
                    onBack={() => navigate(-1)}
                />

                {isInitialTicketLoad ? (
                    <div className="rounded-3xl border border-white/10 bg-white/5 p-6">
                        <p className="text-sm text-slate-300">Загрузка тикета...</p>
                    </div>
                ) : null}

                {ticket && isFetching ? (
                    <div className="rounded-3xl border border-white/10 bg-white/5 p-4">
                        <p className="text-xs font-semibold uppercase tracking-[0.24em] text-slate-400">
                            Обновляем данные тикета...
                        </p>
                    </div>
                ) : null}

                {isError ? (
                    <div className="rounded-3xl border border-rose-300/30 bg-rose-500/10 p-6">
                        <p className="text-sm text-rose-100">Не удалось загрузить тикет.</p>
                    </div>
                ) : null}

                {!isError && ticket ? (
                    <>
                        <TicketSummaryCard
                            reasonValue={reasonValue}
                            deadlineDisplay={deadlineDisplay}
                            description={ticket.description}
                            referenceTicket={ticket.referenceTicket}
                        />

                        <section className="space-y-3">
                            <h2 className="text-xl font-semibold tracking-tight text-slate-300 sm:text-2xl">Оборудование</h2>
                            <TicketNavigationCard
                                onClick={handleOpenDevice}
                                disabled={!canOpenDevice}
                                value={ticket.deviceName}
                                subtitle={`С/Н: ${ticket.deviceSerialNumber || "Не указано"}`}
                            />
                        </section>

                        <section className="space-y-3">
                            <h2 className="text-xl font-semibold tracking-tight text-slate-300 sm:text-2xl">Клиент</h2>
                            <TicketNavigationCard
                                onClick={handleOpenClient}
                                disabled={!canOpenClient}
                                value={ticket.clientName}
                                subtitle={ticket.clientAddress}
                            />
                            {hasContactData ? (
                                <TicketContactCard
                                    contactName={ticket.contactName}
                                    contactPosition={ticket.contactPosition}
                                    phoneHref={phoneHref}
                                    emailHref={emailHref}
                                />
                            ) : null}
                        </section>

                        {hasWorkResult ? (
                            <TicketWorkResultSection
                                ticket={ticket}
                                workDuration={workDuration}
                                onDownloadAttachment={handleDownloadAttachment}
                            />
                        ) : null}
                        <TicketHistorySection ticket={ticket} />
                    </>
                ) : null}
            </section>

            <TicketStatusActionWidget
                actionState={actionState}
                errorMessage={actionError}
                isLoading={isPatching}
                onSubmit={handleTicketAction}
            />

            {isAssignmentSheetOpen ? (
                <Suspense fallback={null}>
                    <TicketAssignmentSheet
                        isOpen={isAssignmentSheetOpen}
                        isSubmitting={isPatching}
                        onClose={() => {
                            setAssignmentSubmitError("");
                            setIsAssignmentSheetOpen(false);
                        }}
                        onSubmitAssign={handleAssignEngineer}
                        submitError={assignmentSubmitError}
                        ticket={ticket}
                    />
                </Suspense>
            ) : null}

            {isReportSheetOpen ? (
                <Suspense fallback={null}>
                    <TicketReportSheet
                        isOpen={isReportSheetOpen}
                        isSubmitting={isReportSubmitting}
                        onClose={() => {
                            setReportSubmitError("");
                            setIsReportSheetOpen(false);
                        }}
                        onDownloadAttachment={handleDownloadAttachment}
                        onSubmitClose={handleCloseTicketReport}
                        onUploadAttachment={handleUploadAttachment}
                        resolvedReason={ticket?.resolvedReason}
                        deviceName={ticket?.deviceName}
                        clientName={ticket?.clientName}
                        submitError={reportSubmitError}
                        ticketNumber={ticketNumber}
                    />
                </Suspense>
            ) : null}
        </PageShell>
    );
}
