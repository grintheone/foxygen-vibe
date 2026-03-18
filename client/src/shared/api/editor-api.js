import { createApi } from "@reduxjs/toolkit/query/react";
import { baseQueryWithAuth } from "./authenticated-fetch";

export const editorApi = createApi({
  reducerPath: "editorApi",
  baseQuery: baseQueryWithAuth,
  tagTypes: ["EditorClient", "EditorRegion"],
  endpoints: (builder) => ({
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
  }),
});

export const {
  useGetEditorClientByIdQuery,
  useGetEditorClientsQuery,
  useGetEditorRegionsQuery,
  usePatchEditorClientMutation,
} = editorApi;
