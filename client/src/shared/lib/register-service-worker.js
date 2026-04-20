const updateIntervalMs = 1000 * 60 * 5;

export function registerServiceWorker() {
  if (!import.meta.env.PROD || !("serviceWorker" in navigator)) {
    return;
  }

  window.addEventListener("load", () => {
    setupServiceWorker().catch((error) => {
      console.error("Failed to register the service worker.", error);
    });
  });
}

async function setupServiceWorker() {
  let hasReloadedForUpdate = false;

  const registration = await navigator.serviceWorker.register("/sw.js");

  navigator.serviceWorker.addEventListener("controllerchange", () => {
    if (hasReloadedForUpdate) {
      return;
    }

    hasReloadedForUpdate = true;
    window.location.reload();
  });

  watchForServiceWorkerUpdates(registration);
  activateWaitingWorker(registration);
  scheduleServiceWorkerUpdates(registration);

  await registration.update();
}

function watchForServiceWorkerUpdates(registration) {
  registration.addEventListener("updatefound", () => {
    const installingWorker = registration.installing;

    if (!installingWorker) {
      return;
    }

    installingWorker.addEventListener("statechange", () => {
      if (installingWorker.state !== "installed" || !navigator.serviceWorker.controller) {
        return;
      }

      window.setTimeout(() => {
        activateWaitingWorker(registration);
      }, 0);
    });
  });
}

function scheduleServiceWorkerUpdates(registration) {
  window.setInterval(() => {
    registration.update().catch((error) => {
      console.error("Failed to refresh the service worker.", error);
    });
  }, updateIntervalMs);

  window.addEventListener("focus", () => {
    registration.update().catch((error) => {
      console.error("Failed to refresh the service worker.", error);
    });
  });

  document.addEventListener("visibilitychange", () => {
    if (document.visibilityState !== "visible") {
      return;
    }

    registration.update().catch((error) => {
      console.error("Failed to refresh the service worker.", error);
    });
  });
}

function activateWaitingWorker(registration) {
  if (!registration.waiting) {
    return;
  }

  registration.waiting.postMessage({
    type: "SKIP_WAITING",
  });
}
