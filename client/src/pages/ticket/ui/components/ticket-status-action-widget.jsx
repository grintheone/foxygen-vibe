import { BottomPageAction } from "../../../../shared/ui/bottom-page-action";

export function TicketStatusActionWidget({
  actionState,
  errorMessage,
  isLoading,
  onSubmit,
}) {
  if (!actionState?.isVisible) {
    return null;
  }

  return (
    <BottomPageAction
      buttonClassName={actionState.colorClassName}
      disabled={!actionState.isEnabled || isLoading}
      errorMessage={errorMessage}
      onClick={onSubmit}
    >
      {isLoading ? (
        "Сохраняем..."
      ) : (
        <>
          {actionState.hasSuccessIcon ? (
            <span className="mr-2 inline-flex h-6 w-6 items-center justify-center rounded-full bg-white/20">
              <svg
                xmlns="http://www.w3.org/2000/svg"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2.5"
                strokeLinecap="round"
                strokeLinejoin="round"
                className="h-4 w-4"
                aria-hidden="true"
              >
                <path d="M20 6 9 17l-5-5" />
              </svg>
            </span>
          ) : null}
          <span>{actionState.title}</span>
        </>
      )}
    </BottomPageAction>
  );
}
