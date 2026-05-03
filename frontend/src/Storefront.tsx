import { useMemo, useState } from "react";
import { useMutation, useQuery, type ApolloError } from "@apollo/client";
import { useAuth } from "./auth/AuthProvider";
import { CartProvider, useCart } from "./cart/CartProvider";
import { PetCard } from "./components/PetCard";
import { PetCardSkeletonGrid } from "./components/PetCardSkeleton";
import { CartDrawer } from "./components/CartDrawer";
import { ConfirmDialog } from "./components/ConfirmDialog";
import { useToast } from "./components/ToastProvider";
import { AVAILABLE_PETS, CHECKOUT, PURCHASE_PET } from "./graphql/operations";
import type { Pet } from "./types";

interface Props {
  storeSlug: string;
}

function humanizeError(err: unknown): string {
  if (!err) return "Something went wrong.";
  const apollo = err as ApolloError;
  if (apollo.graphQLErrors?.length) return apollo.graphQLErrors[0]!.message;
  if (apollo.networkError) {
    if ("statusCode" in apollo.networkError && apollo.networkError.statusCode === 401) {
      return "You are not signed in or your credentials are invalid.";
    }
    return "Could not reach the server. Please try again.";
  }
  if (apollo.message) return apollo.message;
  return "Something went wrong.";
}

export function Storefront({ storeSlug }: Props) {
  return (
    <CartProvider>
      <StorefrontInner storeSlug={storeSlug} />
    </CartProvider>
  );
}

function StorefrontInner({ storeSlug }: Props) {
  const { credentials, signOut } = useAuth();
  const cart = useCart();
  const toast = useToast();
  const [cartOpen, setCartOpen] = useState(false);
  const [busyPetId, setBusyPetId] = useState<string | null>(null);
  const [pendingBuy, setPendingBuy] = useState<Pet | null>(null);
  const [pendingCheckout, setPendingCheckout] = useState(false);

  const { data, loading, error, refetch } = useQuery<{ availablePets: Pet[] }>(AVAILABLE_PETS, {
    variables: { store: storeSlug },
  });

  const [purchasePet, { loading: buyingOne }] = useMutation(PURCHASE_PET);
  const [checkout, { loading: checkingOut }] = useMutation(CHECKOUT);

  const availableIds = useMemo(
    () => new Set((data?.availablePets ?? []).map((p) => p.id)),
    [data],
  );

  const doBuy = async (pet: Pet) => {
    setBusyPetId(pet.id);
    try {
      await purchasePet({ variables: { store: storeSlug, petId: pet.id } });
      cart.remove(pet.id);
      toast.success(`You purchased ${pet.name}!`);
      await refetch();
    } catch (err) {
      toast.error(humanizeError(err));
      // List may be stale if our buy failed because someone else got it.
      await refetch();
    } finally {
      setBusyPetId(null);
      setPendingBuy(null);
    }
  };

  const doCheckout = async () => {
    if (cart.items.length === 0) return;
    setPendingCheckout(false);
    try {
      const result = await checkout({
        variables: { store: storeSlug, petIds: cart.items.map((p) => p.id) },
      });
      const purchased: Pet[] = result.data?.checkout.purchased ?? [];
      const names = purchased.map((p) => p.name).join(", ");
      cart.clear();
      setCartOpen(false);
      toast.success(
        `Purchased ${purchased.length} pet${purchased.length === 1 ? "" : "s"}: ${names}.`,
      );
      await refetch();
    } catch (err) {
      toast.error(humanizeError(err));
      // Prune now-unavailable items from the cart so the next checkout can succeed.
      const refreshed = await refetch();
      const stillAvailable = new Set(
        (refreshed.data?.availablePets ?? []).map((p) => p.id),
      );
      for (const item of cart.items) {
        if (!stillAvailable.has(item.id)) cart.remove(item.id);
      }
    }
  };

  const pets = data?.availablePets ?? [];
  const initialLoading = loading && pets.length === 0;

  return (
    <div className="app">
      <header className="header">
        <div className="header__brand">🐾 {storeSlug}</div>
        <div className="header__right">
          <span>Hi, {credentials?.username}</span>
          <button
            className="btn btn--secondary header__cart-btn"
            onClick={() => setCartOpen(true)}
          >
            Cart
            {cart.count > 0 && <span className="header__cart-badge">{cart.count}</span>}
          </button>
          <button className="btn btn--ghost" onClick={signOut}>Sign out</button>
        </div>
      </header>

      <main className="main">
        <h1 className="title">Pets available now</h1>
        <p className="subtitle">Refresh the page to see the latest listings.</p>

        {error && !data && (
          <div className="empty">{humanizeError(error)}</div>
        )}

        <div className="grid">
          {initialLoading && <PetCardSkeletonGrid count={6} />}

          {!initialLoading &&
            pets.map((pet) => (
              <PetCard
                key={pet.id}
                pet={pet}
                inCart={cart.has(pet.id)}
                busy={busyPetId === pet.id || buyingOne || checkingOut}
                onBuyNow={() => setPendingBuy(pet)}
                onAddToCart={() => {
                  cart.add(pet);
                  toast.success(`Added ${pet.name} to your cart.`);
                }}
                onRemoveFromCart={() => cart.remove(pet.id)}
              />
            ))}
        </div>

        {!loading && pets.length === 0 && !error && (
          <div className="empty">No pets are available right now. Check back soon!</div>
        )}
      </main>

      <CartDrawer
        open={cartOpen}
        busy={checkingOut}
        onClose={() => setCartOpen(false)}
        onCheckout={() => setPendingCheckout(true)}
      />

      <ConfirmDialog
        open={pendingBuy !== null}
        title="Confirm purchase"
        message={
          pendingBuy
            ? `Buy ${pendingBuy.name} now? This will take it off the market immediately.`
            : ""
        }
        confirmLabel="Yes, buy now"
        busy={buyingOne}
        onCancel={() => setPendingBuy(null)}
        onConfirm={() => pendingBuy && doBuy(pendingBuy)}
      />

      <ConfirmDialog
        open={pendingCheckout}
        title="Confirm checkout"
        message={`Purchase ${cart.count} pet${cart.count === 1 ? "" : "s"} from your cart?`}
        confirmLabel="Yes, check out"
        busy={checkingOut}
        onCancel={() => setPendingCheckout(false)}
        onConfirm={doCheckout}
      />

      <StalenessSweeper availableIds={availableIds} />
    </div>
  );
}

function StalenessSweeper({ availableIds }: { availableIds: Set<string> }) {
  const cart = useCart();
  // After every refetch, drop any cart items the server says are no longer for sale.
  useMemo(() => {
    for (const item of cart.items) {
      if (!availableIds.has(item.id)) cart.remove(item.id);
    }
    // Intentionally only react to availableIds changes.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [availableIds]);
  return null;
}
