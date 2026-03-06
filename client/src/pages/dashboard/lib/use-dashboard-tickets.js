import { useMemo } from "react";
import { useGetMyTicketsQuery } from "../../../shared/api/tickets-api";

export function useDashboardTickets(executorId) {
  const { data = [], isError, isFetching, isLoading } = useGetMyTicketsQuery(undefined, {
    skip: !executorId,
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
    isLoading: isLoading || isFetching,
    tickets,
  };
}
