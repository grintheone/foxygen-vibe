import { Navigate, Outlet } from "react-router";
import { routePaths } from "../../../shared/config/routes";
import { RouteLoader } from "../../../shared/ui/route-loader";
import { useAuth } from "../model/use-auth";

export function GuestRoute() {
  const { isBootstrapping, session } = useAuth();

  if (isBootstrapping) {
    return <RouteLoader message="Restoring your session..." />;
  }

  if (session) {
    return <Navigate replace to={routePaths.dashboard} />;
  }

  return <Outlet />;
}
