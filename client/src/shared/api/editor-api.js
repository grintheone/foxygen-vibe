import { createApi } from "@reduxjs/toolkit/query/react";
import { baseQueryWithAuth } from "./authenticated-fetch";

export const editorApi = createApi({
  reducerPath: "editorApi",
  baseQuery: baseQueryWithAuth,
  tagTypes: ["EditorClassificator", "EditorClient", "EditorContact", "EditorDevice", "EditorRegion"],
  endpoints: (builder) => ({
    getEditorClassificators: builder.query({
      query: () => "api/editor/classificators",
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
    getEditorRegions: builder.query({
      query: () => "api/editor/regions",
      providesTags: ["EditorRegion"],
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
  }),
});

export const {
  useGetEditorClassificatorsQuery,
  useGetEditorClientByIdQuery,
  useGetEditorClientsQuery,
  useGetEditorContactByIdQuery,
  useGetEditorContactsQuery,
  useGetEditorDeviceByIdQuery,
  useGetEditorDevicesQuery,
  useGetEditorRegionsQuery,
  usePatchEditorContactMutation,
  usePatchEditorClientMutation,
  usePatchEditorDeviceMutation,
} = editorApi;
