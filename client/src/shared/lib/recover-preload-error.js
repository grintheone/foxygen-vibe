const retryKey = "foxygen-preload-retry";
const retryWindowMs = 60_000;

export function registerPreloadErrorRecovery() {
  window.addEventListener("vite:preloadError", (event) => {
    if (!navigator.onLine) {
      return;
    }

    try {
      const lastRetry = Number(window.sessionStorage.getItem(retryKey));
      const now = Date.now();

      if (lastRetry && now - lastRetry < retryWindowMs) {
        return;
      }

      // Survive the reload so a persistent outage cannot cause a reload loop.
      window.sessionStorage.setItem(retryKey, String(now));
    } catch {
      // Without persistent tracking we cannot safely reload automatically.
      return;
    }

    event.preventDefault();
    window.location.reload();
  });
}
