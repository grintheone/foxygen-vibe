import { useEffect, useState } from "react";
import { Link } from "react-router";
import { routePaths } from "../config/routes";

function PersonIcon({ className }) {
    return (
        <svg
            xmlns="http://www.w3.org/2000/svg"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="1.8"
            className={className}
            aria-hidden="true"
        >
            <circle cx="12" cy="8" r="3.6" />
            <path d="M4.5 19.2C5.9 15.9 8.6 14.4 12 14.4s6.1 1.5 7.5 4.8" />
        </svg>
    );
}

function mergeClassNames(...values) {
    return values.filter(Boolean).join(" ");
}

function stopParentInteraction(event) {
    event.stopPropagation();
}

export function UserAvatar({
    avatarUrl,
    userId,
    name,
    className = "h-10 w-10",
    fallbackClassName = "bg-slate-100 text-slate-500",
    iconClassName = "h-5 w-5",
    imageClassName = "",
    stopPropagation = false,
}) {
    const normalizedAvatarUrl = typeof avatarUrl === "string" ? avatarUrl.trim() : "";
    const normalizedUserId = typeof userId === "string" ? userId.trim() : "";
    const label = name?.trim() || "Профиль пользователя";
    const rootClassName = mergeClassNames("inline-flex shrink-0 overflow-hidden rounded-full", className);
    const [hasImageError, setHasImageError] = useState(false);

    useEffect(() => {
        setHasImageError(false);
    }, [normalizedAvatarUrl]);

    const shouldRenderImage = normalizedAvatarUrl && !hasImageError;
    const content = shouldRenderImage ? (
        <img
            src={normalizedAvatarUrl}
            alt={name || "Аватар пользователя"}
            loading="lazy"
            decoding="async"
            onError={() => setHasImageError(true)}
            className={mergeClassNames("h-full w-full rounded-full object-cover", imageClassName)}
        />
    ) : (
        <span
            aria-hidden="true"
            className={mergeClassNames(
                "flex h-full w-full items-center justify-center rounded-full",
                fallbackClassName,
            )}
        >
            <PersonIcon className={iconClassName} />
        </span>
    );

    if (!normalizedUserId) {
        return <span className={rootClassName}>{content}</span>;
    }

    const interactionProps = stopPropagation
        ? {
              onClick: stopParentInteraction,
              onKeyDown: stopParentInteraction,
          }
        : {};

    return (
        <Link
            to={routePaths.profileById(normalizedUserId)}
            aria-label={`Открыть профиль: ${label}`}
            className={mergeClassNames(
                rootClassName,
                "transition hover:opacity-90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-white/60",
            )}
            {...interactionProps}
        >
            {content}
        </Link>
    );
}
