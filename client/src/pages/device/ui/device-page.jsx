import { useState } from "react";
import { useNavigate, useParams } from "react-router";
import { useAuth } from "../../../features/auth";
import { TicketCardWithExecutor } from "../../dashboard/ui/ticket-card-with-executor";
import {
  useAddCommentMutation,
  useGetCommentsQuery,
  useGetDeviceAgreementsQuery,
  useGetDeviceByIdQuery,
  useGetDeviceTicketsQuery,
} from "../../../shared/api/tickets-api";
import { routePaths } from "../../../shared/config/routes";
import { BottomPageAction } from "../../../shared/ui/bottom-page-action";
import { PageShell } from "../../../shared/ui/page-shell";
import { SlideOverSheet } from "../../../shared/ui/slide-over-sheet";

function formatCommentDate(value) {
  if (!value) {
    return "";
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "";
  }

  const day = String(date.getDate()).padStart(2, "0");
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const year = date.getFullYear();
  const hours = String(date.getHours()).padStart(2, "0");
  const minutes = String(date.getMinutes()).padStart(2, "0");

  return `${day}.${month}.${year} ${hours}:${minutes}`;
}

function formatShortDate(value) {
  if (!value) {
    return "";
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "";
  }

  const day = String(date.getDate()).padStart(2, "0");
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const year = date.getFullYear();

  return `${day}.${month}.${year}`;
}

function formatAgreementRange(agreement) {
  if (!agreement) {
    return "Срок не указан";
  }

  const assignedAt = formatShortDate(agreement.assigned_at);
  const finishedAt = formatShortDate(agreement.finished_at);

  if (!assignedAt && !finishedAt) {
    return "Срок не указан";
  }

  return `с ${assignedAt || "—"} до ${finishedAt || "—"}`;
}

function formatPropertyValue(value) {
  if (typeof value === "boolean") {
    return value ? "Да" : "Нет";
  }

  if (typeof value === "number") {
    return String(value);
  }

  if (typeof value === "string") {
    const normalized = value.trim();
    return normalized || "Не указано";
  }

  if (Array.isArray(value)) {
    return value
      .map((item) => formatPropertyValue(item))
      .filter((item) => item && item !== "Не указано")
      .join(", ") || "Не указано";
  }

  if (value && typeof value === "object") {
    return JSON.stringify(value);
  }

  return "Не указано";
}

function buildPropertyEntries(properties) {
  if (!properties || typeof properties !== "object" || Array.isArray(properties)) {
    return [];
  }

  return Object.entries(properties)
    .filter(([, value]) => value !== null && value !== undefined && value !== "")
    .map(([key, value]) => ({
      label: key,
      value: formatPropertyValue(value),
    }));
}

function BackButton({ onClick }) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-label="Назад"
      className="inline-flex h-11 w-11 items-center justify-center rounded-2xl bg-[#6A3BF2] text-white transition hover:bg-[#7C52F5]"
    >
      <svg
        xmlns="http://www.w3.org/2000/svg"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="2.5"
        strokeLinecap="round"
        strokeLinejoin="round"
        className="h-5 w-5"
        aria-hidden="true"
      >
        <path d="M15 18l-6-6 6-6" />
      </svg>
    </button>
  );
}

function DeviceHeader({ serialNumber, title, onBack }) {
  return (
    <header className="rounded-3xl border border-white/10 bg-slate-950/35 p-6 shadow-xl shadow-black/20 backdrop-blur">
      <BackButton onClick={onBack} />
      <div className="mt-5 text-left">
        <p className="text-xs font-semibold uppercase tracking-[0.3em] text-slate-400">Оборудование</p>
        <h1 className="mt-3 text-3xl font-bold tracking-tight text-slate-50 sm:text-4xl">
          {title || "Устройство"}
        </h1>
        <p className="mt-3 text-base text-slate-300 sm:text-lg">
          С/Н: <span className="font-semibold text-slate-100">{serialNumber || "Не указано"}</span>
        </p>
      </div>
    </header>
  );
}

