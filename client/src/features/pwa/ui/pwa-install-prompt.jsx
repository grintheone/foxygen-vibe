import { useEffect, useRef, useState } from "react";

const dismissalStorageKey = "foxygen.pwa_install_prompt.dismissed_at";
const dismissalTtlMs = 1000 * 60 * 60 * 24 * 3;

function isStandaloneMode() {
  return (
    window.matchMedia("(display-mode: standalone)").matches ||
    window.navigator.standalone === true
  );
}

function isMobileDevice() {
  const hasTouch = window.matchMedia("(pointer: coarse)").matches || navigator.maxTouchPoints > 0;
  const compactViewport = window.matchMedia("(max-width: 1024px)").matches;

  return hasTouch && compactViewport;
}

function isIosDevice() {
  const userAgent = window.navigator.userAgent;
  const platform = window.navigator.platform;
  const touchMac = platform === "MacIntel" && navigator.maxTouchPoints > 1;

  return /iPad|iPhone|iPod/.test(userAgent) || touchMac;
}

function wasDismissedRecently() {
  const rawValue = window.localStorage.getItem(dismissalStorageKey);

  if (!rawValue) {
    return false;
  }

  const dismissedAt = Number(rawValue);

  if (!Number.isFinite(dismissedAt)) {
    window.localStorage.removeItem(dismissalStorageKey);
    return false;
  }

  return Date.now() - dismissedAt < dismissalTtlMs;
}

function rememberDismissal() {
  window.localStorage.setItem(dismissalStorageKey, String(Date.now()));
}

function clearDismissal() {
  window.localStorage.removeItem(dismissalStorageKey);
}

export function PwaInstallPrompt() {
  const deferredPromptRef = useRef(null);
  const [mode, setMode] = useState(null);

  useEffect(() => {
    if (typeof window === "undefined") {
      return undefined;
    }

    let revealTimerId = null;

    function syncPromptVisibility() {
      if (isStandaloneMode() || !isMobileDevice() || wasDismissedRecently()) {
        setMode(null);
        return;
      }

      if (deferredPromptRef.current) {
        setMode("native");
        return;
      }

      if (isIosDevice()) {
        setMode("ios");
        return;
      }

      setMode(null);
    }

    function handleBeforeInstallPrompt(event) {
      event.preventDefault();
      deferredPromptRef.current = event;
      syncPromptVisibility();
    }

    function handleAppInstalled() {
      deferredPromptRef.current = null;
      clearDismissal();
      setMode(null);
    }

    revealTimerId = window.setTimeout(syncPromptVisibility, 1800);

    window.addEventListener("beforeinstallprompt", handleBeforeInstallPrompt);
    window.addEventListener("appinstalled", handleAppInstalled);

    return () => {
      if (revealTimerId) {
        window.clearTimeout(revealTimerId);
      }

      window.removeEventListener("beforeinstallprompt", handleBeforeInstallPrompt);
      window.removeEventListener("appinstalled", handleAppInstalled);
    };
  }, []);

  function handleClose() {
    rememberDismissal();
    setMode(null);
  }

  async function handleInstall() {
    const promptEvent = deferredPromptRef.current;

    if (!promptEvent) {
      return;
    }

    deferredPromptRef.current = null;

    try {
      await promptEvent.prompt();
      const userChoice = await promptEvent.userChoice;

      if (userChoice?.outcome === "accepted") {
        clearDismissal();
        setMode(null);
        return;
      }
    } catch (error) {
      console.error("Failed to show the install prompt.", error);
    }

    rememberDismissal();
    setMode(null);
  }

  if (!mode) {
    return null;
  }

  const body =
    mode === "native"
      ? "Установите приложение, чтобы открывать сервис с главного экрана телефона и работать в полноэкранном режиме."
      : "Чтобы установить приложение на iPhone или iPad, откройте меню «Поделиться» и выберите пункт «На экран Домой».";

  const hint =
    mode === "native"
      ? "Это займет пару секунд."
      : "После этого сервис будет запускаться как отдельное приложение.";

  return (
    <div className="pointer-events-none fixed inset-x-4 top-[calc(1rem+env(safe-area-inset-top))] z-50 mx-auto w-full max-w-sm">
      <section className="pointer-events-auto rounded-[1.75rem] border border-white/15 bg-slate-950/88 p-4 shadow-2xl shadow-black/35 backdrop-blur-xl">
        <div className="flex items-start gap-3">
          <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl bg-[#6A3BF2]/20 text-[#C9B8FF]">
            <svg
              aria-hidden="true"
              xmlns="http://www.w3.org/2000/svg"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="1.8"
              strokeLinecap="round"
              strokeLinejoin="round"
              className="h-5 w-5"
            >
              <path d="M12 3v10" />
              <path d="m8.5 6.5 3.5-3.5 3.5 3.5" />
              <path d="M5 14.5v1.5A2 2 0 0 0 7 18h10a2 2 0 0 0 2-2v-1.5" />
            </svg>
          </div>

          <div className="min-w-0 flex-1">
            <div className="flex items-start justify-between gap-3">
              <div>
                <p className="text-[11px] font-semibold uppercase tracking-[0.22em] text-slate-400">
                  Установка приложения
                </p>
                <h2 className="mt-1 text-base font-semibold text-slate-50">
                  Добавьте сервис на главный экран
                </h2>
              </div>

              <button
                type="button"
                onClick={handleClose}
                aria-label="Закрыть рекомендацию по установке"
                className="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-2xl bg-white/5 text-slate-300 transition hover:bg-white/10 hover:text-white"
              >
                <svg
                  aria-hidden="true"
                  xmlns="http://www.w3.org/2000/svg"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  className="h-4 w-4"
                >
                  <path d="M18 6 6 18" />
                  <path d="m6 6 12 12" />
                </svg>
              </button>
            </div>

            <p className="mt-3 text-sm leading-6 text-slate-200">{body}</p>
            <p className="mt-2 text-xs leading-5 text-slate-400">{hint}</p>

            <div className="mt-4 flex flex-wrap gap-2">
              {mode === "native" ? (
                <button
                  type="button"
                  onClick={handleInstall}
                  className="inline-flex min-h-11 items-center justify-center rounded-2xl bg-[#6A3BF2] px-4 text-sm font-semibold text-white transition hover:bg-[#7C52F5]"
                >
                  Установить
                </button>
              ) : null}

              <button
                type="button"
                onClick={handleClose}
                className="inline-flex min-h-11 items-center justify-center rounded-2xl bg-white/8 px-4 text-sm font-semibold text-slate-100 transition hover:bg-white/12"
              >
                {mode === "native" ? "Позже" : "Понятно"}
              </button>
            </div>
          </div>
        </div>
      </section>
    </div>
  );
}
