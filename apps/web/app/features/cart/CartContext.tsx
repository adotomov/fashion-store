import { createContext, useContext, useEffect, useState, type ReactNode } from "react";

import { type Cart, addCartItem, getCart, removeCartItem, updateCartItemQuantity } from "../../lib/api/cart";
import { useAuth } from "../auth/AuthContext";

type CartContextValue = {
  cart: Cart | null;
  isLoading: boolean;
  itemCount: number;
  addItem: (variantId: string, quantity?: number) => Promise<void>;
  updateQuantity: (itemId: string, quantity: number) => Promise<void>;
  removeItem: (itemId: string) => Promise<void>;
  refresh: () => Promise<void>;
};

const CartContext = createContext<CartContextValue | undefined>(undefined);

export function CartProvider({ children }: { children: ReactNode }) {
  const { profile } = useAuth();
  const [cart, setCart] = useState<Cart | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  async function refresh() {
    try {
      setCart(await getCart());
    } catch {
      // leave previous cart state on a transient fetch failure
    } finally {
      setIsLoading(false);
    }
  }

  // Re-fetch whenever the signed-in identity changes (login, logout, or
  // initial load) — login already merged the guest cart by this point, so
  // this just picks up whichever cart now belongs to the current caller.
  useEffect(() => {
    refresh();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [profile?.id]);

  async function addItem(variantId: string, quantity = 1) {
    setCart(await addCartItem(variantId, quantity));
  }

  async function updateQuantity(itemId: string, quantity: number) {
    setCart(await updateCartItemQuantity(itemId, quantity));
  }

  async function removeItem(itemId: string) {
    setCart(await removeCartItem(itemId));
  }

  return (
    <CartContext.Provider
      value={{
        cart,
        isLoading,
        itemCount: cart?.item_count ?? 0,
        addItem,
        updateQuantity,
        removeItem,
        refresh,
      }}
    >
      {children}
    </CartContext.Provider>
  );
}

export function useCart(): CartContextValue {
  const ctx = useContext(CartContext);
  if (!ctx) {
    throw new Error("useCart must be used within a CartProvider");
  }
  return ctx;
}
