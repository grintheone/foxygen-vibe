import { lazy, Suspense } from "react";
import { useNavigate, useSearchParams } from "react-router";
import { useAuth } from "../../../features/auth";
import { routePaths } from "../../../shared/config/routes";
import { PageShell } from "../../../shared/ui/page-shell";
import { DashboardButton, EditorEntityCard, EditorNoAccess } from "./editor-shared";

const EditorClientsPage = lazy(() =>
  import("./editor-clients-page").then((module) => ({
    default: module.EditorClientsPage,
  })),
);
const EditorContactsPage = lazy(() =>
  import("./editor-contacts-page").then((module) => ({
    default: module.EditorContactsPage,
  })),
);
const EditorDevicesPage = lazy(() =>
  import("./editor-devices-page").then((module) => ({
    default: module.EditorDevicesPage,
  })),
);

function EditorPageLoader({ message }) {
  return (
    <PageShell>
      <section className="w-full rounded-3xl border border-white/10 bg-white/5 p-6">
        <p className="text-sm text-slate-300">{message}</p>
      </section>
    </PageShell>
  );
}

export function EditorPage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const { session } = useAuth();
  const canOpenEditor = session?.role === "coordinator" || session?.role === "admin";
  const entity = searchParams.get("entity") || "";

  function handleBack() {
    navigate(-1);
  }

  function handleOpenDashboard() {
    navigate(routePaths.dashboard);
  }

  if (!canOpenEditor) {
    return <EditorNoAccess onBack={handleBack} />;
  }

  if (entity === "clients") {
    return (
      <Suspense fallback={<EditorPageLoader message="Загружаем редактор клиентов..." />}>
        <EditorClientsPage />
      </Suspense>
    );
  }

  if (entity === "contacts") {
    return (
      <Suspense fallback={<EditorPageLoader message="Загружаем редактор контактов..." />}>
        <EditorContactsPage />
      </Suspense>
    );
  }

  if (entity === "devices") {
    return (
      <Suspense fallback={<EditorPageLoader message="Загружаем редактор устройств..." />}>
        <EditorDevicesPage />
      </Suspense>
    );
  }

  return (
    <PageShell>
      <section className="w-full space-y-6">
        <header className="rounded-3xl border border-white/10 bg-slate-950/35 p-6 shadow-xl shadow-black/20 backdrop-blur">
          <div className="flex items-center justify-between gap-4">
            <div>
              <p className="text-xs font-semibold uppercase tracking-[0.3em] text-slate-400">Редактор</p>
              <h1 className="mt-3 text-3xl font-bold tracking-tight text-white sm:text-4xl">Выбор сущности</h1>
              <p className="mt-3 max-w-2xl text-base text-slate-300">
                Выберите, что именно хотите редактировать. Сейчас доступны рабочие срезы для клиентов, контактов и
                устройств.
              </p>
            </div>
            <DashboardButton onClick={handleOpenDashboard} />
          </div>
        </header>

        <section className="grid gap-6 md:grid-cols-2 xl:grid-cols-3">
          <EditorEntityCard
            title="Клиенты"
            badge="Готово"
            description="Редактирование названия, адреса, региона и JSON-поля location."
            onClick={() => navigate(routePaths.editorClients())}
          />
          <EditorEntityCard
            title="Контакты"
            badge="Готово"
            description="Редактирование имени, должности, телефона, email и привязки к клиенту."
            onClick={() => navigate(routePaths.editorContacts())}
          />
          <EditorEntityCard
            title="Устройства"
            badge="Готово"
            description="Редактирование классификатора, серийного номера, JSON-параметров и служебных флагов."
            onClick={() => navigate(routePaths.editorDevices())}
          />
        </section>
      </section>
    </PageShell>
  );
}
