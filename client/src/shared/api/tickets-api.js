import { createApi, fetchBaseQuery } from "@reduxjs/toolkit/query/react";
import { getAccessToken } from "../lib/auth-tokens";

async function readError(response, fallbackMessage) {
  const errorMessage = await response.text();

  return errorMessage || fallbackMessage;
}

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
    uploadTicketAttachment: builder.mutation({
      query: ({ file, ticketId }) => {
        const body = new FormData();
        body.append("file", file);

        return {
          body,
          method: "POST",
          url: `api/tickets/${ticketId}/attachments`,
        };
      },
      invalidatesTags: (_, __, { ticketId }) => [{ type: "Ticket", id: ticketId }],
    }),
  }),
});

export async function downloadTicketAttachmentFile({ attachmentId, fileName, ticketId }) {
  const accessToken = getAccessToken();
  const response = await fetch(`/api/tickets/${ticketId}/attachments/${attachmentId}/download`, {
    headers: accessToken
      ? {
          Authorization: `Bearer ${accessToken}`,
        }
      : undefined,
  });

  if (!response.ok) {
    throw new Error(await readError(response, "Не удалось скачать вложение."));
  }

  const blob = await response.blob();
  const objectUrl = window.URL.createObjectURL(blob);
  const link = document.createElement("a");

  link.href = objectUrl;
  link.download = fileName || "attachment";
  document.body.appendChild(link);
  link.click();
  link.remove();

  window.setTimeout(() => {
    window.URL.revokeObjectURL(objectUrl);
  }, 0);
}

export const {
  useGetDepartmentTicketsQuery,
  useGetMyTicketsQuery,
  useGetTicketByIdQuery,
  usePatchTicketMutation,
  useUploadTicketAttachmentMutation,
} = ticketsApi;
