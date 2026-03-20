import { createApi } from "@reduxjs/toolkit/query/react";
import { baseQueryWithAuth } from "./authenticated-fetch";

export const editorApi = createApi({
  reducerPath: "editorApi",
  baseQuery: baseQueryWithAuth,
  tagTypes: [
    "EditorAgreement",
    "EditorClassificator",
    "EditorClient",
    "EditorContact",
    "EditorDevice",
    "EditorManufacturer",
    "EditorAccount",
    "EditorRegion",
    "EditorResearchType",
    "EditorTicket",
  ],
  endpoints: (builder) => ({
    getEditorAgreements: builder.query({
      query: ({ limit = 50, q = "" } = {}) => ({
        params: {
          ...(limit ? { limit } : {}),
          ...(q ? { q } : {}),
        },
        url: "api/editor/agreements",
      }),
      providesTags: (result) => [
        { type: "EditorAgreement", id: "LIST" },
        ...(Array.isArray(result)
          ? result.map((agreement) => ({
              type: "EditorAgreement",
              id: agreement.id,
            }))
          : []),
      ],
    }),
    getEditorAgreementById: builder.query({
      query: (agreementId) => `api/editor/agreements/${agreementId}`,
      providesTags: (_, __, agreementId) => [{ type: "EditorAgreement", id: agreementId }],
    }),
    getEditorClassificators: builder.query({
      query: ({ limit = 100, q = "" } = {}) => ({
        params: {
          ...(limit ? { limit } : {}),
          ...(q ? { q } : {}),
        },
        url: "api/editor/classificators",
      }),
      providesTags: (result) => [
        { type: "EditorClassificator", id: "LIST" },
        ...(Array.isArray(result)
          ? result.map((classificator) => ({
              type: "EditorClassificator",
              id: classificator.id,
            }))
          : []),
      ],
    }),
    getEditorClassificatorById: builder.query({
      query: (classificatorId) => `api/editor/classificators/${classificatorId}`,
      providesTags: (_, __, classificatorId) => [{ type: "EditorClassificator", id: classificatorId }],
    }),
    getEditorClients: builder.query({
      query: ({ limit = 50, q = "" } = {}) => ({
        params: {
          ...(limit ? { limit } : {}),
          ...(q ? { q } : {}),
        },
        url: "api/editor/clients",
      }),
      providesTags: (result) => [
        { type: "EditorClient", id: "LIST" },
        ...(Array.isArray(result)
          ? result.map((client) => ({
              type: "EditorClient",
              id: client.id,
            }))
          : []),
      ],
    }),
    getEditorClientById: builder.query({
      query: (clientId) => `api/editor/clients/${clientId}`,
      providesTags: (_, __, clientId) => [{ type: "EditorClient", id: clientId }],
    }),
    getEditorContacts: builder.query({
      query: ({ limit = 50, q = "" } = {}) => ({
        params: {
          ...(limit ? { limit } : {}),
          ...(q ? { q } : {}),
        },
        url: "api/editor/contacts",
      }),
      providesTags: (result) => [
        { type: "EditorContact", id: "LIST" },
        ...(Array.isArray(result)
          ? result.map((contact) => ({
              type: "EditorContact",
              id: contact.id,
            }))
          : []),
      ],
    }),
    getEditorContactById: builder.query({
      query: (contactId) => `api/editor/contacts/${contactId}`,
      providesTags: (_, __, contactId) => [{ type: "EditorContact", id: contactId }],
    }),
    getEditorDevices: builder.query({
      query: ({ limit = 50, q = "" } = {}) => ({
        params: {
          ...(limit ? { limit } : {}),
          ...(q ? { q } : {}),
        },
        url: "api/editor/devices",
      }),
      providesTags: (result) => [
        { type: "EditorDevice", id: "LIST" },
        ...(Array.isArray(result)
          ? result.map((device) => ({
              type: "EditorDevice",
              id: device.id,
            }))
          : []),
      ],
    }),
    getEditorDeviceById: builder.query({
      query: (deviceId) => `api/editor/devices/${deviceId}`,
      providesTags: (_, __, deviceId) => [{ type: "EditorDevice", id: deviceId }],
    }),
    getEditorDeviceOptions: builder.query({
      query: ({ q = "" } = {}) => ({
        params: {
          ...(q ? { q } : {}),
        },
        url: "api/editor/device-options",
      }),
      providesTags: ["EditorDevice"],
    }),
    getEditorTickets: builder.query({
      query: ({ limit = 50, q = "" } = {}) => ({
        params: {
          ...(limit ? { limit } : {}),
          ...(q ? { q } : {}),
        },
        url: "api/editor/tickets",
      }),
      providesTags: (result) => [
        { type: "EditorTicket", id: "LIST" },
        ...(Array.isArray(result)
          ? result.map((ticket) => ({
              type: "EditorTicket",
              id: ticket.id,
            }))
          : []),
      ],
    }),
    getEditorTicketById: builder.query({
      query: (ticketId) => `api/editor/tickets/${ticketId}`,
      providesTags: (_, __, ticketId) => [{ type: "EditorTicket", id: ticketId }],
    }),
    getEditorAccounts: builder.query({
      query: ({ limit = 50, q = "" } = {}) => ({
        params: {
          ...(limit ? { limit } : {}),
          ...(q ? { q } : {}),
        },
        url: "api/editor/accounts",
      }),
      providesTags: (result) => [
        { type: "EditorAccount", id: "LIST" },
        ...(Array.isArray(result)
          ? result.map((account) => ({
              type: "EditorAccount",
              id: account.id,
            }))
          : []),
      ],
    }),
    getEditorRegions: builder.query({
      query: () => "api/editor/regions",
      providesTags: ["EditorRegion"],
    }),
    getEditorManufacturers: builder.query({
      query: () => "api/editor/manufacturers",
      providesTags: (result) => [
        { type: "EditorManufacturer", id: "LIST" },
        ...(Array.isArray(result)
          ? result.map((manufacturer) => ({
              type: "EditorManufacturer",
              id: manufacturer.id,
            }))
          : []),
      ],
    }),
    getEditorResearchTypes: builder.query({
      query: () => "api/editor/research-types",
      providesTags: (result) => [
        { type: "EditorResearchType", id: "LIST" },
        ...(Array.isArray(result)
          ? result.map((researchType) => ({
              type: "EditorResearchType",
              id: researchType.id,
            }))
          : []),
      ],
    }),
    getEditorTicketStatuses: builder.query({
      query: () => "api/editor/ticket-statuses",
      providesTags: ["EditorTicket"],
    }),
    getEditorTicketTypes: builder.query({
      query: () => "api/editor/ticket-types",
      providesTags: ["EditorTicket"],
    }),
    patchEditorAgreement: builder.mutation({
      query: ({ agreementId, patch }) => ({
        body: patch,
        headers: {
          "Content-Type": "application/json",
        },
        method: "PATCH",
        url: `api/editor/agreements/${agreementId}`,
      }),
      invalidatesTags: (_, __, { agreementId }) => [
        { type: "EditorAgreement", id: agreementId },
        { type: "EditorAgreement", id: "LIST" },
      ],
    }),
    patchEditorClassificator: builder.mutation({
      query: ({ classificatorId, patch }) => ({
        body: patch,
        headers: {
          "Content-Type": "application/json",
        },
        method: "PATCH",
        url: `api/editor/classificators/${classificatorId}`,
      }),
      invalidatesTags: (_, __, { classificatorId }) => [
        { type: "EditorClassificator", id: classificatorId },
        { type: "EditorClassificator", id: "LIST" },
      ],
    }),
    patchEditorClient: builder.mutation({
      query: ({ clientId, patch }) => ({
        body: patch,
        headers: {
          "Content-Type": "application/json",
        },
        method: "PATCH",
        url: `api/editor/clients/${clientId}`,
      }),
      invalidatesTags: (_, __, { clientId }) => [
        { type: "EditorClient", id: clientId },
        { type: "EditorClient", id: "LIST" },
      ],
    }),
    patchEditorContact: builder.mutation({
      query: ({ contactId, patch }) => ({
        body: patch,
        headers: {
          "Content-Type": "application/json",
        },
        method: "PATCH",
        url: `api/editor/contacts/${contactId}`,
      }),
      invalidatesTags: (_, __, { contactId }) => [
        { type: "EditorContact", id: contactId },
        { type: "EditorContact", id: "LIST" },
      ],
    }),
    patchEditorDevice: builder.mutation({
      query: ({ deviceId, patch }) => ({
        body: patch,
        headers: {
          "Content-Type": "application/json",
        },
        method: "PATCH",
        url: `api/editor/devices/${deviceId}`,
      }),
      invalidatesTags: (_, __, { deviceId }) => [
        { type: "EditorDevice", id: deviceId },
        { type: "EditorDevice", id: "LIST" },
      ],
    }),
    patchEditorTicket: builder.mutation({
      query: ({ ticketId, patch }) => ({
        body: patch,
        headers: {
          "Content-Type": "application/json",
        },
        method: "PATCH",
        url: `api/editor/tickets/${ticketId}`,
      }),
      invalidatesTags: (_, __, { ticketId }) => [
        { type: "EditorTicket", id: ticketId },
        { type: "EditorTicket", id: "LIST" },
      ],
    }),
  }),
});

export const {
  useGetEditorAccountsQuery,
  useGetEditorAgreementByIdQuery,
  useGetEditorAgreementsQuery,
  useGetEditorClassificatorByIdQuery,
  useGetEditorClassificatorsQuery,
  useGetEditorClientByIdQuery,
  useGetEditorClientsQuery,
  useGetEditorContactByIdQuery,
  useGetEditorContactsQuery,
  useGetEditorDeviceByIdQuery,
  useGetEditorDeviceOptionsQuery,
  useGetEditorDevicesQuery,
  useGetEditorManufacturersQuery,
  useGetEditorRegionsQuery,
  useGetEditorResearchTypesQuery,
  useGetEditorTicketByIdQuery,
  useGetEditorTicketsQuery,
  useGetEditorTicketStatusesQuery,
  useGetEditorTicketTypesQuery,
  usePatchEditorAgreementMutation,
  usePatchEditorClassificatorMutation,
  usePatchEditorContactMutation,
  usePatchEditorClientMutation,
  usePatchEditorDeviceMutation,
  usePatchEditorTicketMutation,
} = editorApi;
