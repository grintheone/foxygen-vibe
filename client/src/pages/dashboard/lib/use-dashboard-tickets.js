import { useMemo } from "react";
import { useGetMyTicketsQuery } from "../../../shared/api/tickets-api";
import { LIVE_DASHBOARD_POLLING_INTERVAL_MS } from "./dashboard-refresh";

export function useDashboardTickets(executorId) {
  const { data = [], isError, isLoading } = useGetMyTicketsQuery(executorId, {
    pollingInterval: LIVE_DASHBOARD_POLLING_INTERVAL_MS,
    refetchOnFocus: true,
    refetchOnReconnect: true,
    skip: !executorId,
    skipPollingIfUnfocused: true,
  });

  const tickets = useMemo(
    () =>
      data.filter(
        (ticket) =>
          (ticket.status === "inWork" || ticket.status === "worksDone"),
      ),
    [data],
  );

  const assignedTickets = useMemo(
    () =>
      data.filter((ticket) => ticket.status === "assigned"),
    [data],
  );

  return {
    assignedTickets,
    isError,
    isLoading,
    tickets,
  };
}
