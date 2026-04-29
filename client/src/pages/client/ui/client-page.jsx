import { useState } from "react";
import { useNavigate, useParams } from "react-router";
import { TicketCardWithExecutor } from "../../dashboard/ui/ticket-card-with-executor";
import {
  useAddCommentMutation,
  isMissingCommentReferenceError,
  useGetClientAgreementsQuery,
  useGetClientByIdQuery,
  useGetClientContactsQuery,
  useGetCommentsQuery,
  useGetClientTicketsQuery,
} from "../../../shared/api/tickets-api";
import { TicketContactCard } from "../../ticket/ui/components/ticket-contact-card";
import { TicketDeviceCard } from "../../ticket/ui/components/ticket-device-section";
import { routePaths } from "../../../shared/config/routes";
import { PageShell } from "../../../shared/ui/page-shell";
import { UserAvatar } from "../../../shared/ui/user-avatar";

function resolveLocationPoint(location) {
  const items = Array.isArray(location) ? location : location ? [location] : [];

  for (const item of items) {
    const lat = Number(item?.lat ?? item?.latitude);
    const lng = Number(item?.lng ?? item?.lon ?? item?.longitude);

    if (Number.isFinite(lat) && Number.isFinite(lng)) {
      return { lat, lng };
    }
  }

  return null;
}

function buildMapEmbedUrl(point) {
  if (!point) {
    return "";
  }

  const delta = 0.01;
  const params = new URLSearchParams({
    bbox: [point.lng - delta, point.lat - delta, point.lng + delta, point.lat + delta].join(","),
    layer: "mapnik",
    marker: `${point.lat},${point.lng}`,
  });

  return `https://www.openstreetmap.org/export/embed.html?${params.toString()}`;
}

function normalizePhoneHref(phone) {
  const value = (phone || "").trim();
  if (!value) {
    return "";
  }

  const normalized = value.replace(/[^\d+]/g, "");
  return normalized ? `tel:${normalized}` : "";
}

function normalizeEmailHref(email) {
  const value = (email || "").trim();
  return value ? `mailto:${value}` : "";
}

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

function BackButton({ onClick }) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-label="Назад"
      className="inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl bg-[#2F3545] text-[#94A3B8] transition hover:bg-[#394055] sm:h-12 sm:w-12 lg:h-14 lg:w-14"
    >
      <svg
        xmlns="http://www.w3.org/2000/svg"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.8"
        strokeLinecap="round"
        strokeLinejoin="round"
        className="h-5 w-5 sm:h-6 sm:w-6 lg:h-7 lg:w-7"
        aria-hidden="true"
      >
        <path d="M15 18l-6-6 6-6" />
      </svg>
    </button>
  );
}

function ClientHeader({ onBack }) {
  return (
    <header className="bg-transparent px-1 pt-2">
      <div className="grid grid-cols-[auto_1fr_auto] items-center gap-4 sm:gap-6 lg:gap-8">
        <BackButton onClick={onBack} />
        <p className="justify-self-center text-center text-sm font-semibold tracking-[0.18em] text-[#94A3B8] sm:text-base lg:text-lg xl:text-xl">
          Клиент
        </p>
        <div className="h-11 w-11 shrink-0 sm:h-12 sm:w-12 lg:h-14 lg:w-14" aria-hidden="true" />
      </div>
    </header>
  );
}

function ClientInfoSection({ title }) {
  return (
    <section className="px-1">
      <div className="min-w-0">
        <h1 className="text-[24px] font-semibold leading-tight tracking-tight text-white sm:text-[28px] lg:text-[32px] xl:text-[36px]">
          {title}
        </h1>
      </div>
    </section>
  );
}

function ClientAddressSection({ address, mapUrl }) {
  return (
    <section className="space-y-4 px-1">
      <h2 className="text-[16px] font-semibold tracking-tight text-[#BCC2CA] sm:text-[18px] lg:text-[20px]">Адрес</h2>

      <div className="overflow-hidden rounded-lg border border-slate-400/20 bg-[#2f3748] shadow-xl shadow-black/20">
        {mapUrl ? (
          <div className="h-64 border-b border-slate-400/10 bg-slate-100 sm:h-72">
            <iframe
              title="Карта адреса клиента"
              src={mapUrl}
              loading="lazy"
              className="h-full w-full border-0"
              referrerPolicy="no-referrer-when-downgrade"
            />
          </div>
        ) : null}

        <div className={`${mapUrl ? "bg-[#3f485a]" : ""} px-4 py-4 sm:px-5 sm:py-5`}>
          <p className="text-[16px] font-semibold leading-7 tracking-tight text-slate-50 sm:text-[18px]">
            {address || "Адрес не указан"}
          </p>
        </div>
      </div>
    </section>
  );
}

