export function StatusMessage({ feedback }) {
  return (
    <div className="rounded-3xl border border-white/10 bg-white/10 p-5 backdrop-blur-xl">
      <p
        className={`text-sm ${
          feedback.tone === "error" ? "text-rose-300" : "text-emerald-300"
        }`}
      >
        {feedback.message}
      </p>
    </div>
  );
}
