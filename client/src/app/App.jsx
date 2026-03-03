import { StoreProvider } from "./providers/store-provider";
import { AppRouter } from "./providers/router";
import { AuthBootstrap } from "../features/auth";

export default function App() {
  return (
    <StoreProvider>
      <AuthBootstrap>
        <AppRouter />
      </AuthBootstrap>
    </StoreProvider>
  );
}