function DeviceStatCard({ label, value }) {
  return (
    <article className="rounded-3xl border border-white/10 bg-slate-950/35 p-5 shadow-xl shadow-black/20 backdrop-blur">
      <p className="text-xs font-semibold uppercase tracking-[0.24em] text-slate-400">{label}</p>
      <p className="mt-3 text-xl font-semibold tracking-tight text-slate-50 sm:text-2xl">{value || "Не указано"}</p>
    </article>
  );
}

function DeviceOverviewSection({ device, propertyEntries }) {
  const agreementLabel = device?.agreementNumber ? `Договор #${device.agreementNumber}` : "Договор не найден";
  const agreementMeta = [
    device?.agreementType?.trim() || "",
    device?.agreement ? (device.isActiveAgreement ? "Активный" : "Неактивный") : "",
    device?.agreement ? (device.onWarranty ? "Гарантия" : "Без гарантии") : "",
  ]
    .filter(Boolean)
    .join(" • ");

  return (
    <section className="space-y-4">
      <h2 className="text-2xl font-bold tracking-tight text-slate-50 sm:text-3xl">Сведения</h2>

      <div className="grid gap-3 sm:grid-cols-2">
        <DeviceStatCard label="Серийный номер" value={device?.serialNumber || "Не указано"} />
        <DeviceStatCard label="LIS" value={device?.connectedToLis ? "Подключено" : "Не подключено"} />
        <DeviceStatCard label="Статус" value={device?.isUsed ? "В эксплуатации" : "Не используется"} />
        <DeviceStatCard label="Договор" value={agreementMeta ? `${agreementLabel} • ${agreementMeta}` : agreementLabel} />
      </div>

      <div className="rounded-3xl border border-white/10 bg-slate-950/35 p-6 shadow-xl shadow-black/20 backdrop-blur">
        <h3 className="text-lg font-semibold tracking-tight text-slate-100 sm:text-2xl">Параметры</h3>

        {propertyEntries.length > 0 ? (
          <div className="mt-5 grid gap-3">
            {propertyEntries.map((entry) => (
              <div
                key={entry.label}
                className="flex flex-col gap-1 rounded-2xl border border-white/10 bg-white/5 px-4 py-3 sm:flex-row sm:items-center sm:justify-between sm:gap-4"
              >
                <p className="text-sm font-semibold uppercase tracking-[0.18em] text-slate-400">{entry.label}</p>
                <p className="text-base text-slate-100 sm:text-right">{entry.value}</p>
              </div>
            ))}
          </div>
        ) : (
          <p className="mt-4 text-sm text-slate-300">Дополнительные параметры не указаны.</p>
        )}
      </div>
    </section>
  );
}

function DeviceServiceSection({
  agreement,
  isError,
  isLoading,
  onOpenClient,
  onOpenExpiredAgreements,
}) {
  return (
    <section className="space-y-4">
      <h2 className="text-2xl font-bold tracking-tight text-slate-50 sm:text-3xl">Сервисные услуги</h2>

      {isLoading ? (
        <div className="rounded-3xl border border-white/10 bg-white/5 p-6">
          <p className="text-sm text-slate-300">Загрузка сервисных услуг...</p>
        </div>
      ) : null}

      {isError ? (
        <div className="rounded-3xl border border-rose-300/30 bg-rose-500/10 p-6">
          <p className="text-sm text-rose-100">Не удалось загрузить сервисные услуги.</p>
        </div>
      ) : null}

      {!isLoading && !isError && agreement?.client ? (
        <>
          <button
            type="button"
            onClick={() => onOpenClient(agreement.client)}
            className="w-full rounded-3xl border border-white/10 bg-slate-950/35 p-6 text-left shadow-xl shadow-black/20 backdrop-blur transition hover:border-white/20 hover:bg-slate-950/45"
          >
            <p className="text-xs font-semibold uppercase tracking-[0.24em] text-slate-400">Активная услуга</p>
            <p className="mt-3 text-2xl font-semibold tracking-tight text-slate-50">
              {agreement.clientName || "Не указано"}
            </p>
            <p className="mt-2 text-lg text-slate-400">{agreement.clientAddress || "Адрес не указан"}</p>
            <p className="mt-6 text-base font-medium text-slate-200">{formatAgreementRange(agreement)}</p>
          </button>

          <button
            type="button"
            onClick={() => onOpenExpiredAgreements(agreement.client)}
            className="inline-flex items-center gap-3 rounded-2xl px-2 py-1 text-lg font-semibold text-[#8B5CFF] transition hover:text-[#A27BFF]"
          >
            <span>Истекшие сервисные услуги</span>
            <svg
              xmlns="http://www.w3.org/2000/svg"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2.2"
              strokeLinecap="round"
              strokeLinejoin="round"
              className="h-5 w-5"
              aria-hidden="true"
            >
              <path d="M9 6l6 6-6 6" />
            </svg>
          </button>
        </>
      ) : null}

      {!isLoading && !isError && !agreement?.client ? (
        <div className="rounded-3xl border border-white/10 bg-white/5 p-6">
          <p className="text-sm text-slate-300">Для этого устройства не найдено активных сервисных услуг.</p>
        </div>
      ) : null}
    </section>
  );
}

