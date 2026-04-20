export function PageShell({ children }) {
  return (
    <main className="min-h-[100dvh] bg-gradient-to-br from-slate-950 via-slate-900 to-[#6A3BF2] px-6 pb-[calc(3rem+env(safe-area-inset-bottom))] pt-[calc(3rem+env(safe-area-inset-top))] text-slate-100">
      <div className="mx-auto flex min-h-[calc(100dvh-6rem-env(safe-area-inset-top)-env(safe-area-inset-bottom))] max-w-6xl items-center justify-center">
        {children}
      </div>
    </main>
  );
}
