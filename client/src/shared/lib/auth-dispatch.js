let authDispatch = null;

export function registerAuthDispatch(dispatch) {
  authDispatch = dispatch;
}

export function getAuthDispatch() {
  return authDispatch;
}