function DeviceLatestTicketsSection({ isError, isLoading, onOpenTicket, tickets }) {
  return (
    <section className="space-y-4">
      <h2 className="text-2xl font-bold tracking-tight text-slate-50 sm:text-3xl">Последние выезды</h2>

      {isLoading ? (
        <div className="rounded-3xl border border-white/10 bg-white/5 p-6">
          <p className="text-sm text-slate-300">Загрузка последних выездов...</p>
        </div>
      ) : null}

      {isError ? (
        <div className="rounded-3xl border border-rose-300/30 bg-rose-500/10 p-6">
          <p className="text-sm text-rose-100">Не удалось загрузить последние выезды.</p>
        </div>
      ) : null}

      {!isLoading && !isError && tickets.length > 0 ? (
        <div className="grid gap-3">
          {tickets.map((ticket) => (
            <TicketCardWithExecutor
              key={ticket.id}
              ticket={ticket}
              executor={{
                department: ticket.executorDepartment,
                name: ticket.executorName,
              }}
              onOpenTicket={onOpenTicket}
            />
          ))}
        </div>
      ) : null}

      {!isLoading && !isError && tickets.length === 0 ? (
        <div className="rounded-3xl border border-white/10 bg-white/5 p-6">
          <p className="text-sm text-slate-300">У этого устройства пока нет закрытых выездов.</p>
        </div>
      ) : null}
    </section>
  );
}

function DeviceCommentsSection({
  comments,
  commentText,
  errorMessage,
  isError,
  isLoading,
  isSubmitting,
  onChangeText,
  onSubmit,
}) {
  return (
    <section className="space-y-4">
      <h2 className="text-2xl font-bold tracking-tight text-slate-50 sm:text-3xl">Комментарии</h2>

      {isLoading ? (
        <div className="rounded-3xl border border-white/10 bg-white/5 p-6">
          <p className="text-sm text-slate-300">Загрузка комментариев...</p>
        </div>
      ) : null}

      {isError ? (
        <div className="rounded-3xl border border-rose-300/30 bg-rose-500/10 p-6">
          <p className="text-sm text-rose-100">Не удалось загрузить комментарии.</p>
        </div>
      ) : null}

      {!isLoading && !isError && comments.length === 0 ? (
        <div className="rounded-3xl border border-white/10 bg-white/5 p-6">
          <p className="text-sm text-slate-300">Пока нет комментариев.</p>
        </div>
      ) : null}

      {!isLoading && !isError && comments.length > 0 ? (
        <div className="grid gap-3">
          {comments.map((comment) => (
            <article
              key={comment.id}
              className="rounded-3xl border border-white/10 bg-slate-950/35 p-6 shadow-xl shadow-black/20 backdrop-blur"
            >
              <p className="text-lg leading-8 text-slate-100 sm:text-2xl sm:leading-10">{comment.text || "—"}</p>

              <div className="mt-6 flex items-end justify-between gap-4">
                <div className="flex items-center gap-4">
                  <div className="h-10 w-10 rounded-full bg-slate-950 sm:h-12 sm:w-12" aria-hidden="true" />
                  <div>
                    <p className="text-lg font-semibold text-slate-100">{comment.authorName || "Не указано"}</p>
                    <p className="text-sm text-slate-400 sm:text-lg">{comment.department || "Отдел не указан"}</p>
                  </div>
                </div>
                <p className="shrink-0 text-sm text-slate-400 sm:text-lg">{formatCommentDate(comment.created_at)}</p>
              </div>
            </article>
          ))}
        </div>
      ) : null}

      <form
        onSubmit={onSubmit}
        className="flex items-end gap-3 rounded-[2rem] border border-white/10 bg-slate-950/35 p-3 shadow-xl shadow-black/20 backdrop-blur"
      >
        <textarea
          value={commentText}
          onChange={(event) => onChangeText(event.target.value)}
          placeholder="Добавить комментарий"
          rows={3}
          className="min-h-[7rem] flex-1 resize-none rounded-[1.6rem] border border-white/10 bg-white/5 px-5 py-4 text-lg text-slate-100 outline-none transition placeholder:text-slate-400 focus:border-white/25"
        />
        <button
          type="submit"
          disabled={isSubmitting || !commentText.trim()}
          className="inline-flex h-14 w-14 shrink-0 items-center justify-center rounded-full bg-[#6A3BF2] text-white transition hover:bg-[#7C52F5] disabled:cursor-not-allowed disabled:opacity-60"
          aria-label="Отправить комментарий"
        >
          <svg
            xmlns="http://www.w3.org/2000/svg"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2.4"
            strokeLinecap="round"
            strokeLinejoin="round"
            className="h-6 w-6"
            aria-hidden="true"
          >
            <path d="M3 11.5 20.5 4 13 21l-2.5-6.5L3 11.5Z" />
          </svg>
        </button>
      </form>

      {errorMessage ? <p className="text-sm text-rose-200">{errorMessage}</p> : null}
    </section>
  );
}

