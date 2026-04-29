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
import { TicketClientSection } from "./components/ticket-client-section";
import { TicketContactCard } from "./components/ticket-contact-card";
import { TicketDeviceSection } from "./components/ticket-device-section";
import { TicketHeader } from "./components/ticket-header";
import { TicketHistorySection } from "./components/ticket-history-section";
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

const mockHistoryTicketData = {
    assignedBy: "mock-coordinator",
    assignedByAvatarUrl: "",
    assignedByName: "Анна Смирнова",
    assigned_at: "2026-04-24T08:15:00.000Z",
    closed_at: "2026-04-24T11:30:00.000Z",
    executor: "mock-engineer",
    executorAvatarUrl: "",
    executorName: "Илья Волков",
    workfinished_at: "2026-04-24T10:55:00.000Z",
    workstarted_at: "2026-04-24T09:05:00.000Z",
};

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
    const historySectionTicket = import.meta.env.DEV && ticket ? { ...ticket, ...mockHistoryTicketData } : ticket;

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
                className={`w-full space-y-6 transition ${hasVisibleActionWidget ? "pb-[4.5rem]" : ""} ${
                    isReportSheetOpen || isAssignmentSheetOpen ? "brightness-75" : ""
                }`}
            >
                <TicketHeader
                    ticketNumber={ticketNumber}
                    onBack={() => navigate(-1)}
                />

                {isInitialTicketLoad ? (
                    <div className="app-subtle-notice">
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

                        {hasWorkResult ? (
                            <TicketWorkResultSection
                                ticket={ticket}
                                workDuration={workDuration}
                                onDownloadAttachment={handleDownloadAttachment}
                            />
                        ) : null}

                        <TicketDeviceSection
                            onOpenDevice={handleOpenDevice}
                            disabled={!canOpenDevice}
                            deviceName={ticket.deviceName}
                            serialNumber={ticket.deviceSerialNumber}
                        />

                        <section className="space-y-4">
                            <TicketClientSection
                                onOpenClient={handleOpenClient}
                                disabled={!canOpenClient}
                                clientName={ticket.clientName}
                                clientAddress={ticket.clientAddress}
                            />
                            <TicketContactCard
                                contactName={ticket.contactName}
                                contactPosition={ticket.contactPosition}
                                phoneHref={phoneHref}
                                emailHref={emailHref}
                            />
                        </section>
                        <TicketHistorySection ticket={historySectionTicket} />
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
