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
import { routePaths } from "../../../shared/config/routes";
import { ContactCard } from "../../../shared/ui/contact-card";
import { NavigationCard } from "../../../shared/ui/navigation-card";
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

function ClientHeader({ title, onBack }) {
  return (
    <header className="grid grid-cols-[auto_minmax(0,1fr)_auto] items-center rounded-3xl border border-white/10 bg-slate-950/35 p-6 shadow-xl shadow-black/20 backdrop-blur">
      <div className="justify-self-start">
        <BackButton onClick={onBack} />
      </div>

      <div className="justify-self-center text-center">
        <p className="text-xs font-semibold uppercase tracking-[0.3em] text-slate-400">Клиент</p>
        <h1 className="mt-3 text-3xl font-bold tracking-tight sm:text-4xl">{title}</h1>
      </div>

      <div className="h-11 w-11" aria-hidden="true" />
    </header>
  );
}

function ClientAddressSection({ address, mapUrl }) {
  return (
    <section className="rounded-3xl border border-white/10 bg-slate-950/35 p-6 shadow-xl shadow-black/20 backdrop-blur">
      <h2 className="text-2xl font-bold tracking-tight text-slate-50 sm:text-3xl">Адрес</h2>

      <div className="mt-5 overflow-hidden rounded-lg border border-white/10 bg-slate-950/45">
        {mapUrl ? (
          <div className="h-64 border-b border-white/10 bg-slate-100 sm:h-72">
            <iframe
              title="Карта адреса клиента"
              src={mapUrl}
              loading="lazy"
              className="h-full w-full border-0"
              referrerPolicy="no-referrer-when-downgrade"
            />
          </div>
        ) : null}

        <div className="p-6 sm:p-8">
          <p className="text-2xl font-semibold tracking-tight text-slate-50 sm:text-3xl">
            {address || "Адрес не указан"}
          </p>
        </div>
      </div>
    </section>
  );
}

function ClientLatestTicketsSection({ clientId, tickets, isError, isLoading, onOpenTicket, onOpenArchive }) {
  return (
    <section className="space-y-4">
      <h2 className="text-2xl font-bold tracking-tight text-slate-50 sm:text-3xl">Последние выезды</h2>

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
    <section className="space-y-4">
      <h2 className="text-2xl font-bold tracking-tight text-slate-50 sm:text-3xl">Контакты</h2>

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
        <div className="grid gap-3">
          {visibleContacts.map((contact) => (
            <ContactCard
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
    <section className="space-y-4">
      <h2 className="text-2xl font-bold tracking-tight text-slate-50 sm:text-3xl">Оборудование</h2>

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
        <div className="grid gap-3">
          {visibleAgreements.map((agreement) => (
            <NavigationCard
              key={agreement.id}
              value={agreement.deviceName}
              subtitle={`С/Н: ${agreement.deviceSerialNumber || "Не указано"}`}
              disabled={!agreement.device}
              onClick={() => onOpenDevice(agreement.device)}
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
      <h2 className="text-2xl font-bold tracking-tight text-slate-50 sm:text-3xl">Комментарии</h2>

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
              className="rounded-3xl border border-white/10 bg-slate-950/35 p-6 shadow-xl shadow-black/20 backdrop-blur"
            >
              <p className="text-lg leading-8 text-slate-100 sm:text-2xl sm:leading-10">{comment.text || "—"}</p>

              <div className="mt-6 flex items-end justify-between gap-4">
                <div className="flex items-center gap-4">
                  <UserAvatar
                    avatarUrl={comment.avatarUrl}
                    userId={comment.author_id}
                    name={comment.authorName}
                    className="h-10 w-10 sm:h-12 sm:w-12"
                    iconClassName="h-5 w-5 sm:h-6 sm:w-6"
                  />
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
        className="flex items-end gap-3 rounded-lg border border-white/10 bg-slate-950/35 p-3 shadow-xl shadow-black/20 backdrop-blur"
      >
        <textarea
          value={commentText}
          onChange={(event) => onChangeText(event.target.value)}
          placeholder="Добавить комментарий"
          rows={3}
          className="min-h-[7rem] flex-1 resize-none rounded-lg border border-white/10 bg-white/5 px-5 py-4 text-lg text-slate-100 outline-none transition placeholder:text-slate-400 focus:border-white/25"
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
        <ClientHeader title={pageTitle} onBack={() => navigate(-1)} />

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
