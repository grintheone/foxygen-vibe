import { createSlice } from "@reduxjs/toolkit";
import {
  fetchProfile,
  loginRequest,
  refreshSessionRequest,
} from "../../../shared/api/auth";
import {
  clearStoredTokens,
  getAccessToken,
  getRefreshToken,
  storeTokens,
} from "../../../shared/lib/auth-tokens";
import { editorApi } from "../../../shared/api/editor-api";
import { sessionCleared } from "../../../shared/lib/session-events";
import { ticketsApi } from "../../../shared/api/tickets-api";

const initialFeedback = {
  tone: "idle",
  message: "",
};

const initialState = {
  feedback: initialFeedback,
  isBootstrapping: true,
  isRefreshing: false,
  isSubmitting: false,
  session: null,
};

const authSlice = createSlice({
  name: "auth",
  initialState,
  reducers: {
    clearFeedback(state) {
      state.feedback = initialFeedback;
    },
    setBootstrapping(state, action) {
      state.isBootstrapping = action.payload;
    },
    setFeedback(state, action) {
      state.feedback = action.payload;
    },
    setRefreshing(state, action) {
      state.isRefreshing = action.payload;
    },
    setSession(state, action) {
      state.session = action.payload;
    },
    setSubmitting(state, action) {
      state.isSubmitting = action.payload;
    },
  },
  extraReducers: (builder) => {
    builder.addCase(sessionCleared, (state) => {
      state.feedback = initialFeedback;
      state.isBootstrapping = false;
      state.isRefreshing = false;
      state.isSubmitting = false;
      state.session = null;
    });
  },
});

const {
  clearFeedback,
  setBootstrapping,
  setFeedback,
  setRefreshing,
  setSession,
  setSubmitting,
} = authSlice.actions;

async function loadSession(dispatch, accessToken) {
  const data = await fetchProfile(accessToken);
  dispatch(setSession(data));

  return data;
}

export function restoreSession() {
  return async function restoreSessionThunk(dispatch) {
    const accessToken = getAccessToken();
    const refreshToken = getRefreshToken();

    if (!accessToken) {
      dispatch(setBootstrapping(false));
      return null;
    }

    try {
      await loadSession(dispatch, accessToken);
    } catch {
      if (!refreshToken) {
        clearStoredTokens();
        dispatch(sessionCleared());
      } else {
        try {
          await dispatch(rotateSessionWithToken(refreshToken, { silent: true }));
        } catch {
          clearStoredTokens();
          dispatch(sessionCleared());
        }
      }
    } finally {
      dispatch(setBootstrapping(false));
    }

    return null;
  };
}

export function login(credentials) {
  return async function loginThunk(dispatch) {
    dispatch(setSubmitting(true));
    dispatch(clearFeedback());

    try {
      const data = await loginRequest(credentials);
      storeTokens(data);
      await loadSession(dispatch, data.access_token);
      dispatch(
        setFeedback({
          tone: "success",
          message: `С возвращением, ${data.username}.`,
        }),
      );

      return data;
    } catch (error) {
      clearStoredTokens();
      dispatch(sessionCleared());
      dispatch(
        setFeedback({
          tone: "error",
          message: error.message,
        }),
      );
      throw error;
    } finally {
      dispatch(setSubmitting(false));
    }
  };
}

export function rotateSessionWithToken(refreshToken, options = {}) {
  return async function rotateSessionThunk(dispatch) {
    const { silent = false } = options;

    dispatch(setRefreshing(true));

    try {
      const data = await refreshSessionRequest(refreshToken);
      storeTokens(data);
      await loadSession(dispatch, data.access_token);

      if (!silent) {
        dispatch(
          setFeedback({
            tone: "success",
            message: "Сессия успешно обновлена.",
          }),
        );
      }

      return data;
    } catch (error) {
      clearStoredTokens();
      dispatch(sessionCleared());

      if (!silent) {
        dispatch(
          setFeedback({
            tone: "error",
            message: error.message,
          }),
        );
      }

      throw error;
    } finally {
      dispatch(setRefreshing(false));
    }
  };
}

export function rotateSession() {
  return async function rotateSessionThunk(dispatch) {
    const refreshToken = getRefreshToken();

    if (!refreshToken) {
      const error = new Error("Refresh token отсутствует.");

      dispatch(
        setFeedback({
          tone: "error",
          message: error.message,
        }),
      );

      throw error;
    }

    dispatch(clearFeedback());

    return dispatch(rotateSessionWithToken(refreshToken));
  };
}

export function signOut() {
  return function signOutThunk(dispatch) {
    clearStoredTokens();
    dispatch(sessionCleared());
    dispatch(editorApi.util.resetApiState());
    dispatch(ticketsApi.util.resetApiState());
    dispatch(
      setFeedback({
        tone: "success",
        message: "Вы вышли из системы.",
      }),
    );
  };
}

export function selectAuthState(state) {
  return state.auth;
}

export const authActions = {
  clearFeedback,
  setFeedback,
};

export const authReducer = authSlice.reducer;
