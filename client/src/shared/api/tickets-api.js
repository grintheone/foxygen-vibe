import { createApi, fetchBaseQuery } from "@reduxjs/toolkit/query/react";
import { getAccessToken } from "../lib/auth-tokens";

export const ticketsApi = createApi({
  tagTypes: ["Ticket", "Tickets"],
  reducerPath: "ticketsApi",
  baseQuery: fetchBaseQuery({
    baseUrl: "/",
    prepareHeaders: (headers) => {
      const accessToken = getAccessToken();
      if (accessToken) {
        headers.set("Authorization", `Bearer ${accessToken}`);
      }

      return headers;
    },
  }),
  endpoints: (builder) => ({
    getMyTickets: builder.query({
      query: () => "api/tickets",
      providesTags: ["Tickets"],
    }),
    getTicketById: builder.query({
      query: (ticketId) => `api/tickets/${ticketId}`,
      providesTags: (_, __, ticketId) => [{ type: "Ticket", id: ticketId }],
    }),
    getDepartmentTickets: builder.query({
      query: () => "api/tickets/department",
      providesTags: ["Tickets"],
    }),
    patchTicket: builder.mutation({
      query: ({ ticketId, patch }) => ({
        body: patch,
        headers: {
          "Content-Type": "application/json",
        },
        method: "PATCH",
        url: `api/tickets/${ticketId}`,
      }),
      invalidatesTags: (_, __, { ticketId }) => [
        "Tickets",
        { type: "Ticket", id: ticketId },
      ],
    }),
  }),
});

export const {
  useGetDepartmentTicketsQuery,
  useGetMyTicketsQuery,
  useGetTicketByIdQuery,
  usePatchTicketMutation,
} = ticketsApi;
