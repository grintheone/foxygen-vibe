import { useEffect, useRef } from "react";
import { useAuth } from "../model/use-auth";

export function AuthBootstrap({ children }) {
  const hasBootstrapped = useRef(false);
  const { restoreSession } = useAuth();

  useEffect(() => {
    if (hasBootstrapped.current) {
      return;
    }

    hasBootstrapped.current = true;
    restoreSession();
  }, [restoreSession]);

  return children;
}
