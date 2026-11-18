import React from "react";
import ReactDOM from "react-dom/client";
import App from "./app/App";
import "./app/styles/index.css";
import { registerServiceWorker } from "./shared/lib/register-service-worker";
import { registerPreloadErrorRecovery } from "./shared/lib/recover-preload-error";

registerPreloadErrorRecovery();

ReactDOM.createRoot(document.getElementById("root")).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
);

registerServiceWorker();