function ClientLatestTicketsSection({ clientId, tickets, isError, isLoading, onOpenTicket, onOpenArchive }) {
  return (
    <section className="space-y-4 px-1">
      <h2 className="text-[16px] font-semibold tracking-tight text-[#BCC2CA] sm:text-[18px] lg:text-[20px]">
        Последние выезды
      </h2>

      {isLoading ? (
        <div className="app-subtle-notice">
          <p className="text-sm text-slate-300">Загрузка последних выездов...</p>
        </div>
      ) : null}

      {isError ? (
        <div className="rounded-3xl border border-rose-300/30 bg-rose-500/10 p-6">
          <p className="text-sm text-rose-100">Не удалось загрузить последние выезды.</p>
        </div>
      ) : null}

      {!isLoading && !isError && tickets.length > 0 ? (
        <div className="grid gap-2">
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
        <div className="app-subtle-notice">
          <p className="text-sm text-slate-300">У этого клиента пока нет закрытых выездов.</p>
        </div>
      ) : null}

      <button
        type="button"
        onClick={() => onOpenArchive(clientId)}
        className="inline-flex items-center gap-3 rounded-2xl px-2 py-1 text-lg font-semibold text-[#9B7BFF] transition hover:text-[#B49CFF]"
      >
        <span>Все выезды</span>
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
          <path d="M5 12h14" />
          <path d="m13 6 6 6-6 6" />
        </svg>
      </button>
    </section>
  );
}

function ClientContactsSection({ clientId, contacts, isError, isLoading, onOpenContacts }) {
  const visibleContacts = contacts.slice(0, 2);
  const shouldShowAllLink = contacts.length > 2;

  return (
    <section className="space-y-4 px-1">
      <h2 className="text-[16px] font-semibold tracking-tight text-[#BCC2CA] sm:text-[18px] lg:text-[20px]">Контакты</h2>

      {isLoading ? (
        <div className="app-subtle-notice">
          <p className="text-sm text-slate-300">Загрузка контактов...</p>
        </div>
      ) : null}

      {isError ? (
        <div className="rounded-3xl border border-rose-300/30 bg-rose-500/10 p-6">
          <p className="text-sm text-rose-100">Не удалось загрузить контакты.</p>
        </div>
      ) : null}

      {!isLoading && !isError && visibleContacts.length > 0 ? (
        <div className="grid gap-4">
          {visibleContacts.map((contact) => (
            <TicketContactCard
              key={contact.id}
              contactName={contact.name}
              contactPosition={contact.position}
              phoneHref={normalizePhoneHref(contact.phone)}
              emailHref={normalizeEmailHref(contact.email)}
            />
          ))}
        </div>
      ) : null}

      {!isLoading && !isError && visibleContacts.length === 0 ? (
        <div className="app-subtle-notice">
          <p className="text-sm text-slate-300">У этого клиента пока нет контактов.</p>
        </div>
      ) : null}

      {shouldShowAllLink ? (
        <button
          type="button"
          onClick={() => onOpenContacts(clientId)}
          className="inline-flex items-center gap-3 rounded-2xl px-2 py-1 text-lg font-semibold text-[#9B7BFF] transition hover:text-[#B49CFF]"
        >
          <span>Все контакты</span>
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
            <path d="M5 12h14" />
            <path d="m13 6 6 6-6 6" />
          </svg>
        </button>
      ) : null}
    </section>
  );
}

