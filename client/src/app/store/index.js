import { configureStore } from "@reduxjs/toolkit";
import { authReducer } from "../../features/auth";
import { ticketsApi } from "../../shared/api/tickets-api";

export const store = configureStore({
  reducer: {
    auth: authReducer,
    [ticketsApi.reducerPath]: ticketsApi.reducer,
  },
  middleware: (getDefaultMiddleware) =>
    getDefaultMiddleware().concat(ticketsApi.middleware),
});
