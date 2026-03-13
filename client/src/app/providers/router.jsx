import { BrowserRouter, Navigate, Route, Routes } from "react-router";
import { ClientAgreementsPage } from "../../pages/client-agreements";
import { GuestRoute, ProtectedRoute } from "../../features/auth";
import { ClientArchivePage } from "../../pages/client-archive";
import { ClientContactsPage } from "../../pages/client-contacts";
import { ClientPage } from "../../pages/client";
import { DashboardPage } from "../../pages/dashboard";
import { DeviceArchivePage } from "../../pages/device-archive";
import { DevicePage } from "../../pages/device";
import { ProfilePage } from "../../pages/profile";
import { ProfileArchivePage } from "../../pages/profile-archive";
import { SignInPage } from "../../pages/sign-in";
import { TicketPage } from "../../pages/ticket";
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
          <Route path={routePaths.profile} element={<ProfilePage />} />
          <Route path={routePaths.profileArchivePattern} element={<ProfileArchivePage />} />
          <Route path={routePaths.memberProfilePattern} element={<ProfilePage />} />
          <Route path={routePaths.clientAgreementsPattern} element={<ClientAgreementsPage />} />
          <Route path={routePaths.clientArchivePattern} element={<ClientArchivePage />} />
          <Route path={routePaths.clientContactsPattern} element={<ClientContactsPage />} />
          <Route path={routePaths.clientPattern} element={<ClientPage />} />
          <Route path={routePaths.deviceArchivePattern} element={<DeviceArchivePage />} />
          <Route path={routePaths.devicePattern} element={<DevicePage />} />
          <Route path={routePaths.ticketPattern} element={<TicketPage />} />
        </Route>
        <Route
          path="*"
          element={<Navigate replace to={routePaths.signIn} />}
        />
      </Routes>
    </BrowserRouter>
  );
}
