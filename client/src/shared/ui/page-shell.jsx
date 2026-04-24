export function PageShell({ children }) {
  return (
    <main className="min-h-[100dvh] bg-gradient-to-br from-slate-950 via-slate-900 to-[#6A3BF2] px-3 pb-[calc(1.5rem+env(safe-area-inset-bottom))] pt-[calc(1rem+env(safe-area-inset-top))] text-slate-100 sm:px-4 sm:pb-[calc(2rem+env(safe-area-inset-bottom))] sm:pt-[calc(2rem+env(safe-area-inset-top))] lg:px-6 lg:pb-[calc(3rem+env(safe-area-inset-bottom))] lg:pt-[calc(3rem+env(safe-area-inset-top))]">
      <div className="mx-auto flex min-h-[calc(100dvh-2.5rem-env(safe-area-inset-top)-env(safe-area-inset-bottom))] w-full max-w-6xl flex-col items-center justify-start sm:min-h-[calc(100dvh-4rem-env(safe-area-inset-top)-env(safe-area-inset-bottom))] lg:min-h-[calc(100dvh-6rem-env(safe-area-inset-top)-env(safe-area-inset-bottom))]">
        {children}
      </div>
    </main>
  );
}
