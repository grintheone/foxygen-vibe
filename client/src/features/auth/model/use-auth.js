import { useDispatch, useSelector } from "react-redux";
import {
  authActions,
  login,
  restoreSession,
  rotateSession,
  selectAuthState,
  signOut,
} from "./auth-slice";

export function useAuth() {
  const dispatch = useDispatch();
  const authState = useSelector(selectAuthState);

  return {
    ...authState,
    clearFeedback() {
      dispatch(authActions.clearFeedback());
    },
    login(credentials) {
      return dispatch(login(credentials));
    },
    restoreSession() {
      return dispatch(restoreSession());
    },
    rotateSession() {
      return dispatch(rotateSession());
    },
    setFeedback(feedback) {
      dispatch(authActions.setFeedback(feedback));
    },
    signOut() {
      dispatch(signOut());
    },
  };
}
