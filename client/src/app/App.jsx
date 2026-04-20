import { StoreProvider } from "./providers/store-provider";
import { AppRouter } from "./providers/router";
import { AuthBootstrap } from "../features/auth";
import { PwaInstallPrompt } from "../features/pwa";

export default function App() {
  return (
    <StoreProvider>
      <AuthBootstrap>
        <AppRouter />
        <PwaInstallPrompt />
      </AuthBootstrap>
    </StoreProvider>
  );
}