function ClientAgreementsSection({ agreements, clientId, isError, isLoading, onOpenAgreementList, onOpenDevice }) {
  const visibleAgreements = agreements.slice(0, 2);
  const shouldShowAllLink = agreements.length > 2;

  return (
    <section className="space-y-4 px-1">
      <h2 className="text-[16px] font-semibold tracking-tight text-[#BCC2CA] sm:text-[18px] lg:text-[20px]">
        Оборудование
      </h2>

      {isLoading ? (
        <div className="app-subtle-notice">
          <p className="text-sm text-slate-300">Загрузка оборудования...</p>
        </div>
      ) : null}

      {isError ? (
        <div className="rounded-3xl border border-rose-300/30 bg-rose-500/10 p-6">
          <p className="text-sm text-rose-100">Не удалось загрузить оборудование.</p>
        </div>
      ) : null}

      {!isLoading && !isError && visibleAgreements.length > 0 ? (
        <div className="grid gap-4">
          {visibleAgreements.map((agreement) => (
            <TicketDeviceCard
              key={agreement.id}
              deviceName={agreement.deviceName}
              serialNumber={agreement.deviceSerialNumber}
              disabled={!agreement.device}
              onOpenDevice={() => onOpenDevice(agreement.device)}
            />
          ))}
        </div>
      ) : null}

      {!isLoading && !isError && visibleAgreements.length === 0 ? (
        <div className="app-subtle-notice">
          <p className="text-sm text-slate-300">У этого клиента пока нет оборудования по договорам.</p>
        </div>
      ) : null}

      {shouldShowAllLink ? (
        <button
          type="button"
          onClick={() => onOpenAgreementList(clientId)}
          className="inline-flex items-center gap-3 rounded-2xl px-2 py-1 text-lg font-semibold text-[#9B7BFF] transition hover:text-[#B49CFF]"
        >
          <span>Все оборудование</span>
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
            <path d="M5 12h14" />
            <path d="m13 6 6 6-6 6" />
          </svg>
        </button>
      ) : null}
    </section>
  );
}

