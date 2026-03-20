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
const EditorClassificatorsPage = lazy(() =>
  import("./editor-classificators-page").then((module) => ({
    default: module.EditorClassificatorsPage,
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
const EditorTicketsPage = lazy(() =>
  import("./editor-tickets-page").then((module) => ({
    default: module.EditorTicketsPage,
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

  if (entity === "classificators") {
    return (
      <Suspense fallback={<EditorPageLoader message="Загружаем редактор классификаторов..." />}>
        <EditorClassificatorsPage />
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

  if (entity === "tickets") {
    return (
      <Suspense fallback={<EditorPageLoader message="Загружаем редактор тикетов..." />}>
        <EditorTicketsPage />
      </Suspense>
    );
  }

  return (
    <PageShell>
      <section className="w-full space-y-6">
        <EditorPageHeader
          title="Выбор сущности"
          action={<DashboardButton onClick={handleOpenDashboard} />}
          textAlign="left"
        />

        <section className="grid gap-6 md:grid-cols-2 xl:grid-cols-3">
          <EditorEntityCard
            title="Клиенты"
            description="Когда нужно уточнить саму карточку клиента, здесь можно поправить название, адрес, регион и точку на карте."
            onClick={() => navigate(routePaths.editorClients())}
          />
          <EditorEntityCard
            title="Договоры"
            description="Если договор связан не с тем клиентом или оборудованием, здесь можно быстро обновить эти привязки, даты и служебные флаги."
            onClick={() => navigate(routePaths.editorAgreements())}
          />
          <EditorEntityCard
            title="Классификаторы"
            description="Для самого классификатора доступны его ключевые данные: название, производитель, тип исследования и регистрационные сведения."
            onClick={() => navigate(routePaths.editorClassificators())}
          />
          <EditorEntityCard
            title="Контакты"
            description="Когда меняются данные контактного лица, здесь можно поправить его имя, должность, контакты и связь с клиентом."
            onClick={() => navigate(routePaths.editorContacts())}
          />
          <EditorEntityCard
            title="Оборудование"
            description="Для единицы оборудования здесь редактируются ее основные признаки: классификатор, серийный номер и служебное состояние."
            onClick={() => navigate(routePaths.editorDevices())}
          />
          <EditorEntityCard
            title="Тикеты"
            description="Если нужно привести в порядок сам тикет, здесь можно обновить его статус, связи, сроки, описание работы и итог."
            onClick={() => navigate(routePaths.editorTickets())}
          />
        </section>
      </section>
    </PageShell>
  );
}
