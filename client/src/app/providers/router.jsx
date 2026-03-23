import { lazy, Suspense } from "react";
import { BrowserRouter, Navigate, Route, Routes } from "react-router";
import { GuestRoute, ProtectedRoute } from "../../features/auth";
import { routePaths } from "../../shared/config/routes";
import { RouteLoader } from "../../shared/ui/route-loader";

const ClientAgreementsPage = lazy(() =>
  import("../../pages/client-agreements").then((module) => ({
    default: module.ClientAgreementsPage,
  })),
);
const ClientArchivePage = lazy(() =>
  import("../../pages/client-archive").then((module) => ({
    default: module.ClientArchivePage,
  })),
);
const ClientContactsPage = lazy(() =>
  import("../../pages/client-contacts").then((module) => ({
    default: module.ClientContactsPage,
  })),
);
const ClientPage = lazy(() =>
  import("../../pages/client").then((module) => ({
    default: module.ClientPage,
  })),
);
const ChangePasswordPage = lazy(() =>
  import("../../pages/change-password").then((module) => ({
    default: module.ChangePasswordPage,
  })),
);
const DashboardPage = lazy(() =>
  import("../../pages/dashboard").then((module) => ({
    default: module.DashboardPage,
  })),
);
const DeviceArchivePage = lazy(() =>
  import("../../pages/device-archive").then((module) => ({
    default: module.DeviceArchivePage,
  })),
);
const DevicePage = lazy(() =>
  import("../../pages/device").then((module) => ({
    default: module.DevicePage,
  })),
);
const EditorPage = lazy(() =>
  import("../../pages/editor").then((module) => ({
    default: module.EditorPage,
  })),
);
const ProfilePage = lazy(() =>
  import("../../pages/profile").then((module) => ({
    default: module.ProfilePage,
  })),
);
const ProfileArchivePage = lazy(() =>
  import("../../pages/profile-archive").then((module) => ({
    default: module.ProfileArchivePage,
  })),
);
const SignInPage = lazy(() =>
  import("../../pages/sign-in").then((module) => ({
    default: module.SignInPage,
  })),
);
const TicketPage = lazy(() =>
  import("../../pages/ticket").then((module) => ({
    default: module.TicketPage,
  })),
);

function withRouteLoader(Component, message) {
  return (
    <Suspense fallback={<RouteLoader message={message} />}>
      <Component />
    </Suspense>
  );
}

export function AppRouter() {
  return (
    <BrowserRouter>
      <Routes>
        <Route element={<GuestRoute />}>
          <Route
            path={routePaths.signIn}
            element={withRouteLoader(SignInPage, "Подготавливаем вход...")}
          />
        </Route>
        <Route element={<ProtectedRoute />}>
          <Route
            path={routePaths.dashboard}
            element={withRouteLoader(DashboardPage, "Загружаем панель...")}
          />
          <Route
            path={routePaths.editor}
            element={withRouteLoader(EditorPage, "Открываем редактор...")}
          />
          <Route
            path={routePaths.profile}
            element={withRouteLoader(ProfilePage, "Открываем профиль...")}
          />
          <Route
            path={routePaths.changePassword}
            element={withRouteLoader(ChangePasswordPage, "Открываем смену пароля...")}
          />
          <Route
            path={routePaths.profileArchivePattern}
            element={withRouteLoader(ProfileArchivePage, "Загружаем архив профиля...")}
          />
          <Route
            path={routePaths.memberProfilePattern}
            element={withRouteLoader(ProfilePage, "Открываем профиль...")}
          />
          <Route
            path={routePaths.clientAgreementsPattern}
            element={withRouteLoader(ClientAgreementsPage, "Загружаем договоры клиента...")}
          />
          <Route
            path={routePaths.clientArchivePattern}
            element={withRouteLoader(ClientArchivePage, "Загружаем архив клиента...")}
          />
          <Route
            path={routePaths.clientContactsPattern}
            element={withRouteLoader(ClientContactsPage, "Загружаем контакты клиента...")}
          />
          <Route
            path={routePaths.clientPattern}
            element={withRouteLoader(ClientPage, "Загружаем карточку клиента...")}
          />
          <Route
            path={routePaths.deviceArchivePattern}
            element={withRouteLoader(DeviceArchivePage, "Загружаем архив устройства...")}
          />
          <Route
            path={routePaths.devicePattern}
            element={withRouteLoader(DevicePage, "Загружаем устройство...")}
          />
          <Route
            path={routePaths.ticketPattern}
            element={withRouteLoader(TicketPage, "Загружаем тикет...")}
          />
        </Route>
        <Route
          path="*"
          element={<Navigate replace to={routePaths.signIn} />}
        />
      </Routes>
    </BrowserRouter>
  );
}