function ClientCommentsSection({
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
      <h2 className="text-[16px] font-semibold tracking-tight text-[#BCC2CA] sm:text-[18px] lg:text-[20px]">
        Комментарии
      </h2>

      {isLoading ? (
        <div className="app-subtle-notice">
          <p className="text-sm text-slate-300">Загрузка комментариев...</p>
        </div>
      ) : null}

      {isError ? (
        <div className="rounded-3xl border border-rose-300/30 bg-rose-500/10 p-6">
          <p className="text-sm text-rose-100">Не удалось загрузить комментарии.</p>
        </div>
      ) : null}

      {!isLoading && !isError && comments.length === 0 ? (
        <div className="app-subtle-notice">
          <p className="text-sm text-slate-300">Пока нет комментариев.</p>
        </div>
      ) : null}

      {!isLoading && !isError && comments.length > 0 ? (
        <div className="grid gap-3">
          {comments.map((comment) => (
            <article
              key={comment.id}
              className="overflow-hidden rounded-lg border border-slate-400/20 bg-[#2f3748] shadow-xl shadow-black/20"
            >
              <div className="px-4 py-4">
                <p className="text-[16px] leading-7 text-slate-100 sm:text-[18px]">{comment.text || "—"}</p>
              </div>

              <div className="border-t border-slate-400/10 bg-[#3f485a] px-4 py-3">
                <div className="flex items-end justify-between gap-4">
                  <div className="flex items-center gap-3">
                    <UserAvatar
                      avatarUrl={comment.avatarUrl}
                      userId={comment.author_id}
                      name={comment.authorName}
                      className="h-10 w-10"
                      iconClassName="h-5 w-5"
                    />
                    <div>
                      <p className="text-[16px] font-semibold text-slate-100">{comment.authorName || "Не указано"}</p>
                      <p className="text-sm text-slate-200/80">{comment.department || "Отдел не указан"}</p>
                    </div>
                  </div>
                  <p className="shrink-0 text-sm text-slate-200/80">{formatCommentDate(comment.created_at)}</p>
                </div>
              </div>
            </article>
          ))}
        </div>
      ) : null}

      <form onSubmit={onSubmit} className="flex items-center gap-3">
        <textarea
          value={commentText}
          onChange={(event) => onChangeText(event.target.value)}
          placeholder="Добавить комментарий"
          rows={1}
          className="h-[42px] min-h-[42px] flex-1 resize-none rounded-full bg-[#3f485a] px-4 py-[9px] text-[16px] leading-6 text-slate-100 outline-none transition placeholder:text-slate-300/70 focus:ring-1 focus:ring-slate-300/35 sm:h-12 sm:min-h-12 sm:px-5 sm:py-3 lg:h-14 lg:min-h-14"
        />
        <button
          type="submit"
          disabled={isSubmitting || !commentText.trim()}
          className="inline-flex h-[42px] w-[42px] shrink-0 items-center justify-center rounded-full bg-[#3f485a] text-slate-100 transition hover:bg-[#4a5468] disabled:cursor-not-allowed disabled:opacity-60 sm:h-12 sm:w-12 lg:h-14 lg:w-14"
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
            className="h-5 w-5 sm:h-6 sm:w-6"
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

export function ClientPage() {
  const navigate = useNavigate();
  const { clientId } = useParams();
  const [commentText, setCommentText] = useState("");
  const [commentError, setCommentError] = useState("");
  const {
    data: client,
    isError,
    isFetching,
    isLoading,
  } = useGetClientByIdQuery(clientId, {
    skip: !clientId,
  });
  const {
    data: closedTickets = [],
    isError: isClosedTicketsError,
    isFetching: isClosedTicketsFetching,
    isLoading: isClosedTicketsLoading,
  } = useGetClientTicketsQuery(
    {
      clientId,
      limit: 2,
      status: "closed",
    },
    {
      skip: !clientId,
    },
  );
  const {
    data: comments = [],
    error: commentsError,
    isError: isCommentsError,
    isFetching: isCommentsFetching,
    isLoading: isCommentsLoading,
  } = useGetCommentsQuery(clientId, {
    skip: !clientId,
  });
  const [addComment, { isLoading: isAddingComment }] = useAddCommentMutation();
  const {
    data: contacts = [],
    isError: isContactsError,
    isFetching: isContactsFetching,
    isLoading: isContactsLoading,
  } = useGetClientContactsQuery(
    {
      clientId,
      limit: 3,
    },
    {
      skip: !clientId,
    },
  );
  const {
    data: agreements = [],
    isError: isAgreementsError,
    isFetching: isAgreementsFetching,
    isLoading: isAgreementsLoading,
  } = useGetClientAgreementsQuery(
    {
      clientId,
      limit: 3,
    },
    {
      skip: !clientId,
    },
  );

  const pageTitle = client?.title?.trim() || "Карточка клиента";
  const address = client?.address?.trim() || "";
  const point = resolveLocationPoint(client?.location);
  const mapUrl = buildMapEmbedUrl(point);
  const hasMissingCommentReference = isMissingCommentReferenceError(commentsError);
  const isCommentsSectionError = isCommentsError && !hasMissingCommentReference;

  async function handleSubmitComment(event) {
    event.preventDefault();

    const nextComment = commentText.trim();
    if (!clientId || !nextComment) {
      return;
    }

    setCommentError("");

    try {
      await addComment({
        referenceId: clientId,
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
      <section className="w-full space-y-6">
        <ClientHeader onBack={() => navigate(-1)} />
        <ClientInfoSection title={pageTitle} />

        {isLoading || isFetching ? (
          <div className="app-subtle-notice">
            <p className="text-sm text-slate-300">Загрузка клиента...</p>
          </div>
        ) : null}

        {isError ? (
          <div className="rounded-3xl border border-rose-300/30 bg-rose-500/10 p-6">
            <p className="text-sm text-rose-100">Не удалось загрузить клиента.</p>
          </div>
        ) : null}

        {!isLoading && !isFetching && !isError && client ? (
          <>
            <ClientAddressSection address={address} mapUrl={mapUrl} />
            <ClientLatestTicketsSection
              clientId={clientId}
              tickets={closedTickets}
              isError={isClosedTicketsError}
              isLoading={isClosedTicketsLoading || isClosedTicketsFetching}
              onOpenTicket={(ticketId) => navigate(routePaths.ticketById(ticketId))}
              onOpenArchive={(targetClientId) => navigate(routePaths.clientArchiveById(targetClientId))}
            />
            <ClientContactsSection
              clientId={clientId}
              contacts={contacts}
              isError={isContactsError}
              isLoading={isContactsLoading || isContactsFetching}
              onOpenContacts={(targetClientId) => navigate(routePaths.clientContactsById(targetClientId))}
            />
            <ClientAgreementsSection
              agreements={agreements}
              clientId={clientId}
              isError={isAgreementsError}
              isLoading={isAgreementsLoading || isAgreementsFetching}
              onOpenAgreementList={(targetClientId) => navigate(routePaths.clientAgreementsById(targetClientId))}
              onOpenDevice={(deviceId) => {
                if (!deviceId) {
                  return;
                }

                navigate(routePaths.deviceById(deviceId));
              }}
            />
            <ClientCommentsSection
              comments={comments}
              commentText={commentText}
              errorMessage={commentError}
              isError={isCommentsSectionError}
              isLoading={isCommentsLoading || isCommentsFetching}
              isSubmitting={isAddingComment}
              onChangeText={setCommentText}
              onSubmit={handleSubmitComment}
            />
          </>
        ) : null}
      </section>
    </PageShell>
  );
}
