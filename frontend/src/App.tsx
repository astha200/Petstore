import { useAuth } from "./auth/AuthProvider";
import { Login } from "./auth/Login";
import { Storefront } from "./Storefront";

function getStoreSlug(): string {
  const params = new URLSearchParams(window.location.search);
  return params.get("store") ?? "PetVerse";
}

export function App() {
  const { credentials } = useAuth();
  const storeSlug = getStoreSlug();

  if (!credentials) return <Login storeSlug={storeSlug} />;
  return <Storefront storeSlug={storeSlug} />;
}
