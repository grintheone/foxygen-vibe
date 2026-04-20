import { PageShell } from "./page-shell";

export function RouteLoader({ message }) {
  return (
    <PageShell>
      <section className="w-full max-w-xl rounded-[2rem] border border-white/10 bg-white/10 p-8 text-center shadow-2xl shadow-[#6A3BF2]/25 backdrop-blur-xl">
        <p className="text-xs font-semibold uppercase tracking-[0.45em] text-[#6A3BF2]">
          Mobile Engineer
        </p>
        <p className="mt-6 text-sm uppercase tracking-[0.3em] text-slate-300">
          {message}
        </p>
      </section>
    </PageShell>
  );
}
