const STATIC_CACHE = "foxygen-static-v4";
const RUNTIME_CACHE = "foxygen-runtime-v4";
const APP_SHELL = [
  "/",
  "/index.html",
  "/offline.html",
  "/site.webmanifest",
  "/favicon.svg",
  "/apple-touch-icon.png",
  "/me-192.png",
  "/me-512.png",
];

self.addEventListener("install", (event) => {
  event.waitUntil(
    caches.open(STATIC_CACHE).then((cache) => cache.addAll(APP_SHELL)),
  );
  self.skipWaiting();
});

self.addEventListener("activate", (event) => {
  event.waitUntil(
    caches
      .keys()
      .then((keys) =>
        Promise.all(
          keys
            .filter((key) => key.startsWith("foxygen-") && key !== STATIC_CACHE && key !== RUNTIME_CACHE)
            .map((key) => caches.delete(key)),
        ),
      )
      .then(() => self.clients.claim()),
  );
});

self.addEventListener("message", (event) => {
  if (event.data?.type === "SKIP_WAITING") {
    self.skipWaiting();
  }
});

self.addEventListener("fetch", (event) => {
  const { request } = event;

  if (request.method !== "GET") {
    return;
  }

  const url = new URL(request.url);

  if (url.origin !== self.location.origin || url.pathname.startsWith("/api/")) {
    return;
  }

  if (request.mode === "navigate") {
    event.respondWith(handleNavigationRequest(request));
    return;
  }

  if (isStaticAssetRequest(request, url)) {
    event.respondWith(handleStaticAssetRequest(request, url));
  }
});

async function handleNavigationRequest(request) {
  try {
    const response = await fetch(request, { cache: "no-cache" });

    await cacheResponse(request, response);

    return response;
  } catch (error) {
    const cachedResponse = await caches.match(request);

    if (cachedResponse) {
      return cachedResponse;
    }

    const appShell = await caches.match("/index.html");

    if (appShell) {
      return appShell;
    }

    return caches.match("/offline.html");
  }
}

async function handleStaticAssetRequest(request, url) {
  // Vite fingerprints these URLs: cached chunks remain valid across deploys.
  // An open tab may still need a chunk removed from the current server image.
  const cachedResponse = await caches.match(request);

  if (cachedResponse && (
    url.pathname.startsWith("/assets/") ||
    request.destination === "image" || request.destination === "font"
  )) {
    return cachedResponse;
  }

  return fetchAndCache(request);
}

async function fetchAndCache(request) {
  try {
    const response = await fetch(request);

    if (!response.ok) {
      const cachedResponse = await caches.match(request);

      if (cachedResponse) {
        return cachedResponse;
      }
    }

    await cacheResponse(request, response);

    return response;
  } catch (error) {
    const cachedResponse = await caches.match(request);

    if (cachedResponse) {
      return cachedResponse;
    }

    throw error;
  }
}

function isStaticAssetRequest(request, url) {
  return (
    request.destination === "script" ||
    request.destination === "style" ||
    request.destination === "image" ||
    request.destination === "font" ||
    url.pathname.startsWith("/assets/")
  );
}

async function cacheResponse(request, response) {
  if (!response.ok) {
    return;
  }

  try {
    const cache = await caches.open(RUNTIME_CACHE);
    await cache.put(request, response.clone());
  } catch {
    // Storage exhaustion or disabled caching must not discard a network response.
  }
}
