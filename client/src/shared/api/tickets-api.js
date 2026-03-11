import { createApi } from "@reduxjs/toolkit/query/react";
import { baseQueryWithAuth, fetchWithAuth } from "./authenticated-fetch";

async function readError(response, fallbackMessage) {
  const errorMessage = await response.text();

  return errorMessage || fallbackMessage;
}

export const ticketsApi = createApi({
  tagTypes: ["Client", "Comment", "Department", "Device", "Ticket", "Tickets"],
  reducerPath: "ticketsApi",
  baseQuery: baseQueryWithAuth,
  endpoints: (builder) => ({
    getDepartments: builder.query({
      query: () => "api/departments",
      providesTags: ["Department"],
    }),
    getClientById: builder.query({
      query: (clientId) => `api/clients/${clientId}`,
      providesTags: (_, __, clientId) => [{ type: "Client", id: clientId }],
    }),
    getDeviceById: builder.query({
      query: (deviceId) => `api/devices/${deviceId}`,
      providesTags: (_, __, deviceId) => [{ type: "Device", id: deviceId }],
    }),
    getComments: builder.query({
      query: (referenceId) => ({
        params: {
          reference_id: referenceId,
        },
        url: "api/comments",
      }),
      providesTags: (_, __, referenceId) => [{ type: "Comment", id: referenceId }],
    }),
    getClientTickets: builder.query({
      query: ({ clientId, limit, status }) => ({
        params: {
          ...(limit ? { limit } : {}),
          ...(status ? { status } : {}),
        },
        url: `api/clients/${clientId}/tickets`,
      }),
      providesTags: (_, __, { clientId }) => [{ type: "Client", id: clientId }],
    }),
    getClientContacts: builder.query({
      query: ({ clientId, limit }) => ({
        params: {
          ...(limit ? { limit } : {}),
        },
        url: `api/clients/${clientId}/contacts`,
      }),
      providesTags: (_, __, { clientId }) => [{ type: "Client", id: clientId }],
    }),
    getClientAgreements: builder.query({
      query: ({ clientId, limit }) => ({
        params: {
          ...(limit ? { limit } : {}),
        },
        url: `api/clients/${clientId}/agreements`,
      }),
      providesTags: (_, __, { clientId }) => [{ type: "Client", id: clientId }],
    }),
    getDeviceTickets: builder.query({
      query: ({ deviceId, limit, status }) => ({
        params: {
          ...(limit ? { limit } : {}),
          ...(status ? { status } : {}),
        },
        url: `api/devices/${deviceId}/tickets`,
      }),
      providesTags: (_, __, { deviceId }) => [{ type: "Device", id: deviceId }],
    }),
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
    addComment: builder.mutation({
      query: ({ referenceId, text }) => ({
        body: {
          reference_id: referenceId,
          text,
        },
        headers: {
          "Content-Type": "application/json",
        },
        method: "POST",
        url: "api/comments",
      }),
      invalidatesTags: (_, __, { referenceId }) => [{ type: "Comment", id: referenceId }],
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
  const response = await fetchWithAuth(
    `/api/tickets/${ticketId}/attachments/${attachmentId}/download`,
  );

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

export async function loadTicketAttachmentPreviewUrl(downloadUrl) {
  if (!downloadUrl) {
    return null;
  }

  const response = await fetchWithAuth(downloadUrl);

  if (!response.ok) {
    throw new Error(await readError(response, "Не удалось загрузить превью вложения."));
  }

  const blob = await response.blob();
  return window.URL.createObjectURL(blob);
}

export const {
  useAddCommentMutation,
  useGetClientByIdQuery,
  useGetClientAgreementsQuery,
  useGetClientContactsQuery,
  useGetCommentsQuery,
  useGetClientTicketsQuery,
  useGetDeviceByIdQuery,
  useGetDeviceTicketsQuery,
  useGetDepartmentsQuery,
  useGetDepartmentTicketsQuery,
  useGetMyTicketsQuery,
  useGetTicketByIdQuery,
  usePatchTicketMutation,
  useUploadTicketAttachmentMutation,
} = ticketsApi;
