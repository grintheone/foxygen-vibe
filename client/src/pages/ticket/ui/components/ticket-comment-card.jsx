import { UserAvatar } from "../../../../shared/ui/user-avatar";

export function TicketCommentCard({ authorAvatarUrl, authorDepartment, authorId, authorName, createdAt, text }) {
    return (
        <article className="rounded-lg border border-white/20 bg-transparent px-4 py-4">
            <p className="text-[16px] leading-relaxed text-slate-100">{text || "—"}</p>

            <div className="mt-4 flex items-end justify-between gap-4">
                <div className="flex min-w-0 items-center gap-4">
                    <UserAvatar
                        avatarUrl={authorAvatarUrl}
                        userId={authorId}
                        name={authorName}
                        className="h-10 w-10 shrink-0"
                        iconClassName="h-5 w-5"
                    />
                    <div className="min-w-0">
                        <p className="truncate text-[16px] font-semibold leading-snug tracking-tight text-slate-50">
                            {authorName || "Не указано"}
                        </p>
                        <p className="mt-1 truncate text-[14px] leading-snug text-slate-200/70">
                            {authorDepartment || "Отдел не указан"}
                        </p>
                    </div>
                </div>

                <p className="shrink-0 text-[14px] text-slate-400">{createdAt}</p>
            </div>
        </article>
    );
}
