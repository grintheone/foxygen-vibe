import { BrowserRouter, Navigate, Route, Routes } from "react-router";
import { GuestRoute, ProtectedRoute } from "../../features/auth";
import { DashboardPage } from "../../pages/dashboard";
import { SignInPage } from "../../pages/sign-in";
import { routePaths } from "../../shared/config/routes";

export function AppRouter() {
  return (
    <BrowserRouter>
      <Routes>
        <Route element={<GuestRoute />}>
          <Route path={routePaths.signIn} element={<SignInPage />} />
        </Route>
        <Route element={<ProtectedRoute />}>
          <Route path={routePaths.dashboard} element={<DashboardPage />} />
        </Route>
        <Route
          path="*"
          element={<Navigate replace to={routePaths.signIn} />}
        />
      </Routes>
    </BrowserRouter>
  );
}
