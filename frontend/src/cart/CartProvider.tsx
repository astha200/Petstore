import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from "react";
import type { Pet } from "../types";

interface CartContextValue {
  items: Pet[];
  add: (pet: Pet) => void;
  remove: (petId: string) => void;
  clear: () => void;
  has: (petId: string) => boolean;
  count: number;
}

const CartContext = createContext<CartContextValue | undefined>(undefined);

export function CartProvider({ children }: { children: ReactNode }) {
  const [items, setItems] = useState<Pet[]>([]);

  const add = useCallback((pet: Pet) => {
    setItems((prev) => (prev.some((p) => p.id === pet.id) ? prev : [...prev, pet]));
  }, []);

  const remove = useCallback((petId: string) => {
    setItems((prev) => prev.filter((p) => p.id !== petId));
  }, []);

  const clear = useCallback(() => setItems([]), []);

  const has = useCallback((petId: string) => items.some((p) => p.id === petId), [items]);

  const value = useMemo<CartContextValue>(
    () => ({ items, add, remove, clear, has, count: items.length }),
    [items, add, remove, clear, has],
  );

  return <CartContext.Provider value={value}>{children}</CartContext.Provider>;
}

export function useCart(): CartContextValue {
  const ctx = useContext(CartContext);
  if (!ctx) throw new Error("useCart must be used within a CartProvider");
  return ctx;
}
