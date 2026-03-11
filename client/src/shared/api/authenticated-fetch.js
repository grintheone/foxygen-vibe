import { fetchBaseQuery } from "@reduxjs/toolkit/query/react";
import { refreshSessionRequest } from "./auth";
import {
  clearStoredTokens,
  getAccessToken,
  getRefreshToken,
  storeTokens,
} from "../lib/auth-tokens";
import { getAuthDispatch } from "../lib/auth-dispatch";
import { sessionCleared } from "../lib/session-events";

let refreshPromise = null;

async function parseResponse(response) {
  const text = await response.text();

  if (!text) {
    return null;
  }

  try {
    return JSON.parse(text);
  } catch {
    return text;
  }
}

function isUnauthorized(error) {
  return error?.status === 401 || error?.originalStatus === 401;
}

function resolveDispatch(dispatch) {
  return dispatch ?? getAuthDispatch() ?? null;
}

function clearSession(dispatch) {
  clearStoredTokens();
  resolveDispatch(dispatch)?.(sessionCleared());
}

async function refreshAccessToken(dispatch) {
  const refreshToken = getRefreshToken();

  if (!refreshToken) {
    clearSession(dispatch);
    return false;
  }

  if (!refreshPromise) {
    refreshPromise = refreshSessionRequest(refreshToken)
      .then((data) => {
        storeTokens(data);
        return data;
      })
      .finally(() => {
        refreshPromise = null;
      });
  }

  try {
    await refreshPromise;
    return true;
  } catch {
    clearSession(dispatch);
    return false;
  }
}

function withAccessToken(headersInit) {
  const headers = new Headers(headersInit);
  const accessToken = getAccessToken();

  if (accessToken && !headers.has("Authorization")) {
    headers.set("Authorization", `Bearer ${accessToken}`);
  }

  return headers;
}

export async function fetchWithAuth(input, init = {}, options = {}) {
  const dispatch = resolveDispatch(options.dispatch);
  const execute = () =>
    fetch(input, {
      ...init,
      headers: withAccessToken(init.headers),
    });

  let response = await execute();

  if (response.status !== 401) {
    return response;
  }

  const refreshed = await refreshAccessToken(dispatch);

  if (!refreshed) {
    return response;
  }

  response = await execute();

  if (response.status === 401) {
    clearSession(dispatch);
  }

  return response;
}

const rawBaseQuery = fetchBaseQuery({
  baseUrl: "/",
  prepareHeaders: (headers) => withAccessToken(headers),
  responseHandler: parseResponse,
});

export async function baseQueryWithAuth(args, api, extraOptions) {
  let result = await rawBaseQuery(args, api, extraOptions);

  if (!isUnauthorized(result.error)) {
    return result;
  }

  const refreshed = await refreshAccessToken(api.dispatch);

  if (!refreshed) {
    return result;
  }

  result = await rawBaseQuery(args, api, extraOptions);

  if (isUnauthorized(result.error)) {
    clearSession(api.dispatch);
  }

  return result;
}
