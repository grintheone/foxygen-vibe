import { configureStore } from "@reduxjs/toolkit";
import { setupListeners } from "@reduxjs/toolkit/query";
import { authReducer } from "../../features/auth";
import { editorApi } from "../../shared/api/editor-api";
import { ticketsApi } from "../../shared/api/tickets-api";
import { registerAuthDispatch } from "../../shared/lib/auth-dispatch";

export const store = configureStore({
  reducer: {
    auth: authReducer,
    [editorApi.reducerPath]: editorApi.reducer,
    [ticketsApi.reducerPath]: ticketsApi.reducer,
  },
  middleware: (getDefaultMiddleware) =>
    getDefaultMiddleware().concat(ticketsApi.middleware, editorApi.middleware),
});

setupListeners(store.dispatch);
registerAuthDispatch(store.dispatch);
