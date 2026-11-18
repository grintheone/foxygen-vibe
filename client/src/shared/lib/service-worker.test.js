import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import vm from "node:vm";
import { registerPreloadErrorRecovery } from "./recover-preload-error.js";

const workerSource = readFileSync(new URL("../../../public/sw.js", import.meta.url), "utf8");

function worker({ cached, fetch, put = async () => {} }) {
  const listeners = {};
  const context = vm.createContext({
    URL,
    self: {
      location: { origin: "https://app.example" },
      addEventListener: (name, callback) => { listeners[name] = callback; },
    },
    caches: {
      match: async () => cached,
      open: async () => ({ put }),
    },
    fetch,
  });
  vm.runInContext(workerSource, context);
  return (path, mode = "cors") => {
    let response;
    listeners.fetch({
      request: { url: `https://app.example${path}`, method: "GET", mode, destination: "script" },
      respondWith: (promise) => { response = promise; },
    });
    return response;
  };
}

test("an old tab can load a cached fingerprinted chunk without the server", async () => {
  const cached = new Response("old chunk");
  const request = worker({ cached, fetch: () => { assert.fail("must use cached chunk"); } });
  assert.equal(await request("/assets/old.js"), cached);
});

test("a cache write failure does not discard successful network responses", async () => {
  for (const [path, mode] of [["/assets/new.js", "cors"], ["/tickets", "navigate"]]) {
    const response = new Response("network response");
    const request = worker({
      fetch: async (_request, options) => {
        if (mode === "navigate") assert.equal(options.cache, "no-cache");
        return response;
      },
      put: async () => { throw new Error("quota exceeded"); },
    });
    assert.equal(await request(path, mode), response);
  }
});

test("a non-fingerprinted asset falls back to cache on HTTP and network errors", async () => {
  const cached = new Response("cached script");
  for (const fetch of [async () => new Response("missing", { status: 404 }), async () => { throw new TypeError("offline"); }]) {
    assert.equal(await worker({ cached, fetch })("/script.js"), cached);
  }
});

test("preload recovery survives reloads, skips offline and storage failures", () => {
  const originalWindow = Object.getOwnPropertyDescriptor(globalThis, "window");
  const originalNavigator = Object.getOwnPropertyDescriptor(globalThis, "navigator");
  let handler;
  let reloads = 0;
  const storage = new Map();
  const browser = {
    addEventListener: (_name, callback) => { handler = callback; },
    location: { reload: () => { reloads++; } },
    sessionStorage: { getItem: (key) => storage.get(key), setItem: (key, value) => storage.set(key, value) },
  };
  const navigator = { onLine: true };
  Object.defineProperty(globalThis, "window", { configurable: true, value: browser });
  Object.defineProperty(globalThis, "navigator", { configurable: true, value: navigator });
  try {
    let prevented = 0;
    const event = { preventDefault: () => { prevented++; } };
    registerPreloadErrorRecovery();
    handler(event);
    registerPreloadErrorRecovery();
    handler(event);
    assert.equal(reloads, 1);
    assert.equal(prevented, 1);
    storage.clear();
    navigator.onLine = false;
    handler(event);
    navigator.onLine = true;
    browser.sessionStorage.getItem = () => { throw new Error("storage disabled"); };
    handler(event);
    assert.equal(reloads, 1);
  } finally {
    if (originalWindow) Object.defineProperty(globalThis, "window", originalWindow);
    else delete globalThis.window;
    if (originalNavigator) Object.defineProperty(globalThis, "navigator", originalNavigator);
    else delete globalThis.navigator;
  }
});
