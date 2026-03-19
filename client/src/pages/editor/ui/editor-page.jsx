import { lazy, Suspense } from "react";
import { useNavigate, useSearchParams } from "react-router";
import { useAuth } from "../../../features/auth";
import { routePaths } from "../../../shared/config/routes";
import { PageShell } from "../../../shared/ui/page-shell";
import { DashboardButton, EditorEntityCard, EditorNoAccess, EditorNoticeCard, EditorPageHeader } from "./editor-shared";

const EditorClientsPage = lazy(() =>
  import("./editor-clients-page").then((module) => ({
    default: module.EditorClientsPage,
  })),
);
const EditorAgreementsPage = lazy(() =>
  import("./editor-agreements-page").then((module) => ({
    default: module.EditorAgreementsPage,
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
      <EditorNoticeCard message={message} />
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

  if (entity === "agreements") {
    return (
      <Suspense fallback={<EditorPageLoader message="Загружаем редактор договоров..." />}>
        <EditorAgreementsPage />
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
        <EditorPageHeader
          title="Выбор сущности"
          description="Выберите, что именно хотите редактировать. Сейчас доступны рабочие срезы для клиентов, договоров, контактов и устройств."
          action={<DashboardButton onClick={handleOpenDashboard} />}
        />

        <section className="grid gap-6 md:grid-cols-2 xl:grid-cols-3">
          <EditorEntityCard
            title="Клиенты"
            badge="Готово"
            description="Редактирование названия, адреса, региона и JSON-поля location."
            onClick={() => navigate(routePaths.editorClients())}
          />
          <EditorEntityCard
            title="Договоры"
            badge="Готово"
            description="Редактирование клиента, дистрибьютора, устройства, статуса, гарантии и дат договора."
            onClick={() => navigate(routePaths.editorAgreements())}
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
