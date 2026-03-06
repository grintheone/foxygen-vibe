import { createApi, fetchBaseQuery } from "@reduxjs/toolkit/query/react";
import { getAccessToken } from "../lib/auth-tokens";

export const ticketsApi = createApi({
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
    }),
    getDepartmentTickets: builder.query({
      query: () => "api/tickets/department",
    }),
  }),
});

export const { useGetDepartmentTicketsQuery, useGetMyTicketsQuery } = ticketsApi;
