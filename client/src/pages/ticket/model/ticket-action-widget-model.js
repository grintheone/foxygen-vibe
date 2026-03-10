export function resolveTicketActionState({ currentUserId, ticket }) {
  if (
    !ticket ||
    (ticket.status !== "assigned" &&
      ticket.status !== "inWork" &&
      ticket.status !== "worksDone")
  ) {
    return null;
  }

  const canPatch =
    Boolean(currentUserId) &&
    Boolean(ticket.executor) &&
    ticket.executor === currentUserId;

  if (!canPatch) {
    return null;
  }

  if (ticket.status === "inWork") {
    return {
      colorClassName: "bg-emerald-500 hover:bg-emerald-400",
      actionType: "patch",
      isEnabled: true,
      nextStatus: "worksDone",
      isVisible: true,
      title: "Завершить работу",
    };
  }

  if (ticket.status === "worksDone") {
    return {
      colorClassName: "bg-emerald-500 hover:bg-emerald-400",
      actionType: "openReportSheet",
      hasSuccessIcon: true,
      isEnabled: true,
      isVisible: true,
      title: "Написать отчет и закрыть тикет",
    };
  }

  return {
    colorClassName: "bg-[#6A3BF2] hover:bg-[#7C52F5]",
    actionType: "patch",
    isEnabled: true,
    nextStatus: "inWork",
    isVisible: true,
    title: "Начать работу",
  };
}

export function buildTicketPatchPayload({ actionState, ticket }) {
  if (!actionState?.nextStatus || !ticket) {
    return null;
  }

  if (actionState.nextStatus === "worksDone") {
    return {
      status: actionState.nextStatus,
      workfinished_at: new Date().toISOString(),
    };
  }

  return {
    status: actionState.nextStatus,
    workstarted_at: new Date().toISOString(),
  };
}
