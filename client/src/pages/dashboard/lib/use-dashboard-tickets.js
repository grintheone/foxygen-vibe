import { useMemo } from "react";
import { MOCK_EXECUTOR_ID, MOCK_TICKETS } from "../model/mock-dashboard-data";

export function useDashboardTickets(executorId) {
  const resolvedExecutorId = executorId || MOCK_EXECUTOR_ID;

  const tickets = useMemo(
    () =>
      MOCK_TICKETS.filter(
        (ticket) =>
          ticket.executor === resolvedExecutorId &&
          (ticket.status === "inWork" || ticket.status === "worksDone"),
      ),
    [resolvedExecutorId],
  );

  const assignedTickets = useMemo(
    () =>
      MOCK_TICKETS.filter(
        (ticket) => ticket.executor === resolvedExecutorId && ticket.status === "assigned",
      ),
    [resolvedExecutorId],
  );

  return {
    tickets,
    assignedTickets,
  };
}
