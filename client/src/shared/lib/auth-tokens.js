const accessTokenKey = "foxygen.access_token";
const refreshTokenKey = "foxygen.refresh_token";

export function getAccessToken() {
  return window.localStorage.getItem(accessTokenKey);
}

export function getRefreshToken() {
  return window.localStorage.getItem(refreshTokenKey);
}

export function storeTokens(payload) {
  window.localStorage.setItem(accessTokenKey, payload.access_token);
  window.localStorage.setItem(refreshTokenKey, payload.refresh_token);
}

export function clearStoredTokens() {
  window.localStorage.removeItem(accessTokenKey);
  window.localStorage.removeItem(refreshTokenKey);
}