function DeviceCreateTicketSheet({ device, isOpen, onClose }) {
  const deviceTitle = device?.title?.trim() || "Устройство";
  const serialNumber = device?.serialNumber?.trim() || "Не указано";
  const clientName = device?.clientName?.trim() || "Клиент не указан";
  const clientAddress = device?.clientAddress?.trim() || "Адрес не указан";

  return (
    <SlideOverSheet
      isOpen={isOpen}
      onClose={onClose}
      closeLabel="Закрыть создание тикета"
      eyebrow="Новый тикет"
      title="Создание тикета на прибор"
    >
      <div className="mt-8 space-y-6">
        <div className="rounded-2xl border border-white/10 bg-white/5 p-5">
          <p className="text-sm text-slate-400">Прибор</p>
          <p className="mt-2 text-lg font-semibold text-slate-100">{deviceTitle}</p>
          <p className="mt-2 text-sm text-slate-300">С/Н: {serialNumber}</p>
        </div>

        <div className="rounded-2xl border border-white/10 bg-white/5 p-5">
          <p className="text-sm text-slate-400">Клиент</p>
          <p className="mt-2 text-lg font-semibold text-slate-100">{clientName}</p>
          <p className="mt-2 text-sm text-slate-300">{clientAddress}</p>
        </div>

        <div className="rounded-3xl border border-dashed border-white/15 bg-slate-900/40 p-6">
          <h3 className="text-xl font-semibold text-slate-100">Форма в следующем шаге</h3>
          <p className="mt-3 text-base leading-7 text-slate-300">
            Шторка и контекст прибора готовы. Дальше сюда можно добавить поля причины, описания и остальных данных для
            создания тикета.
          </p>
        </div>
      </div>
    </SlideOverSheet>
  );
}

