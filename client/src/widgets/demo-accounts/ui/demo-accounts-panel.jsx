export function DemoAccountsPanel({ accounts, onSelect }) {
  return (
    <aside className="rounded-[2rem] border border-white/10 bg-white/10 p-8 shadow-2xl shadow-cyan-950/50 backdrop-blur-xl">
      <p className="text-xs font-semibold uppercase tracking-[0.35em] text-slate-400">
        Demo Accounts
      </p>
      <div className="mt-6 space-y-4">
        {accounts.map((account) => (
          <button
            key={account.username}
            type="button"
            onClick={() => onSelect(account)}
            className="block w-full rounded-3xl border border-white/10 bg-slate-950/30 p-5 text-left transition hover:border-cyan-300/40 hover:bg-slate-950/40"
          >
            <p className="text-xs font-semibold uppercase tracking-[0.3em] text-cyan-300">
              {account.title}
            </p>
            <p className="mt-3 text-lg font-semibold text-slate-50">
              {account.username}
            </p>
            <p className="mt-1 text-sm text-slate-300">
              Password: {account.password}
            </p>
          </button>
        ))}
      </div>
    </aside>
  );
}