export function DevicePage() {
  const navigate = useNavigate();
  const { deviceId } = useParams();
  const { session } = useAuth();
  const [commentText, setCommentText] = useState("");
  const [commentError, setCommentError] = useState("");
  const [isCreateTicketSheetOpen, setIsCreateTicketSheetOpen] = useState(false);
  const {
    data: device,
    isError,
    isFetching,
    isLoading,
  } = useGetDeviceByIdQuery(deviceId, {
    skip: !deviceId,
  });
  const {
    data: agreements = [],
    isError: isAgreementsError,
    isFetching: isAgreementsFetching,
    isLoading: isAgreementsLoading,
  } = useGetDeviceAgreementsQuery(
    {
      active: true,
      deviceId,
    },
    {
      skip: !deviceId,
    },
  );
  const {
    data: tickets = [],
    isError: isTicketsError,
    isFetching: isTicketsFetching,
    isLoading: isTicketsLoading,
  } = useGetDeviceTicketsQuery(
    {
      deviceId,
      limit: 2,
      status: "closed",
    },
    {
      skip: !deviceId,
    },
  );
  const {
    data: comments = [],
    isError: isCommentsError,
    isFetching: isCommentsFetching,
    isLoading: isCommentsLoading,
  } = useGetCommentsQuery(deviceId, {
    skip: !deviceId,
  });
  const [addComment, { isLoading: isAddingComment }] = useAddCommentMutation();

  const pageTitle = device?.title?.trim() || "Устройство";
  const serialNumber = device?.serialNumber?.trim() || "";
  const propertyEntries = buildPropertyEntries(device?.properties);
  const activeAgreement = agreements[0] || null;
  const canCreateTicket = session?.role === "admin" || session?.role === "coordinator";
  const hasCreateTicketWidget = canCreateTicket && !isLoading && !isFetching && !isError && Boolean(device);

  async function handleSubmitComment(event) {
    event.preventDefault();

    const nextComment = commentText.trim();
    if (!deviceId || !nextComment) {
      return;
    }

    setCommentError("");

    try {
      await addComment({
        referenceId: deviceId,
        text: nextComment,
      }).unwrap();
      setCommentText("");
    } catch (error) {
      if (typeof error?.data === "string") {
        setCommentError(error.data);
        return;
      }

      if (typeof error?.error === "string") {
        setCommentError(error.error);
        return;
      }

      setCommentError("Не удалось добавить комментарий.");
    }
  }

  return (
    <PageShell>
      <section
        className={`w-full space-y-6 transition ${hasCreateTicketWidget ? "pb-28" : ""} ${
          isCreateTicketSheetOpen ? "brightness-75" : ""
        }`}
      >
        <DeviceHeader
          title={pageTitle}
          serialNumber={serialNumber}
          onBack={() => navigate(-1)}
        />

        {isLoading || isFetching ? (
          <div className="rounded-3xl border border-white/10 bg-white/5 p-6">
            <p className="text-sm text-slate-300">Загрузка устройства...</p>
          </div>
        ) : null}

        {isError ? (
          <div className="rounded-3xl border border-rose-300/30 bg-rose-500/10 p-6">
            <p className="text-sm text-rose-100">Не удалось загрузить устройство.</p>
          </div>
        ) : null}

        {!isLoading && !isFetching && !isError && device ? (
          <>
            <DeviceServiceSection
              agreement={activeAgreement}
              isError={isAgreementsError}
              isLoading={isAgreementsLoading || isAgreementsFetching}
              onOpenClient={(clientIdValue) => navigate(routePaths.clientById(clientIdValue))}
              onOpenExpiredAgreements={(clientIdValue) => navigate(routePaths.clientAgreementsById(clientIdValue))}
            />
            <DeviceOverviewSection device={device} propertyEntries={propertyEntries} />
            <DeviceLatestTicketsSection
              tickets={tickets}
              isError={isTicketsError}
              isLoading={isTicketsLoading || isTicketsFetching}
              onOpenTicket={(ticketId) => navigate(routePaths.ticketById(ticketId))}
            />
            <DeviceCommentsSection
              comments={comments}
              commentText={commentText}
              errorMessage={commentError}
              isError={isCommentsError}
              isLoading={isCommentsLoading || isCommentsFetching}
              isSubmitting={isAddingComment}
              onChangeText={setCommentText}
              onSubmit={handleSubmitComment}
            />
          </>
        ) : null}
      </section>

      {hasCreateTicketWidget ? (
        <BottomPageAction onClick={() => setIsCreateTicketSheetOpen(true)}>
          <span>Создать тикет на прибор</span>
        </BottomPageAction>
      ) : null}

      <DeviceCreateTicketSheet
        device={device}
        isOpen={isCreateTicketSheetOpen}
        onClose={() => setIsCreateTicketSheetOpen(false)}
      />
    </PageShell>
  );
}
